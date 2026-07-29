package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/luchrv/lazyncu/config"
	"github.com/luchrv/lazyncu/orchestrator"
	"github.com/luchrv/lazyncu/scanner"
)

func TestTreeKeyInTableShowsHintAndDoesNotFire(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	registerPath(a, "/tmp/project")
	a.tableFocused = true

	// Act
	got := a.handleKey(runeEvent('d'))

	// Assert
	if got != nil {
		t.Error("out-of-context bound key must be swallowed")
	}
	if _, ok := a.state["/tmp/project"]; !ok {
		t.Fatal("'d' in the table must not remove the path")
	}
	if a.pages.HasPage(pageConfirm) {
		t.Error("'d' in the table must not open the confirmation")
	}
	if msg := a.statusMsg.GetText(true); !strings.Contains(msg, "only works in Sources") {
		t.Errorf("expected teaching hint, got %q", msg)
	}
}

func TestMarkKeyInVulnsViewShowsHint(t *testing.T) {
	a := newTestApp(t)
	a.tableFocused = true
	a.showVulns = true

	got := a.handleKey(runeEvent(' '))

	if got != nil {
		t.Error("space in the vulns view must be swallowed")
	}
	if msg := a.statusMsg.GetText(true); !strings.Contains(msg, "packages view") {
		t.Errorf("expected packages-view hint, got %q", msg)
	}
}

func TestUnboundKeyPassesThroughSilently(t *testing.T) {
	a := newTestApp(t)

	got := a.handleKey(runeEvent('z'))

	if got == nil {
		t.Error("unbound keys must pass through to the focused widget")
	}
	if msg := a.statusMsg.GetText(true); msg != "" {
		t.Errorf("unbound keys must not hint, got %q", msg)
	}
}

func TestAboutModalKeysAreInert(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	registerPath(a, "/tmp/project")
	a.toggleAbout()

	// Act — dashboard keys under the modal.
	for _, r := range []rune{'a', 'd', 'v', 'r', 'm'} {
		if got := a.handleKey(runeEvent(r)); got != nil {
			t.Errorf("%q must be swallowed while About is open", r)
		}
	}

	// Assert — nothing leaked underneath.
	if a.pages.HasPage(pageAddPath) {
		t.Error("'a' stacked the add-path input on top of About")
	}
	if a.pages.HasPage(pageConfirm) {
		t.Error("'d' opened a confirmation under About")
	}
	if _, ok := a.state["/tmp/project"]; !ok {
		t.Error("'d' removed the path while About was open")
	}
	if a.showVulns {
		t.Error("'v' toggled the vulns view while About was open")
	}

	// Act — the modal's own keys still work.
	a.handleKey(runeEvent('h'))
	if a.pages.HasPage(pageAbout) {
		t.Error("'h' must close the About modal")
	}
}

func TestRemovePathAsksConfirmation(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	registerPath(a, "/tmp/project")

	// Act
	a.handleKey(runeEvent('d'))

	// Assert — nothing removed yet, question on screen.
	if !a.pages.HasPage(pageConfirm) {
		t.Fatal("'d' must open the confirmation modal")
	}
	if _, ok := a.state["/tmp/project"]; !ok {
		t.Error("the path must survive until the user confirms")
	}
}

func TestRemoveGlobalSourceIsRefusedWithoutModal(t *testing.T) {
	a := newTestApp(t)
	globalSel(a)

	a.handleKey(runeEvent('d'))

	if a.pages.HasPage(pageConfirm) {
		t.Error("the global source must be refused without a confirmation modal")
	}
	if msg := a.statusMsg.GetText(true); !strings.Contains(msg, "cannot be removed") {
		t.Errorf("expected refusal message, got %q", msg)
	}
}

func TestRescanWithMarksAsksConfirmation(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	registerPath(a, "/tmp/project")
	a.state["/tmp/project"].marks = map[int]map[string]bool{
		0: {"axios": true, "chalk": true},
	}

	// Act
	a.handleKey(runeEvent('r'))

	// Assert — scan not started, marks intact, question on screen.
	if !a.pages.HasPage(pageConfirm) {
		t.Fatal("rescan over marks must ask first")
	}
	if a.state["/tmp/project"].loading {
		t.Error("the rescan must not start until the user confirms")
	}
	if len(a.state["/tmp/project"].marks) == 0 {
		t.Error("marks must survive until the user confirms")
	}
}

func TestRescanWithoutMarksIsImmediate(t *testing.T) {
	a := newTestApp(t)
	registerPath(a, "/tmp/project")

	a.handleKey(runeEvent('r'))

	if a.pages.HasPage(pageConfirm) {
		t.Error("rescan without marks must not ask")
	}
	if !a.state["/tmp/project"].loading {
		t.Error("rescan without marks must start immediately")
	}
}

func TestRescanBlockedWhileScanning(t *testing.T) {
	a := newTestApp(t)
	registerPath(a, "/tmp/project")
	a.state["/tmp/project"].loading = true
	a.state["/tmp/project"].marks = map[int]map[string]bool{0: {"axios": true}}

	a.handleKey(runeEvent('r'))

	if a.pages.HasPage(pageConfirm) {
		t.Error("an in-flight scan must win over the mark confirmation")
	}
	if msg := a.statusMsg.GetText(true); !strings.Contains(msg, "still scanning") {
		t.Errorf("expected in-flight guard message, got %q", msg)
	}
}

func TestKeysModalIsInert(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	registerPath(a, "/tmp/project")
	a.handleKey(runeEvent('?'))

	if !a.pages.HasPage(pageKeys) {
		t.Fatal("'?' must open the cheat sheet")
	}
	if got := a.currentContext(); got != ctxModal {
		t.Fatalf("cheat-sheet context = %d, want modal", got)
	}

	// Act — dashboard keys under the modal.
	for _, r := range []rune{'a', 'd', 'v', 'r', 'h'} {
		if got := a.handleKey(runeEvent(r)); got != nil {
			t.Errorf("%q must be swallowed while the cheat sheet is open", r)
		}
	}

	// Assert — nothing leaked.
	if a.pages.HasPage(pageAddPath) || a.pages.HasPage(pageConfirm) || a.pages.HasPage(pageAbout) {
		t.Error("dashboard keys acted underneath the cheat sheet")
	}

	// Act — '?' closes it again.
	a.handleKey(runeEvent('?'))
	if a.pages.HasPage(pageKeys) {
		t.Error("'?' must close the cheat sheet")
	}
}

func TestFocusJumpKeys(t *testing.T) {
	a := newTestApp(t)

	a.handleKey(runeEvent('2'))
	if !a.tableFocused {
		t.Error("'2' must focus the packages table")
	}
	a.handleKey(runeEvent('1'))
	if a.tableFocused {
		t.Error("'1' must focus the sources tree")
	}
}

func TestFirstRunShowsTreeHintAndOnboarding(t *testing.T) {
	// Arrange — no paths registered, global scan finished empty.
	a := newTestApp(t)
	a.state[orchestrator.SourceGlobal].loading = false

	// Act
	a.refreshAll()

	// Assert — tree carries the hint entries.
	children := a.tree.GetRoot().GetChildren()
	if len(children) != 4 { // Global + spacer + two hint lines
		t.Fatalf("tree children = %d, want 4", len(children))
	}
	if got := children[2].GetText(); got != "No paths registered." {
		t.Errorf("hint line = %q", got)
	}

	// Assert — Packages panel shows onboarding, not "up to date".
	if got := a.detail.GetCell(0, 0).Text; got != "No project paths registered yet." {
		t.Errorf("onboarding first line = %q", got)
	}
}

func TestOnboardingNeverHidesGlobalResults(t *testing.T) {
	a := newTestApp(t)
	st := a.state[orchestrator.SourceGlobal]
	st.loading = false
	st.event = orchestrator.Event{
		Source:   orchestrator.SourceGlobal,
		Packages: []scanner.Package{{Name: "eslint", Current: "10.7.0", New: "10.8.0"}},
	}

	a.refreshAll()

	if got := a.detail.GetCell(0, 0).Text; got != "Package" {
		t.Errorf("global rows must win over onboarding, header = %q", got)
	}
}

func TestHintGoneOncePathRegistered(t *testing.T) {
	a := newTestApp(t)
	a.cfg = config.Config{Paths: []config.Path{{Path: "/tmp/project"}}}
	registerPath(a, "/tmp/project")

	a.refreshAll()

	for _, node := range a.tree.GetRoot().GetChildren() {
		if node.GetText() == "No paths registered." {
			t.Fatal("hint must disappear once a path is registered")
		}
	}
}

func TestSortAndFilterKeysScopedToTableViews(t *testing.T) {
	for _, r := range []rune{'s', '/'} {
		b, ok := lookupDispatch(runeEvent(r))
		if !ok {
			t.Fatalf("%q must be dispatchable", r)
		}
		if !b.activeIn(ctxTablePackages) || !b.activeIn(ctxTableVulns) {
			t.Errorf("%q must be active in both table views", r)
		}
		if b.activeIn(ctxTree) {
			t.Errorf("%q must not be active in the tree", r)
		}
	}
}

func TestCycleSortWraps(t *testing.T) {
	a := newTestApp(t)
	a.tableFocused = true

	a.handleKey(runeEvent('s'))
	if a.sortMode != sortSeverity {
		t.Errorf("first s = %v, want severity", a.sortMode)
	}
	a.handleKey(runeEvent('s'))
	a.handleKey(runeEvent('s'))
	if a.sortMode != sortScan {
		t.Errorf("cycle must wrap back to scan order, got %v", a.sortMode)
	}
}

func TestEscapePeelsFilterBeforeTree(t *testing.T) {
	// Arrange — focused table with an active filter.
	a := newTestApp(t)
	a.setTableFocus(true)
	a.filterText = "axi"

	// Act — first Esc clears the filter, keeps focus.
	a.handleKey(keyEvent(tcell.KeyEscape))
	if a.filterText != "" {
		t.Error("first Esc must clear the filter")
	}
	if !a.tableFocused {
		t.Error("first Esc must keep the table focused")
	}

	// Act — second Esc leaves the table.
	a.handleKey(keyEvent(tcell.KeyEscape))
	if a.tableFocused {
		t.Error("second Esc must return focus to the tree")
	}
}

func TestFilteredCommandStillUsesAllMarks(t *testing.T) {
	// Arrange — global source with two marked packages, filter hiding one.
	a := newTestApp(t)
	globalSel(a)
	st := a.state[orchestrator.SourceGlobal]
	st.loading = false
	st.event = orchestrator.Event{Source: orchestrator.SourceGlobal, Packages: []scanner.Package{
		{Name: "axios", Current: "0.21.1", New: "1.0.0"},
		{Name: "chalk", Current: "4.1.0", New: "5.0.0"},
	}}
	a.toggleMark("axios")
	a.toggleMark("chalk")
	a.filterText = "axi" // hides chalk

	// Act
	update, _ := a.currentCommands()

	// Assert — visibility must not narrow the command.
	if !strings.Contains(update, "chalk@5.0.0") {
		t.Errorf("command must include hidden marked package: %q", update)
	}
}

func TestRescanAllSkipsInFlightAndSweepsIdle(t *testing.T) {
	// Arrange — one source already scanning, two idle, no marks.
	a := newTestApp(t)
	a.state[orchestrator.SourceGlobal].loading = false
	registerPath(a, "/tmp/a")
	registerPath(a, "/tmp/b")
	a.state["/tmp/a"].loading = true

	// Act
	a.handleKey(runeEvent('R'))

	// Assert — idle sources launched, in-flight untouched, no confirm.
	if a.pages.HasPage(pageConfirm) {
		t.Error("sweep without marks must not ask")
	}
	if !a.state[orchestrator.SourceGlobal].loading || !a.state["/tmp/b"].loading {
		t.Error("idle sources must be rescanning")
	}
	if !a.state["/tmp/a"].loading {
		t.Error("in-flight source must keep scanning")
	}
}

func TestRescanAllWithMarksAsksOnce(t *testing.T) {
	a := newTestApp(t)
	a.state[orchestrator.SourceGlobal].loading = false
	registerPath(a, "/tmp/a")
	a.state["/tmp/a"].marks = map[int]map[string]bool{0: {"x": true}}

	a.handleKey(runeEvent('R'))

	if !a.pages.HasPage(pageConfirm) {
		t.Fatal("sweep over marks must ask first")
	}
	if a.state["/tmp/a"].loading {
		t.Error("the sweep must not start until the user confirms")
	}
}

func TestRescanAllAllScanningWarns(t *testing.T) {
	a := newTestApp(t) // global starts loading; no other sources

	a.handleKey(runeEvent('R'))

	if msg := a.statusMsg.GetText(true); !strings.Contains(msg, "already scanning") {
		t.Errorf("expected all-scanning warning, got %q", msg)
	}
}

func TestEscapeInTreeFoldsSelectedSource(t *testing.T) {
	// Arrange — a source with projects, expanded, selected.
	a := newTestApp(t)
	registerPath(a, "/tmp/a")
	a.state["/tmp/a"].loading = false
	a.state["/tmp/a"].event = orchestrator.Event{Source: "/tmp/a",
		Projects: []orchestrator.ProjectResult{{}}}

	// Act — Esc folds.
	a.handleKey(keyEvent(tcell.KeyEscape))
	if !a.state["/tmp/a"].collapsed {
		t.Fatal("Esc in the tree must collapse the selected source")
	}

	// Act — Esc again: nothing foldable → info hint, no silence.
	a.handleKey(keyEvent(tcell.KeyEscape))
	if msg := a.statusMsg.GetText(true); !strings.Contains(msg, "nothing to fold") {
		t.Errorf("expected fold hint, got %q", msg)
	}
}

func TestCopyUsesFullCommandDespiteTruncation(t *testing.T) {
	// Arrange — a command far beyond the bar budget at a narrow width.
	a := newTestApp(t)
	a.screenW = 60
	globalSel(a)
	st := a.state[orchestrator.SourceGlobal]
	st.loading = false
	pkgs := make([]scanner.Package, 30)
	for i := range pkgs {
		pkgs[i] = scanner.Package{Name: strings.Repeat("p", 10), Current: "1.0.0", New: "2.0.0"}
	}
	st.event = orchestrator.Event{Source: orchestrator.SourceGlobal, Packages: pkgs}
	a.refreshCommandBar()

	// Assert — bar shows the indicator; the copyable command is complete.
	if !strings.Contains(a.cmdBar.GetText(true), "… (c copies full)") {
		t.Error("oversized command must show the truncation indicator")
	}
	update, _ := a.currentCommands()
	if strings.Contains(update, "…") {
		t.Error("the copyable command must never be truncated")
	}
}
