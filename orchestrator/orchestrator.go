// Package orchestrator fans out version scans and vulnerability audits —
// one goroutine per source, one audit goroutine per discovered project —
// and streams immutable per-source events on a channel as they complete.
package orchestrator

import (
	"context"
	"sync"

	"github.com/luchrv/ncu-tui/audit"
	"github.com/luchrv/ncu-tui/detect"
	"github.com/luchrv/ncu-tui/scanner"
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

// ProjectResult pairs one project's scan result with its audit outcome.
type ProjectResult struct {
	scanner.Project
	Audit audit.Result
}

// Event is one source's complete outcome. Exactly one Event is delivered per
// source; a source failure sets Err and never affects sibling sources.
type Event struct {
	// Source is SourceGlobal or the registered path.
	Source string
	// Packages holds global-source results (Source == SourceGlobal).
	Packages    []scanner.Package
	GlobalAudit audit.Result
	// Projects holds path-source results, each with its audit.
	Projects []ProjectResult
	Err      error
}

// Run launches all scans concurrently and returns the event channel, which
// closes once every source has delivered exactly one event.
func Run(ctx context.Context, sc Scanner, auditor Auditor, paths []string) <-chan Event {
	events := make(chan Event)
	var wg sync.WaitGroup

	wg.Go(func() {
		events <- RunGlobal(ctx, sc)
	})

	for _, path := range paths {
		wg.Go(func() {
			events <- scanAndAuditPath(ctx, sc, auditor, path)
		})
	}

	go func() {
		wg.Wait()
		close(events)
	}()
	return events
}

// RunOne scans and audits a single path synchronously — used when a path is
// added at runtime or rescanned and only that source needs scanning.
func RunOne(ctx context.Context, sc Scanner, auditor Auditor, path string) Event {
	return scanAndAuditPath(ctx, sc, auditor, path)
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
	}
}

// scanAndAuditPath scans one registered path, then audits every discovered
// project concurrently. Audit failures degrade to per-project badges; only a
// scan failure marks the source as failed.
func scanAndAuditPath(ctx context.Context, sc Scanner, auditor Auditor, path string) Event {
	projects, err := sc.ScanPath(ctx, path)
	if err != nil {
		return Event{Source: path, Err: err}
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
	return Event{Source: path, Projects: results}
}
