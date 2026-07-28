// The declarative keymap is the single source of truth for keybindings:
// dispatch, the contextual help bar, and out-of-context teaching hints are
// all derived from one table, so the help can never drift from behavior.
package ui

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// keyContext identifies which UI area currently owns the keyboard.
type keyContext int

const (
	ctxTree keyContext = iota
	ctxTablePackages
	ctxTableVulns
	ctxModal
)

var (
	allPanels    = []keyContext{ctxTree, ctxTablePackages, ctxTableVulns}
	tableViews   = []keyContext{ctxTablePackages, ctxTableVulns}
	treeOnly     = []keyContext{ctxTree}
	packagesOnly = []keyContext{ctxTablePackages}
)

// binding is one keymap row. Rows with a nil do are help-only (the widget
// itself implements the behavior, e.g. Enter folding via the tree); rows
// with an empty desc are dispatch-only and never appear in help or hints.
type binding struct {
	r        rune         // rune key; 0 when key drives the match
	key      tcell.Key    // special key; ignored when r != 0
	label    string       // key label rendered in the help bar
	desc     string       // action label; "" hides the row from help
	contexts []keyContext // contexts where the binding is active
	compact  bool         // kept in the compact help variant
	do       func(a *App) // action; nil for help-only rows
}

// keymap lists every binding in help-bar order. Labels use the
// [yellow]…[-] tag convention: bare "[q]" literals would be swallowed as
// color tags by tview's dynamic-colors parser. Populated in init: the
// action closures reference App methods that read the keymap back (help
// generation), which a plain var initializer would reject as a cycle.
var keymap []binding

func init() {
	keymap = []binding{
		{r: 'q', label: "q", desc: "quit", contexts: allPanels, compact: true,
			do: func(a *App) { a.tv.Stop() }},
		{key: tcell.KeyUp, label: "↑↓", desc: "move", contexts: tableViews}, // widget navigates
		{r: ' ', label: "␣", desc: "mark", contexts: packagesOnly,
			do: func(a *App) { a.toggleMarkUnderCursor() }},
		{r: 'x', label: "x", desc: "clear", contexts: packagesOnly,
			do: func(a *App) { a.clearMarksSelected() }},
		{r: 'c', label: "c", desc: "copy cmd", contexts: allPanels, compact: true,
			do: func(a *App) { a.copyCommand() }},
		{r: 'v', label: "v", desc: "vulns", contexts: allPanels,
			do: func(a *App) { a.toggleVulns() }},
		{r: 'r', label: "r", desc: "rescan", contexts: allPanels,
			do: func(a *App) { a.rescanSelected() }},
		{r: 'a', label: "a", desc: "add path", contexts: treeOnly,
			do: func(a *App) { a.openAddPath() }},
		{r: 'd', label: "d", desc: "del path", contexts: treeOnly,
			do: func(a *App) { a.removeSelectedPath() }},
		{key: tcell.KeyEnter, label: "↵", desc: "fold", contexts: treeOnly}, // tree widget folds
		{r: 'm', label: "m", desc: "msgs", contexts: allPanels,
			do: func(a *App) { a.toggleMessages() }},
		{r: 'h', label: "h", desc: "about", contexts: allPanels, compact: true,
			do: func(a *App) { a.toggleAbout() }},
		{key: tcell.KeyTab, label: "Tab", desc: "pkgs", contexts: treeOnly, compact: true}, // tree wording
		{key: tcell.KeyTab, label: "Tab[-]/[yellow]Esc", desc: "back", contexts: tableViews, // table wording
			compact: true},
		{key: tcell.KeyTab, contexts: allPanels, // dispatch row behind both Tab help rows
			do: func(a *App) { a.setTableFocus(!a.tableFocused) }},
		{key: tcell.KeyEscape, contexts: tableViews,
			do: func(a *App) { a.setTableFocus(false) }},
	}
}

// activeIn reports whether the binding fires in the given context.
func (b binding) activeIn(ctx keyContext) bool {
	return slices.Contains(b.contexts, ctx)
}

// matches reports whether the key event hits this binding's key.
func (b binding) matches(ev *tcell.EventKey) bool {
	if b.r != 0 {
		return ev.Key() == tcell.KeyRune && ev.Rune() == b.r
	}
	return ev.Key() == b.key
}

// lookupDispatch resolves a key event to its actionable binding, skipping
// help-only rows.
func lookupDispatch(ev *tcell.EventKey) (binding, bool) {
	for _, b := range keymap {
		if b.do != nil && b.matches(ev) {
			return b, true
		}
	}
	return binding{}, false
}

// hintFor words the teaching hint shown when a bound key is pressed in a
// context it does not belong to.
func hintFor(b binding, cur keyContext) string {
	if len(b.contexts) == 0 || b.desc == "" {
		return ""
	}
	switch target := b.contexts[0]; {
	case target == ctxTree:
		return fmt.Sprintf("%s only works in Sources — Tab to go back", b.label)
	case target == ctxTablePackages && cur == ctxTableVulns:
		return fmt.Sprintf("%s only works in the packages view — v to go back", b.label)
	case target == ctxTablePackages:
		return fmt.Sprintf("%s only works in the packages table — Tab to switch", b.label)
	}
	return ""
}

// helpText renders the help-bar content for a context, full or compact.
func helpText(ctx keyContext, compact bool) string {
	parts := make([]string, 0, len(keymap))
	for _, b := range keymap {
		if b.desc == "" || !b.activeIn(ctx) {
			continue
		}
		if compact && !b.compact {
			continue
		}
		parts = append(parts, fmt.Sprintf("[yellow]%s[-] %s", b.label, b.desc))
	}
	return strings.Join(parts, "  ")
}

// minMessageZone is the column count always reserved for status messages
// next to the help bar.
const minMessageZone = 24

var colorTag = regexp.MustCompile(`\[[a-zA-Z:\-]*\]`)

// printableWidth measures a help string in terminal columns, ignoring
// tview color tags.
func printableWidth(s string) int {
	return len([]rune(colorTag.ReplaceAllString(s, "")))
}

// helpForWidth picks the widest help variant that still leaves the message
// zone room, returning the text and its printable width.
func helpForWidth(ctx keyContext, screen int) (string, int) {
	full := helpText(ctx, false)
	if w := printableWidth(full); screen-w >= minMessageZone {
		return full, w
	}
	compact := helpText(ctx, true)
	return compact, printableWidth(compact)
}
