package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/orchestrator"
	"github.com/luchrv/lazyncu/scanner"
	"github.com/luchrv/lazyncu/semver"
)

// entry builds a settled folder entry with the given label and directory.
func entry(label, dir string, pkgs ...scanner.Package) orchestrator.ProjectResult {
	return orchestrator.ProjectResult{
		Project: scanner.Project{Dir: dir, Label: label, Packages: pkgs,
			Counters: semver.Counters{Major: len(pkgs)}},
		Audit: audit.Result{Status: audit.StatusOK},
	}
}

func pendingEntry(label, dir string) orchestrator.ProjectResult {
	return orchestrator.ProjectResult{Project: scanner.Project{Dir: dir, Label: label}, Pending: true}
}

func failedEntry(label, dir string, err error) orchestrator.ProjectResult {
	return orchestrator.ProjectResult{Project: scanner.Project{Dir: dir, Label: label}, Err: err}
}

func folderEvent(src string, done bool, entries ...orchestrator.ProjectResult) orchestrator.Event {
	return orchestrator.Event{Source: src, Projects: entries, Folder: true, Done: done}
}

func printable(s string) string { return colorTag.ReplaceAllString(s, "") }

// --- applyEvent with progressive snapshots ---

func TestFolderSnapshotsKeepSourceLoadingUntilDone(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.state["/hub"].loading = true

	// Act: placeholders, then one completion, then the final snapshot
	a.applyEvent(folderEvent("/hub", false, pendingEntry("api", "/hub/api"), pendingEntry("web", "/hub/web")))
	stillLoading := a.state["/hub"].loading
	a.applyEvent(folderEvent("/hub", false, entry("api", "/hub/api"), pendingEntry("web", "/hub/web")))
	a.applyEvent(folderEvent("/hub", true, entry("api", "/hub/api"), entry("web", "/hub/web")))

	// Assert
	if !stillLoading {
		t.Error("source must stay loading while entries are pending")
	}
	if a.state["/hub"].loading {
		t.Error("source must stop loading on the Done snapshot")
	}
	if got := len(a.state["/hub"].event.Projects); got != 2 {
		t.Errorf("entries = %d, want 2", got)
	}
}

func TestSelectionFollowsEntryLabelWhenSnapshotChangesShape(t *testing.T) {
	// Arrange: cursor on "web" (index 1); "mono" then expands into two
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.applyEvent(folderEvent("/hub", false, pendingEntry("mono", "/hub/mono"), pendingEntry("web", "/hub/web")))
	a.sel = selection{source: "/hub", projectIdx: 1}

	// Act
	a.applyEvent(folderEvent("/hub", false,
		entry("mono/packages/a", "/hub/mono/packages/a"),
		entry("mono/packages/b", "/hub/mono/packages/b"),
		pendingEntry("web", "/hub/web")))

	// Assert
	if a.sel != (selection{source: "/hub", projectIdx: 2}) {
		t.Errorf("sel = %+v, want web re-anchored at index 2", a.sel)
	}
}

func TestMarksClearedWhenSnapshotChangesShape(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.applyEvent(folderEvent("/hub", false, entry("api", "/hub/api", scanner.Package{Name: "axios"}), pendingEntry("mono", "/hub/mono")))
	a.state["/hub"].marks = map[int]map[string]bool{0: {"axios": true}}

	// Act: same shape keeps marks; a splice clears them
	a.applyEvent(folderEvent("/hub", false, entry("api", "/hub/api", scanner.Package{Name: "axios"}), pendingEntry("mono", "/hub/mono")))
	kept := len(a.state["/hub"].marks)
	a.applyEvent(folderEvent("/hub", true,
		entry("api", "/hub/api", scanner.Package{Name: "axios"}),
		entry("mono/a", "/hub/mono/a"), entry("mono/b", "/hub/mono/b")))

	// Assert
	if kept != 1 {
		t.Error("marks must survive a snapshot with the same shape")
	}
	if len(a.state["/hub"].marks) != 0 {
		t.Error("marks must be cleared when the entry count changes")
	}
}

func TestPendingSelectionWaitsForTheSettledEntry(t *testing.T) {
	// Arrange
	a := newLaunchApp(t, pathsConfig("/hub"), &Launch{Source: "/hub", ProjectDir: "/hub/api"})

	// Act: placeholder snapshot must not resolve; the settled one must
	a.applyEvent(folderEvent("/hub", false, pendingEntry("api", "/hub/api"), pendingEntry("web", "/hub/web")))
	afterPlaceholders := a.sel
	a.applyEvent(folderEvent("/hub", false, entry("api", "/hub/api"), pendingEntry("web", "/hub/web")))

	// Assert
	if afterPlaceholders != (selection{source: "/hub", projectIdx: -1}) || a.pendingProject == nil && afterPlaceholders.projectIdx >= 0 {
		t.Errorf("selection after placeholders = %+v, want the source row", afterPlaceholders)
	}
	if a.sel != (selection{source: "/hub", projectIdx: 0}) || a.pendingProject != nil {
		t.Errorf("sel = %+v pending = %+v, want api selected and pending cleared", a.sel, a.pendingProject)
	}
}

// --- Rows ---

func TestSourceTextFolderProgress(t *testing.T) {
	st := &sourceState{loading: true, event: folderEvent("/hub", false,
		entry("a", "/hub/a"), pendingEntry("b", "/hub/b"), failedEntry("c", "/hub/c", errors.New("boom")))}

	got := printable(sourceText("/hub", st, spinnerGlyph(0)))

	if !strings.Contains(got, spinnerGlyph(0)+" scanning 2/3") {
		t.Errorf("folder row = %q, want scanning 2/3", got)
	}
}

func TestSourceTextFolderFailedCountAndEmpty(t *testing.T) {
	failed := &sourceState{event: folderEvent("/hub", true,
		entry("a", "/hub/a", scanner.Package{Name: "x"}), failedEntry("b", "/hub/b", errors.New("boom")))}
	empty := &sourceState{event: folderEvent("/hub", true)}

	gotFailed := printable(sourceText("/hub", failed, spinnerGlyph(0)))
	gotEmpty := printable(sourceText("/hub", empty, spinnerGlyph(0)))

	if !strings.Contains(gotFailed, "▲1") || !strings.Contains(gotFailed, "1 failed") {
		t.Errorf("folder row = %q, want aggregate of healthy entries plus '1 failed'", gotFailed)
	}
	if !strings.Contains(gotEmpty, "no projects found") {
		t.Errorf("empty folder row = %q, want 'no projects found'", gotEmpty)
	}
}

func TestProjectTextPendingAndFailed(t *testing.T) {
	pending := printable(projectText(pendingEntry("api", "/hub/api"), spinnerGlyph(3)))
	failed := printable(projectText(failedEntry("api", "/hub/api", errors.New("boom")), spinnerGlyph(3)))

	if !strings.Contains(pending, "api  "+spinnerGlyph(3)+" scanning…") {
		t.Errorf("pending row = %q", pending)
	}
	if !strings.Contains(failed, "api  ✗ scan failed") {
		t.Errorf("failed row = %q", failed)
	}
}

func TestAggregateSourceSkipsPendingAndCountsScanFailures(t *testing.T) {
	agg := aggregateSource([]orchestrator.ProjectResult{
		entry("a", "/hub/a", scanner.Package{Name: "x"}),
		pendingEntry("b", "/hub/b"),
		failedEntry("c", "/hub/c", errors.New("boom")),
	})

	if agg.updates.Major != 1 || agg.audited != 1 || agg.scanFailed != 1 || agg.failed != 0 {
		t.Errorf("aggregate = %+v, want 1 major, 1 audited, 1 scanFailed", agg)
	}
}

func TestAnyLoadingIncludesPendingEntries(t *testing.T) {
	a := newTestApp(t)
	a.state[orchestrator.SourceGlobal].loading = false
	registerPath(a, "/hub")
	a.state["/hub"].event = folderEvent("/hub", true, entry("a", "/hub/a"))
	if a.anyLoading() {
		t.Fatal("settled folder must not report loading")
	}

	a.state["/hub"].event = folderEvent("/hub", true, pendingEntry("a", "/hub/a"))

	if !a.anyLoading() {
		t.Error("a pending entry rescan must keep the spinner alive")
	}
}

// --- Detail panel and command bar ---

func TestDetailShowsEntryStateInsideLoadingFolder(t *testing.T) {
	// Arrange: folder still scanning; api settled, web pending, bad failed
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.state["/hub"].loading = true
	a.state["/hub"].event = folderEvent("/hub", false,
		entry("api", "/hub/api", scanner.Package{Name: "axios", Current: "1.0.0", New: "2.0.0"}),
		failedEntry("bad", "/hub/bad", errors.New("signal: killed")),
		pendingEntry("web", "/hub/web"))

	// Act / Assert: settled entry renders its table and its command
	a.sel = selection{source: "/hub", projectIdx: 0}
	a.refreshDetail()
	if got := a.detail.GetCell(0, 0).Text; got != "Package" {
		t.Errorf("settled entry header = %q, want the package table", got)
	}
	if update, _ := a.currentCommands(); !strings.Contains(update, "/hub/api") {
		t.Errorf("settled entry command = %q, want an update command for /hub/api", update)
	}

	// failed entry shows its own reason and no command
	a.sel = selection{source: "/hub", projectIdx: 1}
	a.refreshDetail()
	if got := a.detail.GetCell(0, 0).Text; !strings.Contains(got, "scan failed") {
		t.Errorf("failed entry cell = %q, want the scan-failed header", got)
	}
	if got := a.detail.GetCell(2, 0).Text; !strings.Contains(got, "signal: killed") {
		t.Errorf("failed entry reason = %q, want the failure text", got)
	}
	if update, fix := a.currentCommands(); update != "" || fix != "" {
		t.Errorf("failed entry commands = %q/%q, want none", update, fix)
	}

	// pending entry shows the loading message and no command
	a.sel = selection{source: "/hub", projectIdx: 2}
	a.refreshDetail()
	if got := a.detail.GetCell(0, 0).Text; !strings.Contains(got, "scanning") {
		t.Errorf("pending entry cell = %q, want the loading message", got)
	}
	if update, _ := a.currentCommands(); update != "" {
		t.Errorf("pending entry command = %q, want none", update)
	}
}

func TestDetailEmptyFolderSaysNoProjects(t *testing.T) {
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.cfg.Paths = pathsConfig("/hub").Paths
	a.state["/hub"].event = folderEvent("/hub", true)

	a.refreshDetail()

	if got := a.detail.GetCell(0, 0).Text; got != "no projects found" {
		t.Errorf("empty folder detail = %q", got)
	}
}

// --- Entry rescan ---

func TestRescanEntryMarksOnlyThatEntryPending(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.state["/hub"].event = folderEvent("/hub", true, entry("api", "/hub/api"), entry("web", "/hub/web"))
	a.sel = selection{source: "/hub", projectIdx: 1}

	// Act
	a.handleKey(runeEvent('r'))

	// Assert
	projects := a.state["/hub"].event.Projects
	if projects[0].Pending || !projects[1].Pending {
		t.Errorf("entries = %+v, want only web pending", projects)
	}
	if a.state["/hub"].loading {
		t.Error("an entry rescan must not put the whole folder in loading state")
	}
	if a.pages.HasPage(pageConfirm) {
		t.Error("entry rescan without marks must not ask")
	}
}

func TestRescanEntryWithMarksAsksAndCountsOnlyItsMarks(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.state["/hub"].event = folderEvent("/hub", true, entry("api", "/hub/api"), entry("web", "/hub/web"))
	a.state["/hub"].marks = map[int]map[string]bool{0: {"axios": true, "chalk": true, "lodash": true}, 1: {"react": true}}
	a.sel = selection{source: "/hub", projectIdx: 1}

	// Act
	a.handleKey(runeEvent('r'))

	// Assert
	if !a.pages.HasPage(pageConfirm) {
		t.Fatal("rescan of a marked entry must ask first")
	}
	if a.state["/hub"].event.Projects[1].Pending {
		t.Error("the rescan must not start until confirmed")
	}
	if got := confirmRescanText("web", 1, 1); !strings.Contains(got, "1 mark across 1 project") {
		t.Errorf("confirmation wording = %q, want only the entry's own mark counted", got)
	}
}

func TestRescanEntryBlockedWhilePending(t *testing.T) {
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.state["/hub"].event = folderEvent("/hub", true, pendingEntry("api", "/hub/api"), entry("web", "/hub/web"))
	a.sel = selection{source: "/hub", projectIdx: 0}

	a.handleKey(runeEvent('r'))

	if msg := a.statusMsg.GetText(true); !strings.Contains(msg, "still scanning") {
		t.Errorf("expected in-flight guard message, got %q", msg)
	}
}

func TestRescanFolderRowBlockedWhileAnEntryIsPending(t *testing.T) {
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.state["/hub"].event = folderEvent("/hub", true, pendingEntry("api", "/hub/api"), entry("web", "/hub/web"))
	a.sel = selection{source: "/hub", projectIdx: -1}

	a.handleKey(runeEvent('r'))

	if a.state["/hub"].loading {
		t.Error("folder rescan must not start while an entry rescan is in flight")
	}
	if msg := a.statusMsg.GetText(true); !strings.Contains(msg, "still scanning") {
		t.Errorf("expected in-flight guard message, got %q", msg)
	}
}

func TestRescanFolderRowRescansWholeSource(t *testing.T) {
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.state["/hub"].event = folderEvent("/hub", true, entry("api", "/hub/api"), entry("web", "/hub/web"))
	a.sel = selection{source: "/hub", projectIdx: -1}

	a.handleKey(runeEvent('r'))

	if !a.state["/hub"].loading {
		t.Error("rescanning the folder row must put the source in loading state")
	}
}

func TestSpliceEntryReplacesByLabelAndReanchors(t *testing.T) {
	// Arrange: cursor on web; mono is pending and expands into two entries
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.state["/hub"].event = folderEvent("/hub", true, pendingEntry("mono", "/hub/mono"), entry("web", "/hub/web"))
	a.sel = selection{source: "/hub", projectIdx: 1}

	// Act
	a.spliceEntry("/hub", "mono", []orchestrator.ProjectResult{
		entry("mono/a", "/hub/mono/a"), entry("mono/b", "/hub/mono/b"),
	})

	// Assert
	projects := a.state["/hub"].event.Projects
	if len(projects) != 3 || projects[0].Label != "mono/a" || projects[2].Label != "web" || projects[0].Pending {
		t.Errorf("entries = %+v, want mono expanded in place before web", projects)
	}
	if a.sel != (selection{source: "/hub", projectIdx: 2}) {
		t.Errorf("sel = %+v, want web re-anchored at index 2", a.sel)
	}
}

func TestSpliceEntryIgnoresRemovedSourceOrUnknownLabel(t *testing.T) {
	a := newTestApp(t)
	registerPath(a, "/hub")
	a.state["/hub"].event = folderEvent("/hub", true, entry("api", "/hub/api"))

	a.spliceEntry("/gone", "api", nil)
	a.spliceEntry("/hub", "nope", []orchestrator.ProjectResult{entry("x", "/x")})

	if got := a.state["/hub"].event.Projects; len(got) != 1 || got[0].Label != "api" {
		t.Errorf("entries = %+v, want untouched", got)
	}
}
