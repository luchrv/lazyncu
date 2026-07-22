package orchestrator

import (
	"context"
	"errors"
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
	events := Run(context.Background(), sc, okAuditor, []string{"/p/a", "/p/b"})
	got := collect(t, events, 3)

	// Assert: channel closes after exactly one event per source
	if _, open := <-events; open {
		t.Error("channel still open after all sources delivered")
	}
	seen := map[string]int{}
	for _, ev := range got {
		seen[ev.Source]++
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
	events := Run(context.Background(), sc, okAuditor, []string{"/p/fast"})

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
	events := Run(context.Background(), sc, okAuditor, []string{"/p/ok", "/p/bad"})
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
	events := Run(context.Background(), sc, failingAuditor, []string{"/p/repo"})
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
	ev := RunOne(context.Background(), sc, okAuditor, "/p/a")

	// Assert
	if ev.Source != "/p/a" || ev.Err != nil || len(ev.Projects) != 1 {
		t.Errorf("RunOne() = %+v, want one project for /p/a", ev)
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
	events := Run(context.Background(), sc, okAuditor, nil)
	got := collect(t, events, 1)

	// Assert
	if got[0].GlobalAudit.Status != audit.StatusNotAvailable {
		t.Errorf("global audit status = %v, want StatusNotAvailable", got[0].GlobalAudit.Status)
	}
}
