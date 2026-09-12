package orchestrator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/scanner"
)

// fakeScanner lets each source block on a gate channel to control ordering.
type fakeScanner struct {
	globalPkgs []scanner.Package
	globalErr  error
	pathRes    map[string][]scanner.Project
	pathErr    map[string]error
	gates      map[string]chan struct{} // optional per-source gate ("global" or path)
}

func (f *fakeScanner) ScanGlobal(ctx context.Context) ([]scanner.Package, error) {
	f.wait(ctx, "global")
	return f.globalPkgs, f.globalErr
}

func (f *fakeScanner) ScanPath(ctx context.Context, dir string) ([]scanner.Project, error) {
	f.wait(ctx, dir)
	if err, ok := f.pathErr[dir]; ok {
		return nil, err
	}
	return f.pathRes[dir], nil
}

func (f *fakeScanner) wait(ctx context.Context, key string) {
	if gate, ok := f.gates[key]; ok {
		select {
		case <-gate:
		case <-ctx.Done():
		}
	}
}

func okAuditor(ctx context.Context, dir string, pm detect.PackageManager) audit.Result {
	return audit.Result{Status: audit.StatusOK}
}

func collect(t *testing.T, events <-chan Event, want int) []Event {
	t.Helper()
	got := make([]Event, 0, want)
	timeout := time.After(5 * time.Second)
	for len(got) < want {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Fatalf("channel closed after %d events, want %d", len(got), want)
			}
			got = append(got, ev)
		case <-timeout:
			t.Fatalf("timed out after %d events, want %d", len(got), want)
		}
	}
	return got
}

func TestFanOutDeliversAllSourcesExactlyOnce(t *testing.T) {
	// Arrange
	sc := &fakeScanner{
		globalPkgs: []scanner.Package{{Name: "typescript", New: "5.6.2"}},
		pathRes: map[string][]scanner.Project{
			"/p/a": {{Dir: "/p/a", Label: ".", PM: detect.Npm}},
			"/p/b": {{Dir: "/p/b", Label: ".", PM: detect.Npm}},
		},
	}

	// Act
	events := Run(context.Background(), Deps{Scanner: sc, Auditor: okAuditor}, []string{"/p/a", "/p/b"})
	got := collect(t, events, 3)

	// Assert: channel closes after exactly one done event per source
	if _, open := <-events; open {
		t.Error("channel still open after all sources delivered")
	}
	seen := map[string]int{}
	for _, ev := range got {
		seen[ev.Source]++
		if !ev.Done {
			t.Errorf("non-folder source %q delivered an event without Done", ev.Source)
		}
	}
	for _, source := range []string{SourceGlobal, "/p/a", "/p/b"} {
		if seen[source] != 1 {
			t.Errorf("source %q delivered %d times, want exactly once", source, seen[source])
		}
	}
}

func TestSlowSourceDoesNotBlockFastOnes(t *testing.T) {
	// Arrange: global is gated (slow); paths are free
	gate := make(chan struct{})
	sc := &fakeScanner{
		pathRes: map[string][]scanner.Project{"/p/fast": {{Dir: "/p/fast", Label: "."}}},
		gates:   map[string]chan struct{}{"global": gate},
	}

	// Act
	events := Run(context.Background(), Deps{Scanner: sc, Auditor: okAuditor}, []string{"/p/fast"})

	// Assert: the fast path arrives while global is still blocked
	first := collect(t, events, 1)[0]
	if first.Source != "/p/fast" {
		t.Errorf("first event = %q, want /p/fast while global hangs", first.Source)
	}
	close(gate)
	rest := collect(t, events, 1)
	if rest[0].Source != SourceGlobal {
		t.Errorf("second event = %q, want global after release", rest[0].Source)
	}
}

func TestScanErrorIsIsolatedPerSource(t *testing.T) {
	// Arrange
	sc := &fakeScanner{
		pathRes: map[string][]scanner.Project{"/p/ok": {{Dir: "/p/ok", Label: "."}}},
		pathErr: map[string]error{"/p/bad": errors.New("ncu exploded")},
	}

	// Act
	events := Run(context.Background(), Deps{Scanner: sc, Auditor: okAuditor}, []string{"/p/ok", "/p/bad"})
	got := collect(t, events, 3)

	// Assert
	bySource := map[string]Event{}
	for _, ev := range got {
		bySource[ev.Source] = ev
	}
	if bySource["/p/bad"].Err == nil {
		t.Error("failed source must carry its error")
	}
	if bySource["/p/ok"].Err != nil || len(bySource["/p/ok"].Projects) != 1 {
		t.Errorf("healthy source affected by sibling failure: %+v", bySource["/p/ok"])
	}
}

func TestAuditRunsPerProjectAndFailureKeepsScanResult(t *testing.T) {
	// Arrange: two projects under one path; audits fail
	sc := &fakeScanner{
		pathRes: map[string][]scanner.Project{
			"/p/repo": {
				{Dir: "/p/repo/api", Label: "api", PM: detect.Npm,
					Packages: []scanner.Package{{Name: "express", Current: "4.18.0", New: "5.1.0"}}},
				{Dir: "/p/repo/web", Label: "web", PM: detect.Npm},
			},
		},
	}
	audited := make(chan string, 2)
	failingAuditor := func(ctx context.Context, dir string, pm detect.PackageManager) audit.Result {
		audited <- dir
		return audit.Result{Status: audit.StatusFailed, Err: "audit exploded"}
	}

	// Act
	events := Run(context.Background(), Deps{Scanner: sc, Auditor: failingAuditor}, []string{"/p/repo"})
	got := collect(t, events, 2)

	// Assert: both projects audited
	close(audited)
	auditedDirs := map[string]bool{}
	for dir := range audited {
		auditedDirs[dir] = true
	}
	if !auditedDirs["/p/repo/api"] || !auditedDirs["/p/repo/web"] {
		t.Errorf("audited dirs = %v, want both projects", auditedDirs)
	}
	// Assert: scan results intact despite audit failure
	var repo Event
	for _, ev := range got {
		if ev.Source == "/p/repo" {
			repo = ev
		}
	}
	if repo.Err != nil {
		t.Fatalf("source Err = %v, audit failure must not fail the source", repo.Err)
	}
	for _, p := range repo.Projects {
		if p.Audit.Status != audit.StatusFailed {
			t.Errorf("project %s audit status = %v, want StatusFailed", p.Label, p.Audit.Status)
		}
		if p.Label == "api" && len(p.Packages) != 1 {
			t.Errorf("api packages lost: %+v", p.Packages)
		}
	}
}

func TestRunOneScansSinglePath(t *testing.T) {
	// Arrange
	sc := &fakeScanner{
		pathRes: map[string][]scanner.Project{"/p/a": {{Dir: "/p/a", Label: ".", PM: detect.Npm}}},
	}

	// Act
	events := RunOne(context.Background(), Deps{Scanner: sc, Auditor: okAuditor}, "/p/a")
	ev := collect(t, events, 1)[0]

	// Assert: one final event, then the channel closes
	if ev.Source != "/p/a" || ev.Err != nil || len(ev.Projects) != 1 || !ev.Done {
		t.Errorf("RunOne() = %+v, want one done project event for /p/a", ev)
	}
	if _, open := <-events; open {
		t.Error("RunOne channel still open after the final event")
	}
}

func TestRunGlobalScansGlobalSource(t *testing.T) {
	// Arrange
	sc := &fakeScanner{globalPkgs: []scanner.Package{{Name: "typescript", New: "5.6.2"}}}

	// Act
	ev := RunGlobal(context.Background(), sc)

	// Assert
	if ev.Source != SourceGlobal || len(ev.Packages) != 1 {
		t.Errorf("RunGlobal() = %+v, want the global package list", ev)
	}
	if ev.GlobalAudit.Status != audit.StatusNotAvailable {
		t.Errorf("GlobalAudit.Status = %v, want StatusNotAvailable", ev.GlobalAudit.Status)
	}
}

func TestGlobalEventCarriesAuditNotAvailable(t *testing.T) {
	// Arrange
	sc := &fakeScanner{globalPkgs: []scanner.Package{{Name: "typescript", New: "5.6.2"}}}

	// Act
	events := Run(context.Background(), Deps{Scanner: sc, Auditor: okAuditor}, nil)
	got := collect(t, events, 1)

	// Assert
	if got[0].GlobalAudit.Status != audit.StatusNotAvailable {
		t.Errorf("global audit status = %v, want StatusNotAvailable", got[0].GlobalAudit.Status)
	}
}

// --- Folder sources ---

// folderDiscoverer returns canned repositories for the given paths.
func folderDiscoverer(folders map[string][]detect.Repo, errs map[string]error) Discoverer {
	return func(dir string) ([]detect.Repo, bool, error) {
		if err, ok := errs[dir]; ok {
			return nil, true, err
		}
		repos, ok := folders[dir]
		return repos, ok, nil
	}
}

func repo(dir, label string) detect.Repo { return detect.Repo{Dir: dir, Label: label} }

func labelsOf(projects []ProjectResult) []string {
	out := make([]string, len(projects))
	for i, p := range projects {
		out[i] = p.Label
	}
	return out
}

func TestFolderAnnouncesPendingEntriesBeforeAnyResult(t *testing.T) {
	// Arrange: two repositories, both gated
	gateA, gateB := make(chan struct{}), make(chan struct{})
	sc := &fakeScanner{
		pathRes: map[string][]scanner.Project{
			"/hub/api": {{Dir: "/hub/api", Label: ".", PM: detect.Npm}},
			"/hub/web": {{Dir: "/hub/web", Label: ".", PM: detect.Npm}},
		},
		gates: map[string]chan struct{}{"/hub/api": gateA, "/hub/web": gateB},
	}
	deps := Deps{Scanner: sc, Auditor: okAuditor, Discoverer: folderDiscoverer(
		map[string][]detect.Repo{"/hub": {repo("/hub/web", "web"), repo("/hub/api", "api")}}, nil)}

	// Act
	events := RunOne(context.Background(), deps, "/hub")
	first := collect(t, events, 1)[0]

	// Assert: placeholders sorted by label, nothing done yet
	if !first.Folder || first.Done {
		t.Errorf("first event = {Folder:%v Done:%v}, want folder snapshot not done", first.Folder, first.Done)
	}
	if got := labelsOf(first.Projects); len(got) != 2 || got[0] != "api" || got[1] != "web" {
		t.Errorf("placeholder labels = %v, want [api web]", got)
	}
	for _, p := range first.Projects {
		if !p.Pending {
			t.Errorf("entry %s delivered as not pending before its scan ran", p.Label)
		}
	}

	// Act: web completes first, then api
	close(gateB)
	second := collect(t, events, 1)[0]
	close(gateA)
	third := collect(t, events, 1)[0]

	// Assert: each completion changes only its own slot; the last is Done
	if second.Done || second.Projects[0].Pending != true || second.Projects[1].Pending {
		t.Errorf("second snapshot = %+v, want api pending and web resolved", second.Projects)
	}
	if !third.Done || third.Projects[0].Pending || third.Projects[1].Pending {
		t.Errorf("third snapshot = %+v, want both resolved and Done", third.Projects)
	}
	if third.Projects[0].PM != detect.Npm || third.Projects[0].Audit.Status != audit.StatusOK {
		t.Errorf("api entry = %+v, want scanned and audited", third.Projects[0])
	}
	if _, open := <-events; open {
		t.Error("channel still open after the folder's final snapshot")
	}
}

func TestFolderSnapshotsAreNeverMutatedAfterEmission(t *testing.T) {
	// Arrange
	gate := make(chan struct{})
	sc := &fakeScanner{
		pathRes: map[string][]scanner.Project{
			"/hub/api": {{Dir: "/hub/api", Label: "."}},
			"/hub/web": {{Dir: "/hub/web", Label: "."}},
		},
		gates: map[string]chan struct{}{"/hub/api": gate},
	}
	deps := Deps{Scanner: sc, Auditor: okAuditor, Discoverer: folderDiscoverer(
		map[string][]detect.Repo{"/hub": {repo("/hub/api", "api"), repo("/hub/web", "web")}}, nil)}

	// Act
	events := RunOne(context.Background(), deps, "/hub")
	got := collect(t, events, 2)
	close(gate)
	collect(t, events, 1)

	// Assert: the placeholder snapshot still says both pending
	for _, p := range got[0].Projects {
		if !p.Pending {
			t.Errorf("earlier snapshot mutated: %s no longer pending", p.Label)
		}
	}
}

func TestFolderOneFailingRepositoryDoesNotFailTheSource(t *testing.T) {
	// Arrange
	sc := &fakeScanner{
		pathRes: map[string][]scanner.Project{"/hub/ok": {{Dir: "/hub/ok", Label: "."}}},
		pathErr: map[string]error{"/hub/bad": errors.New("signal: killed")},
	}
	deps := Deps{Scanner: sc, Auditor: okAuditor, Discoverer: folderDiscoverer(
		map[string][]detect.Repo{"/hub": {repo("/hub/bad", "bad"), repo("/hub/ok", "ok")}}, nil)}

	// Act
	events := RunOne(context.Background(), deps, "/hub")
	got := collect(t, events, 3)
	final := got[2]

	// Assert
	if final.Err != nil || !final.Done {
		t.Fatalf("final = {Err:%v Done:%v}, want no source error and Done", final.Err, final.Done)
	}
	if final.Projects[0].Err == nil || final.Projects[0].Label != "bad" || final.Projects[0].Pending {
		t.Errorf("bad entry = %+v, want its own error", final.Projects[0])
	}
	if final.Projects[1].Err != nil || final.Projects[1].Audit.Status != audit.StatusOK {
		t.Errorf("ok entry = %+v, want healthy result", final.Projects[1])
	}
}

func TestFolderEmptyDiscoveryIsDoneWithZeroEntries(t *testing.T) {
	// Arrange
	deps := Deps{Scanner: &fakeScanner{}, Auditor: okAuditor,
		Discoverer: folderDiscoverer(map[string][]detect.Repo{"/hub": {}}, nil)}

	// Act
	events := RunOne(context.Background(), deps, "/hub")
	got := collect(t, events, 1)[0]

	// Assert
	if !got.Done || !got.Folder || got.Err != nil || len(got.Projects) != 0 {
		t.Errorf("empty folder event = %+v, want Done folder with zero entries", got)
	}
	if _, open := <-events; open {
		t.Error("channel still open after the empty folder's event")
	}
}

func TestFolderDiscoveryErrorFailsTheSource(t *testing.T) {
	// Arrange
	deps := Deps{Scanner: &fakeScanner{}, Auditor: okAuditor,
		Discoverer: folderDiscoverer(nil, map[string]error{"/hub": errors.New("permission denied")})}

	// Act
	got := collect(t, RunOne(context.Background(), deps, "/hub"), 1)[0]

	// Assert
	if got.Err == nil || !got.Done {
		t.Errorf("discovery failure event = %+v, want source Err and Done", got)
	}
}

func TestFolderMonorepoRepositoryExpandsUnderItsLabel(t *testing.T) {
	// Arrange: the "mono" repository yields two workspace projects
	sc := &fakeScanner{
		pathRes: map[string][]scanner.Project{
			"/hub/mono": {
				{Dir: "/hub/mono/packages/a", Label: "packages/a"},
				{Dir: "/hub/mono/packages/b", Label: "packages/b"},
			},
			"/hub/plain": {{Dir: "/hub/plain", Label: "."}},
		},
	}
	deps := Deps{Scanner: sc, Auditor: okAuditor, Discoverer: folderDiscoverer(
		map[string][]detect.Repo{"/hub": {repo("/hub/mono", "mono"), repo("/hub/plain", "plain")}}, nil)}

	// Act
	got := collect(t, RunOne(context.Background(), deps, "/hub"), 3)
	final := got[2]

	// Assert: mono expanded in place, plain keeps its own label
	want := []string{"mono/packages/a", "mono/packages/b", "plain"}
	if labels := labelsOf(final.Projects); !slices.Equal(labels, want) {
		t.Errorf("labels = %v, want %v", labels, want)
	}
}

func TestScanProjectCoversSuccessFailureAndEmptyScan(t *testing.T) {
	// Arrange
	sc := &fakeScanner{
		pathRes: map[string][]scanner.Project{
			"/hub/ok":    {{Dir: "/hub/ok", Label: ".", PM: detect.Pnpm}},
			"/hub/empty": {},
		},
		pathErr: map[string]error{"/hub/bad": errors.New("boom")},
	}
	deps := Deps{Scanner: sc, Auditor: okAuditor}

	// Act
	ok := ScanProject(context.Background(), deps, repo("/hub/ok", "ok"))
	bad := ScanProject(context.Background(), deps, repo("/hub/bad", "bad"))
	empty := ScanProject(context.Background(), deps, repo("/hub/empty", "empty"))

	// Assert
	if len(ok) != 1 || ok[0].Label != "ok" || ok[0].PM != detect.Pnpm || ok[0].Audit.Status != audit.StatusOK {
		t.Errorf("ok = %+v, want one audited entry labeled ok", ok)
	}
	if len(bad) != 1 || bad[0].Err == nil || bad[0].Label != "bad" || bad[0].Dir != "/hub/bad" {
		t.Errorf("bad = %+v, want one failed entry keeping dir and label", bad)
	}
	if len(empty) != 1 || empty[0].Label != "empty" || empty[0].Err != nil || empty[0].Pending {
		t.Errorf("empty = %+v, want one up-to-date entry so the repository stays visible", empty)
	}
}

func TestNilDiscovererTreatsEveryPathAsNonFolder(t *testing.T) {
	// Arrange
	sc := &fakeScanner{pathRes: map[string][]scanner.Project{"/p/a": {{Dir: "/p/a", Label: "."}}}}

	// Act
	got := collect(t, RunOne(context.Background(), Deps{Scanner: sc, Auditor: okAuditor}, "/p/a"), 1)[0]

	// Assert
	if got.Folder || !got.Done || len(got.Projects) != 1 {
		t.Errorf("event = %+v, want a plain done event", got)
	}
}

func TestDiscoverReposClassifiesFolderAndProject(t *testing.T) {
	// Arrange: a folder holding one repository, and the repository itself
	hub := t.TempDir()
	api := filepath.Join(hub, "api")
	if err := os.MkdirAll(api, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(api, "package.json"), []byte(`{"name":"api"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Act
	repos, isFolder, err := DiscoverRepos(hub)
	_, projectIsFolder, projectErr := DiscoverRepos(api)

	// Assert
	if err != nil || !isFolder || len(repos) != 1 || repos[0].Label != "api" {
		t.Errorf("DiscoverRepos(folder) = %v, %v, %v; want one repo labeled api", repos, isFolder, err)
	}
	if projectErr != nil || projectIsFolder {
		t.Errorf("DiscoverRepos(project) = folder %v, err %v; want a non-folder", projectIsFolder, projectErr)
	}
}
