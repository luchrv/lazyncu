// Package orchestrator fans out version scans and vulnerability audits —
// one goroutine per source, one goroutine per repository inside a folder
// source, one audit goroutine per discovered project — and streams
// immutable per-source snapshot events on a channel as work completes.
package orchestrator

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/scanner"
)

// SourceGlobal identifies the global-packages source in events.
const SourceGlobal = "global"

// Scanner is the subset of scanner.Scanner the orchestrator needs (injected
// so tests control timing and results).
type Scanner interface {
	ScanGlobal(ctx context.Context) ([]scanner.Package, error)
	ScanPath(ctx context.Context, dir string) ([]scanner.Project, error)
}

// Auditor audits one project directory (production: audit.Run over a Runner).
type Auditor func(ctx context.Context, dir string, pm detect.PackageManager) audit.Result

// Discoverer classifies a registered path: for a folder of repositories it
// returns the discovered repositories and isFolder = true; for a single
// project or monorepo it returns isFolder = false. A nil Discoverer treats
// every path as a non-folder.
type Discoverer func(dir string) (repos []detect.Repo, isFolder bool, err error)

// DiscoverRepos is the production Discoverer: detect.ScanMode decides, and
// detect.Repos walks folder-mode paths.
func DiscoverRepos(dir string) ([]detect.Repo, bool, error) {
	if detect.ScanMode(dir) != detect.ModeFolder {
		return nil, false, nil
	}
	repos, err := detect.Repos(dir)
	return repos, true, err
}

// Deps bundles the injected collaborators every scan needs.
type Deps struct {
	Scanner    Scanner
	Auditor    Auditor
	Discoverer Discoverer
}

// ProjectResult pairs one project's scan result with its audit outcome.
type ProjectResult struct {
	scanner.Project
	Audit audit.Result
	// Pending marks a folder entry whose repository has not finished
	// scanning yet; only Dir and Label are meaningful.
	Pending bool
	// Err carries a repository's own scan failure inside a folder source;
	// it never marks the source as failed.
	Err error
}

// Event is one source's current state. Non-folder sources deliver exactly
// one event with Done set; a folder source delivers a snapshot after
// discovery (all entries Pending) and a fresh snapshot each time a
// repository completes, the last one with Done set. Snapshots are never
// mutated after delivery. A source-level failure sets Err and never
// affects sibling sources.
type Event struct {
	// Source is SourceGlobal or the registered path.
	Source string
	// Packages holds global-source results (Source == SourceGlobal).
	Packages    []scanner.Package
	GlobalAudit audit.Result
	// Projects holds path-source results, each with its audit.
	Projects []ProjectResult
	Err      error
	// Folder marks a source scanned as a folder of repositories, whose
	// entries can be rescanned one at a time.
	Folder bool
	// Done reports that no more events follow for this source.
	Done bool
}

// Run launches all scans concurrently and returns the event channel, which
// closes once every source has delivered its final event.
func Run(ctx context.Context, deps Deps, paths []string) <-chan Event {
	events := make(chan Event)
	emit := func(ev Event) { events <- ev }
	var wg sync.WaitGroup

	wg.Go(func() {
		emit(RunGlobal(ctx, deps.Scanner))
	})

	for _, path := range paths {
		wg.Go(func() {
			scanSource(ctx, deps, path, emit)
		})
	}

	go func() {
		wg.Wait()
		close(events)
	}()
	return events
}

// RunOne scans and audits a single path — used when a path is added at
// runtime or rescanned. The channel closes after the source's final event.
func RunOne(ctx context.Context, deps Deps, path string) <-chan Event {
	events := make(chan Event)
	go func() {
		scanSource(ctx, deps, path, func(ev Event) { events <- ev })
		close(events)
	}()
	return events
}

// RunGlobal scans the global source synchronously — used at launch (inside
// Run's fan-out) and for manual rescans of the global source.
func RunGlobal(ctx context.Context, sc Scanner) Event {
	pkgs, err := sc.ScanGlobal(ctx)
	return Event{
		Source:      SourceGlobal,
		Packages:    pkgs,
		GlobalAudit: audit.GlobalResult(),
		Err:         err,
		Done:        true,
	}
}

// ScanProject scans and audits one repository as a unit. A scan failure
// yields a single entry carrying Err; a repository that is a workspaces
// monorepo yields one entry per workspace, labeled under the repository.
func ScanProject(ctx context.Context, deps Deps, repo detect.Repo) []ProjectResult {
	projects, err := deps.Scanner.ScanPath(ctx, repo.Dir)
	if err != nil {
		return []ProjectResult{{Project: placeholderProject(repo), Err: err}}
	}
	if len(projects) == 0 {
		projects = []scanner.Project{{Dir: repo.Dir, Label: ".", PM: detect.PackageManagerFor(repo.Dir)}}
	}
	results := auditAll(ctx, deps.Auditor, projects)
	for i := range results {
		results[i].Label = joinLabel(repo.Label, results[i].Label)
	}
	return results
}

// scanSource routes one registered path: discovery failure → failed
// source; folder → per-repository streaming; otherwise one final event.
func scanSource(ctx context.Context, deps Deps, path string, emit func(Event)) {
	repos, isFolder, err := discover(deps, path)
	switch {
	case err != nil:
		emit(Event{Source: path, Err: err, Folder: true, Done: true})
	case isFolder:
		scanFolder(ctx, deps, path, repos, emit)
	default:
		emit(scanAndAuditPath(ctx, deps, path))
	}
}

func discover(deps Deps, path string) ([]detect.Repo, bool, error) {
	if deps.Discoverer == nil {
		return nil, false, nil
	}
	return deps.Discoverer(path)
}

// scanFolder announces the discovered repositories as pending entries, then
// scans them concurrently, emitting a fresh flattened snapshot per
// completion. Each repository owns one slot so a monorepo expanding into
// several entries never disturbs its siblings' bookkeeping.
func scanFolder(ctx context.Context, deps Deps, path string, repos []detect.Repo, emit func(Event)) {
	repos = slices.Clone(repos)
	slices.SortFunc(repos, func(a, b detect.Repo) int { return strings.Compare(a.Label, b.Label) })
	slots := make([][]ProjectResult, len(repos))
	for i, repo := range repos {
		slots[i] = []ProjectResult{{Project: placeholderProject(repo), Pending: true}}
	}
	emit(Event{Source: path, Projects: flatten(slots), Folder: true, Done: len(repos) == 0})
	if len(repos) == 0 {
		return
	}

	type slotResult struct {
		idx     int
		results []ProjectResult
	}
	completed := make(chan slotResult)
	for i, repo := range repos {
		go func() { completed <- slotResult{idx: i, results: ScanProject(ctx, deps, repo)} }()
	}
	for n := range len(repos) {
		r := <-completed
		slots[r.idx] = r.results
		emit(Event{Source: path, Projects: flatten(slots), Folder: true, Done: n == len(repos)-1})
	}
}

// scanAndAuditPath scans one single/monorepo path, then audits every
// discovered project concurrently. Audit failures degrade to per-project
// badges; only a scan failure marks the source as failed.
func scanAndAuditPath(ctx context.Context, deps Deps, path string) Event {
	projects, err := deps.Scanner.ScanPath(ctx, path)
	if err != nil {
		return Event{Source: path, Err: err, Done: true}
	}
	return Event{Source: path, Projects: auditAll(ctx, deps.Auditor, projects), Done: true}
}

// auditAll audits projects concurrently and pairs each with its result. A
// nil Auditor marks every audit as not available instead of failing.
func auditAll(ctx context.Context, auditor Auditor, projects []scanner.Project) []ProjectResult {
	if auditor == nil {
		auditor = func(context.Context, string, detect.PackageManager) audit.Result {
			return audit.Result{Status: audit.StatusNotAvailable}
		}
	}
	results := make([]ProjectResult, len(projects))
	var wg sync.WaitGroup
	for i, project := range projects {
		wg.Go(func() {
			results[i] = ProjectResult{
				Project: project,
				Audit:   auditor(ctx, project.Dir, project.PM),
			}
		})
	}
	wg.Wait()
	return results
}

// flatten builds a fresh snapshot from the per-repository slots.
func flatten(slots [][]ProjectResult) []ProjectResult {
	out := make([]ProjectResult, 0, len(slots))
	for _, slot := range slots {
		out = append(out, slot...)
	}
	return out
}

func placeholderProject(repo detect.Repo) scanner.Project {
	return scanner.Project{Dir: repo.Dir, Label: repo.Label}
}

// joinLabel nests a repository's project labels under the repository:
// "." (the repository itself) keeps the repository label.
func joinLabel(repoLabel, projectLabel string) string {
	if projectLabel == "." || projectLabel == "" {
		return repoLabel
	}
	return filepath.Join(repoLabel, projectLabel)
}
