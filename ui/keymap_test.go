package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func runeEvent(r rune) *tcell.EventKey {
	return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
}

func keyEvent(k tcell.Key) *tcell.EventKey {
	return tcell.NewEventKey(k, 0, tcell.ModNone)
}

func TestLookupDispatchFindsBoundRune(t *testing.T) {
	// Arrange
	ev := runeEvent('d')

	// Act
	b, ok := lookupDispatch(ev)

	// Assert
	if !ok {
		t.Fatal("expected 'd' to be a dispatchable binding")
	}
	if b.do == nil {
		t.Fatal("dispatch binding must carry an action")
	}
	if !b.activeIn(ctxTree) {
		t.Error("'d' must be active in the tree context")
	}
	if b.activeIn(ctxTablePackages) {
		t.Error("'d' must not be active in the packages table")
	}
	if b.activeIn(ctxTableVulns) {
		t.Error("'d' must not be active in the vulnerability view")
	}
}

func TestLookupDispatchIgnoresUnboundKeys(t *testing.T) {
	if _, ok := lookupDispatch(runeEvent('z')); ok {
		t.Error("'z' is unbound and must not resolve")
	}
	if _, ok := lookupDispatch(keyEvent(tcell.KeyF5)); ok {
		t.Error("F5 is unbound and must not resolve")
	}
}

func TestLookupDispatchSkipsHelpOnlyRows(t *testing.T) {
	// Enter is help-only (the tree widget folds); it must not dispatch.
	if _, ok := lookupDispatch(keyEvent(tcell.KeyEnter)); ok {
		t.Error("Enter is help-only and must not resolve to an action")
	}
}

func TestGlobalKeysActiveEverywhere(t *testing.T) {
	contexts := []keyContext{ctxTree, ctxTablePackages, ctxTableVulns}
	for _, r := range []rune{'q', 'c', 'v', 'r', 'm', 'h'} {
		b, ok := lookupDispatch(runeEvent(r))
		if !ok {
			t.Fatalf("%q must be dispatchable", r)
		}
		for _, ctx := range contexts {
			if !b.activeIn(ctx) {
				t.Errorf("%q must be active in context %d", r, ctx)
			}
		}
		if b.activeIn(ctxModal) {
			t.Errorf("%q must not be active in the modal context", r)
		}
	}
}

func TestTableKeysScopedToPackagesView(t *testing.T) {
	for _, r := range []rune{' ', 'x'} {
		b, ok := lookupDispatch(runeEvent(r))
		if !ok {
			t.Fatalf("%q must be dispatchable", r)
		}
		if !b.activeIn(ctxTablePackages) {
			t.Errorf("%q must be active in the packages table", r)
		}
		if b.activeIn(ctxTableVulns) {
			t.Errorf("%q must not be active in the vulnerability view", r)
		}
		if b.activeIn(ctxTree) {
			t.Errorf("%q must not be active in the tree", r)
		}
	}
}

func TestHintForTreeKeyPressedInTable(t *testing.T) {
	b, _ := lookupDispatch(runeEvent('d'))

	got := hintFor(b, ctxTablePackages)

	want := "d only works in Sources — Tab to go back"
	if got != want {
		t.Errorf("hint = %q, want %q", got, want)
	}
}

func TestHintForMarkKeyPressedInVulnsView(t *testing.T) {
	b, _ := lookupDispatch(runeEvent(' '))

	got := hintFor(b, ctxTableVulns)

	want := "␣ only works in the packages view — v to go back"
	if got != want {
		t.Errorf("hint = %q, want %q", got, want)
	}
}

func TestHintForTableKeyPressedInTree(t *testing.T) {
	b, _ := lookupDispatch(runeEvent('x'))

	got := hintFor(b, ctxTree)

	want := "x only works in the packages table — Tab to switch"
	if got != want {
		t.Errorf("hint = %q, want %q", got, want)
	}
}

func TestHelpTextTreeContext(t *testing.T) {
	help := helpText(ctxTree, false)

	for _, want := range []string{
		"[yellow]q[-] quit", "[yellow]c[-] copy cmd", "[yellow]v[-] vulns",
		"[yellow]r[-] rescan", "[yellow]a[-] add path", "[yellow]d[-] del path",
		"[yellow]↵[-] fold", "[yellow]m[-] msgs", "[yellow]h[-] about",
		"[yellow]Tab[-] pkgs",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("tree help missing %q in %q", want, help)
		}
	}
	for _, forbidden := range []string{"mark", "clear", "move"} {
		if strings.Contains(help, forbidden) {
			t.Errorf("tree help must not advertise table binding %q", forbidden)
		}
	}
}

func TestHelpTextPackagesTableContext(t *testing.T) {
	help := helpText(ctxTablePackages, false)

	for _, want := range []string{
		"[yellow]q[-] quit", "[yellow]↑↓[-] move", "[yellow]␣[-] mark",
		"[yellow]x[-] clear", "[yellow]c[-] copy cmd",
		"[yellow]Tab[-]/[yellow]Esc[-] back",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("table help missing %q in %q", want, help)
		}
	}
	for _, forbidden := range []string{"add path", "del path", "fold"} {
		if strings.Contains(help, forbidden) {
			t.Errorf("table help must not advertise tree binding %q", forbidden)
		}
	}
}

func TestHelpTextVulnsContextExcludesMarking(t *testing.T) {
	help := helpText(ctxTableVulns, false)

	for _, forbidden := range []string{"mark", "clear"} {
		if strings.Contains(help, forbidden) {
			t.Errorf("vulns help must not advertise %q in %q", forbidden, help)
		}
	}
	if !strings.Contains(help, "[yellow]↑↓[-] move") {
		t.Error("vulns help must keep the move hint")
	}
}

func TestHelpTextCompactIsSubset(t *testing.T) {
	full := helpText(ctxTree, false)
	compact := helpText(ctxTree, true)

	if printableWidth(compact) >= printableWidth(full) {
		t.Errorf("compact (%d cols) must be narrower than full (%d cols)",
			printableWidth(compact), printableWidth(full))
	}
	if !strings.Contains(compact, "[yellow]q[-] quit") {
		t.Error("compact help must keep quit")
	}
	if !strings.Contains(compact, "[yellow]h[-] about") {
		t.Error("compact help must keep the about entry (hosts the legend)")
	}
}

func TestPrintableWidthStripsColorTags(t *testing.T) {
	got := printableWidth("[yellow]q[-] quit")

	if got != len([]rune("q quit")) {
		t.Errorf("printableWidth = %d, want %d", got, len([]rune("q quit")))
	}
}

func TestHelpForWidthPicksVariant(t *testing.T) {
	full := helpText(ctxTree, false)
	wide := printableWidth(full) + minMessageZone + 10

	text, w := helpForWidth(ctxTree, wide)
	if text != full {
		t.Error("wide terminal must get the full help variant")
	}
	if w != printableWidth(full) {
		t.Errorf("reported width = %d, want %d", w, printableWidth(full))
	}

	text, _ = helpForWidth(ctxTree, 60)
	if text != helpText(ctxTree, true) {
		t.Error("narrow terminal must get the compact help variant")
	}
}

func TestCurrentContextResolution(t *testing.T) {
	a := newTestApp(t)

	if got := a.currentContext(); got != ctxTree {
		t.Errorf("default context = %d, want tree", got)
	}

	a.tableFocused = true
	if got := a.currentContext(); got != ctxTablePackages {
		t.Errorf("focused table context = %d, want packages", got)
	}

	a.showVulns = true
	if got := a.currentContext(); got != ctxTableVulns {
		t.Errorf("focused vulns context = %d, want vulns", got)
	}

	a.toggleAbout()
	if got := a.currentContext(); got != ctxModal {
		t.Errorf("about-open context = %d, want modal", got)
	}
	a.closeAbout()

	a.confirm("really?", func() {})
	if got := a.currentContext(); got != ctxModal {
		t.Errorf("confirm-open context = %d, want modal", got)
	}
}

func TestBarHiddenRowsExcludedFromHelpBar(t *testing.T) {
	for _, ctx := range []keyContext{ctxTree, ctxTablePackages, ctxTableVulns} {
		help := helpText(ctx, false)
		for _, forbidden := range []string{"focus sources", "focus packages"} {
			if strings.Contains(help, forbidden) {
				t.Errorf("bar must not advertise bar-hidden binding %q (ctx %d)", forbidden, ctx)
			}
		}
	}
}

func TestHelpKeysAreGlobalDispatch(t *testing.T) {
	for _, r := range []rune{'?', '1', '2'} {
		b, ok := lookupDispatch(runeEvent(r))
		if !ok {
			t.Fatalf("%q must be dispatchable", r)
		}
		for _, ctx := range []keyContext{ctxTree, ctxTablePackages, ctxTableVulns} {
			if !b.activeIn(ctx) {
				t.Errorf("%q must be active in context %d", r, ctx)
			}
		}
	}
}

func TestHelpBarAdvertisesQuestionMark(t *testing.T) {
	if !strings.Contains(helpText(ctxTree, true), "[yellow]?[-] help") {
		t.Error("compact help must advertise the cheat sheet")
	}
}

func TestKeysTextGroupsAndLegend(t *testing.T) {
	// Assert on the printable text — what the user actually sees.
	text := colorTag.ReplaceAllString(keysText(), "")

	for _, want := range []string{
		"Global", "Sources", "Packages", // group headers
		"quit", "help", "focus sources", "focus packages", // globals incl. bar-hidden
		"add path", "del path", "fold", // tree group
		"mark", "clear", "move", "back", // table group
		"Add path", "input/tree", "select folder", "hidden folders", // browser group
		"▲ major", "● minor", "▪ patch", // legend
		"C critical", "H high", "M moderate", "L low",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("cheat sheet missing %q", want)
		}
	}
	if strings.Index(text, "Global") > strings.Index(text, "Sources") {
		t.Error("Global group must come before Sources")
	}
}
