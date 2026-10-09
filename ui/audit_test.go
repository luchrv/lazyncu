package ui

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/orchestrator"
	"github.com/luchrv/lazyncu/scanner"
)

func pendingProject(dir string) orchestrator.ProjectResult {
	return orchestrator.ProjectResult{
		Project: scanner.Project{Dir: dir, Label: dir, PM: detect.Npm},
		Audit:   audit.Deferred(detect.Npm),
	}
}

func pendingEvent(source string) orchestrator.Event {
	ev := orchestrator.Event{Source: source, Done: true, Projects: []orchestrator.ProjectResult{pendingProject(source)}}
	ev.Projects[0].Label = "."
	return ev
}

func TestSelectingProjectStartsAuditOnce(t *testing.T) {
	// Arrange: an auditor that records every request
	a := newTestApp(t)
	audited := make(chan string, 4)
	a.deps.Auditor = func(_ context.Context, dir string, _ detect.PackageManager) audit.Result {
		audited <- dir
		return audit.Result{Status: audit.StatusOK}
	}
	registerPath(a, "/p/app")

	// Act: the scan lands, then the selection is refreshed twice
	a.applyEvent(pendingEvent("/p/app"))
	a.ensureAudit()
	a.ensureAudit()

	// Assert: one audit in flight, marked running
	select {
	case dir := <-audited:
		if dir != "/p/app" {
			t.Errorf("audited %q, want /p/app", dir)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("auditor not invoked after the project was selected")
	}
	select {
	case dir := <-audited:
		t.Fatalf("auditor invoked twice (%s), want once per pending audit", dir)
	case <-time.After(50 * time.Millisecond):
	}
	if got := a.state["/p/app"].event.Projects[0].Audit.Status; got != audit.StatusRunning {
		t.Errorf("audit status = %v, want StatusRunning while in flight", got)
	}
}

func TestApplyAuditRecordsResultAndDropsStaleOnes(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	registerPath(a, "/p/app")
	a.applyEvent(pendingEvent("/p/app"))
	key := auditKey{source: "/p/app", dir: "/p/app"}
	a.audits.gen = map[string]int{"/p/app": 2}
	a.audits.running = 2
	fresh := audit.Result{Status: audit.StatusOK, Counters: audit.Counters{High: 1}}

	// Act
	a.applyAudit(key, 1, audit.Result{Status: audit.StatusFailed, Err: "stale"})
	a.applyAudit(key, 2, fresh)

	// Assert
	got := a.state["/p/app"].event.Projects[0].Audit
	if got.Status != audit.StatusOK || got.Counters.High != 1 {
		t.Errorf("audit = %+v, want the fresh result and the stale one discarded", got)
	}
	if a.audits.running != 0 {
		t.Errorf("running = %d, want 0", a.audits.running)
	}
}

func TestNoAuditStartedForUnauditableOrScanningEntries(t *testing.T) {
	a := newTestApp(t)
	a.deps.Auditor = func(context.Context, string, detect.PackageManager) audit.Result {
		t.Error("auditor invoked for an entry that is not pending")
		return audit.Result{}
	}
	registerPath(a, "/p/hub")
	a.applyEvent(orchestrator.Event{Source: "/p/hub", Folder: true, Projects: []orchestrator.ProjectResult{
		{Project: scanner.Project{Dir: "/p/hub/a", Label: "a"}, Pending: true},
		{Project: scanner.Project{Dir: "/p/hub/b", Label: "b", PM: detect.Yarn}, Audit: audit.Deferred(detect.Yarn)},
	}})
	a.sel = selection{source: "/p/hub", projectIdx: 0}
	a.ensureAudit()
	a.sel = selection{source: "/p/hub", projectIdx: 1}
	a.ensureAudit()
	a.enqueuePendingAudits()
	time.Sleep(50 * time.Millisecond)
}

func TestQueueAuditsAllPendingBoundedByLimit(t *testing.T) {
	// Arrange: a folder of five projects, limit 2, auditor blocks until released
	a := newTestApp(t)
	a.cfg.MaxParallel = 2
	var inflight, peak int32
	release := make(chan struct{})
	a.deps.Auditor = func(_ context.Context, _ string, _ detect.PackageManager) audit.Result {
		n := atomic.AddInt32(&inflight, 1)
		for p := atomic.LoadInt32(&peak); n > p && !atomic.CompareAndSwapInt32(&peak, p, n); p = atomic.LoadInt32(&peak) {
		}
		<-release
		atomic.AddInt32(&inflight, -1)
		return audit.Result{Status: audit.StatusOK}
	}
	registerPath(a, "/p/hub")
	var projects []orchestrator.ProjectResult
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		projects = append(projects, pendingProject("/p/hub/"+name))
	}
	globalSel(a)
	a.applyEvent(orchestrator.Event{Source: "/p/hub", Folder: true, Done: true, Projects: projects})

	// Act: the scan stream closes
	a.enqueuePendingAudits()
	time.Sleep(50 * time.Millisecond)

	// Assert: two running, three queued in panel order
	if a.audits.running != 2 || len(a.audits.pending) != 3 || a.audits.pending[0].dir != "/p/hub/c" {
		t.Errorf("running = %d pending = %v, want 2 running and c,d,e queued", a.audits.running, a.audits.pending)
	}
	if atomic.LoadInt32(&peak) != 2 {
		t.Errorf("peak in-flight audits = %d, want 2", peak)
	}

	// Act: selecting a queued project jumps the queue
	a.sel = selection{source: "/p/hub", projectIdx: 4}
	a.ensureAudit()
	time.Sleep(50 * time.Millisecond)
	if a.audits.running != 3 || containsDir(a.audits.pending, "/p/hub/e") {
		t.Errorf("after selecting e: running = %d pending = %v, want e started immediately", a.audits.running, a.audits.pending)
	}
	close(release)
}

func containsDir(keys []auditKey, dir string) bool {
	for _, k := range keys {
		if k.dir == dir {
			return true
		}
	}
	return false
}
