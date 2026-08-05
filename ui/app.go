// Package ui is the thin tview layer of the dashboard. It holds no business
// logic: it renders immutable results produced by the core packages and
// funnels every widget mutation from goroutines through one choke point.
package ui

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/rivo/tview"

	"github.com/luchrv/lazyncu/config"
	"github.com/luchrv/lazyncu/orchestrator"
)

// sourceState tracks one source's lifecycle in the panel.
type sourceState struct {
	loading   bool
	collapsed bool
	event     orchestrator.Event
	// marks holds the packages selected for a filtered update command,
	// keyed by projectIdx (-1 = the source itself / global). In-memory
	// only: cleared on rescan, never persisted.
	marks map[int]map[string]bool
}

// selection identifies what the user has highlighted in the panel.
type selection struct {
	source string
	// projectIdx indexes Event.Projects; -1 selects the source itself
	// (meaningful for the global source).
	projectIdx int
}

// App wires the tview widgets to the orchestrator's event stream.
type App struct {
	tv      *tview.Application
	ctx     context.Context
	cfg     config.Config
	cfgPath string
	sc      orchestrator.Scanner
	auditor orchestrator.Auditor

	state map[string]*sourceState
	order []string
	sel   selection

	pages       *tview.Pages
	tree        *tview.TreeView
	detail      *tview.Table
	cmdBar      *tview.TextView
	statusMsg   *tview.TextView
	progress    *tview.TextView
	helpBar     *tview.TextView
	filterInput *tview.InputField
	bottom      *tview.Flex
	right       *tview.Flex
	// browser holds the add-path picker's widgets while its modal is open.
	browser *pathBrowser
	// firstRun is true when the config file was created by this launch; it
	// triggers the one-time welcome browser.
	firstRun bool
	// launch is the classified positional CLI argument (nil without one);
	// pendingProject defers selecting its project until the scan lands.
	launch         *Launch
	pendingProject *pendingSelect
	showVulns      bool
	msgsHidden     bool
	lastMsg        string
	// msgGen increments on every status message; expiry timers compare
	// against it so they never clear a newer message. expiresGen records
	// the last generation that scheduled an expiry (errors do not).
	msgGen     int
	expiresGen int
	// screenW is the last observed terminal width, used to pick the help
	// variant that still leaves the message zone room.
	screenW int
	// spinFrame advances on every spinner tick; spinning guards against
	// starting a second ticker goroutine.
	spinFrame int
	spinning  bool
	// sortMode and filterText shape the detail table; session-only.
	sortMode   sortMode
	filterText string

	// tableFocused: keyboard focus is on the Packages table (Tab toggles).
	tableFocused bool
	// rowPkgs maps rendered package-table rows (minus header) to package
	// names; nil while the vulnerability view is shown.
	rowPkgs []string
}

// currentMarks returns the mark set of the selected entry (nil when none).
func (a *App) currentMarks() map[string]bool {
	st, ok := a.state[a.sel.source]
	if !ok || st.marks == nil {
		return nil
	}
	return st.marks[a.sel.projectIdx]
}

// toggleMark flips one package's mark for the selected entry, rebuilding the
// inner set instead of mutating it in place.
func (a *App) toggleMark(name string) {
	st, ok := a.state[a.sel.source]
	if !ok {
		return
	}
	next := make(map[string]bool, len(st.marks[a.sel.projectIdx])+1)
	maps.Copy(next, st.marks[a.sel.projectIdx])
	if next[name] {
		delete(next, name)
	} else {
		next[name] = true
	}
	if st.marks == nil {
		st.marks = map[int]map[string]bool{}
	}
	st.marks[a.sel.projectIdx] = next
}

// clearMarks removes every mark of the selected entry.
func (a *App) clearMarks() {
	st, ok := a.state[a.sel.source]
	if !ok || st.marks == nil {
		return
	}
	delete(st.marks, a.sel.projectIdx)
}

// New assembles the dashboard around an already-loaded config. firstRun
// marks a launch that just created the config file, enabling the one-time
// welcome browser. l carries the classified positional CLI argument (nil
// when launched without one).
func New(ctx context.Context, cfg config.Config, cfgPath string,
	sc orchestrator.Scanner, auditor orchestrator.Auditor, firstRun bool, l *Launch) *App {
	a := &App{
		tv:       tview.NewApplication(),
		ctx:      ctx,
		cfg:      cfg,
		cfgPath:  cfgPath,
		sc:       sc,
		auditor:  auditor,
		state:    map[string]*sourceState{},
		sel:      selection{source: orchestrator.SourceGlobal, projectIdx: -1},
		firstRun: firstRun,
	}
	a.order = append(a.order, orchestrator.SourceGlobal)
	a.state[orchestrator.SourceGlobal] = &sourceState{loading: true}
	for _, p := range cfg.Paths {
		a.order = append(a.order, p.Path)
		a.state[p.Path] = &sourceState{loading: true}
	}
	a.applyLaunch(l)
	a.buildLayout()
	return a
}

// Run starts the parallel scan of every source and enters the UI loop.
func (a *App) Run() error {
	paths := make([]string, 0, len(a.cfg.Paths))
	for _, p := range a.cfg.Paths {
		paths = append(paths, p.Path)
	}
	a.consume(orchestrator.Run(a.ctx, a.sc, a.auditor, paths))

	a.refreshAll()
	a.maybeOfferConsolidation()
	a.maybeOpenWelcome()
	return a.tv.SetRoot(a.pages, true).EnableMouse(true).Run()
}

// consume is the single choke point for async UI updates: scan goroutines
// never touch widgets; their events are applied inside QueueUpdateDraw.
func (a *App) consume(events <-chan orchestrator.Event) {
	go func() {
		for ev := range events {
			a.tv.QueueUpdateDraw(func() { a.applyEvent(ev) })
		}
	}()
}

// scanOne (re)scans a single source — the global one or a registered path —
// through the same choke point as the launch fan-out.
func (a *App) scanOne(source string) {
	go func() {
		var ev orchestrator.Event
		if source == orchestrator.SourceGlobal {
			ev = orchestrator.RunGlobal(a.ctx, a.sc)
		} else {
			ev = orchestrator.RunOne(a.ctx, a.sc, a.auditor, source)
		}
		a.tv.QueueUpdateDraw(func() { a.applyEvent(ev) })
	}()
}

// applyEvent records one source result. Only ever called on the UI thread.
func (a *App) applyEvent(ev orchestrator.Event) {
	st, ok := a.state[ev.Source]
	if !ok {
		return // source removed while its scan was in flight
	}
	st.loading = false
	st.event = ev
	a.resolvePendingSelection(ev)
	a.refreshAll()
}

func (a *App) refreshAll() {
	a.refreshTree()
	a.refreshDetail()
	a.refreshCommandBar()
	a.refreshProgress()
	a.ensureSpinner()
}

// anyLoading reports whether any source's scan is still in flight.
func (a *App) anyLoading() bool {
	for _, st := range a.state {
		if st.loading {
			return true
		}
	}
	return false
}

// scanProgress counts finished sources against the total.
func (a *App) scanProgress() (done, total int) {
	total = len(a.order)
	for _, src := range a.order {
		if !a.state[src].loading {
			done++
		}
	}
	return done, total
}

// spinnerInterval is the spinner frame cadence.
const spinnerInterval = 120 * time.Millisecond

// ensureSpinner starts the single spinner goroutine when something is
// loading. Every frame is applied through the QueueUpdateDraw choke point;
// the goroutine stops itself once no source is loading.
func (a *App) ensureSpinner() {
	if a.spinning || !a.anyLoading() {
		return
	}
	a.spinning = true
	go func() {
		ticker := time.NewTicker(spinnerInterval)
		defer ticker.Stop()
		for range ticker.C {
			stopped := false
			a.tv.QueueUpdateDraw(func() {
				if !a.anyLoading() {
					a.spinning = false
					stopped = true
					return
				}
				a.spinFrame++
				a.refreshTree()
				a.refreshDetail()
			})
			if stopped {
				return
			}
		}
	}()
}

// msgLevel is the severity of a status message; it decides the icon and
// color in one place, so severity survives monochrome terminals.
type msgLevel int

const (
	msgInfo msgLevel = iota
	msgOK
	msgWarn
	msgError
)

// decorate prefixes the level icon and applies the level color.
func (l msgLevel) decorate(text string) string {
	switch l {
	case msgOK:
		return "[green]✓[-] " + text
	case msgWarn:
		return "[yellow]! " + text + "[-]"
	case msgError:
		return "[red]✗ " + text + "[-]"
	default:
		return "[gray]· " + text + "[-]"
	}
}

// msgTTL is how long non-error messages stay visible.
const msgTTL = 5 * time.Second

// setStatus shows a leveled message in the left status zone; the key help
// on the right stays untouched. Info/ok/warn messages expire after msgTTL
// under the generation guard; errors persist until replaced.
func (a *App) setStatus(level msgLevel, format string, args ...any) {
	a.msgGen++
	a.lastMsg = level.decorate(fmt.Sprintf(format, args...))
	if !a.msgsHidden {
		a.statusMsg.SetText(a.lastMsg)
	}
	if level == msgError {
		return
	}
	a.expiresGen = a.msgGen
	gen := a.msgGen
	time.AfterFunc(msgTTL, func() {
		a.tv.QueueUpdateDraw(func() { a.clearIfCurrent(gen) })
	})
}

// clearIfCurrent clears the status zone only while gen still identifies
// the visible message: an expired message must never erase a newer one,
// and an expired message is forgotten — `m` does not resurrect it.
func (a *App) clearIfCurrent(gen int) {
	if a.msgGen != gen {
		return
	}
	a.lastMsg = ""
	if !a.msgsHidden {
		a.statusMsg.SetText("")
	}
}

// currentContext derives the keymap context from existing UI state; there
// is no separate context field to keep in sync.
func (a *App) currentContext() keyContext {
	switch {
	case a.pages.HasPage(pageAbout) || a.pages.HasPage(pageConfirm) ||
		a.pages.HasPage(pageKeys) || a.pages.HasPage(pageAddPath):
		return ctxModal
	case a.tableFocused && a.showVulns:
		return ctxTableVulns
	case a.tableFocused:
		return ctxTablePackages
	default:
		return ctxTree
	}
}

// toggleVulns switches the detail panel between packages and
// vulnerabilities, moving the table between its two keymap contexts.
func (a *App) toggleVulns() {
	a.showVulns = !a.showVulns
	a.refreshDetail()
	a.renderHelp()
}

// cycleSort advances the detail-table order: scan → severity → name.
func (a *App) cycleSort() {
	a.sortMode = (a.sortMode + 1) % 3
	a.refreshDetail()
}

// escape peels one layer in whichever panel owns the keyboard.
func (a *App) escape() {
	if a.currentContext() == ctxTree {
		a.escapeTree()
		return
	}
	a.escapeTable()
}

// escapeTable peels one layer: an active filter clears first; without one,
// focus returns to the tree.
func (a *App) escapeTable() {
	if a.filterText != "" {
		a.clearFilter()
		return
	}
	a.setTableFocus(false)
}

// escapeTree collapses the selected source (a selected project folds its
// parent); when nothing is foldable it says so instead of staying silent.
func (a *App) escapeTree() {
	src := a.sel.source
	st, ok := a.state[src]
	if ok && len(st.event.Projects) > 0 && !st.collapsed {
		a.toggleFold(src)
		return
	}
	a.setStatus(msgInfo, "nothing to fold here")
}

// rescanAll sweeps every idle source, asking once when marks would be
// lost anywhere; in-flight sources are skipped by the overlap guard.
func (a *App) rescanAll() {
	idle := make([]string, 0, len(a.order))
	for _, src := range a.order {
		if !a.state[src].loading {
			idle = append(idle, src)
		}
	}
	if len(idle) == 0 {
		a.setStatus(msgWarn, "all sources are already scanning")
		return
	}
	if marks, projects := allSourcesMarks(a.state); marks > 0 {
		a.confirm(confirmRescanText("all sources", marks, projects),
			func() { a.doRescanAll(idle) })
		return
	}
	a.doRescanAll(idle)
}

// doRescanAll launches the sweep over the given idle sources.
func (a *App) doRescanAll(sources []string) {
	for _, src := range sources {
		st, ok := a.state[src]
		if !ok || st.loading {
			continue
		}
		st.loading = true
		st.marks = nil
		a.scanOne(src)
	}
	a.refreshAll()
	a.setStatus(msgInfo, "rescanning %d sources…", len(sources))
}

// clearFilter drops the filter and re-renders the full table.
func (a *App) clearFilter() {
	a.filterText = ""
	a.filterInput.SetText("")
	a.refreshDetail()
}

// clearMarksSelected clears the selected entry's marks and refreshes the
// views that render them.
func (a *App) clearMarksSelected() {
	a.clearMarks()
	a.refreshDetail()
	a.refreshCommandBar()
}

// toggleMessages hides or restores the status-message zone.
func (a *App) toggleMessages() {
	a.msgsHidden = !a.msgsHidden
	if a.msgsHidden {
		a.statusMsg.SetText("")
		return
	}
	a.statusMsg.SetText(a.lastMsg)
}

// markStats returns the selected entry's marked count and its total
// package count (before filtering) for the title indicator.
func (a *App) markStats() (marked, total int) {
	marked = len(a.currentMarks())
	st, ok := a.state[a.sel.source]
	if !ok {
		return marked, 0
	}
	if a.sel.source == orchestrator.SourceGlobal {
		return marked, len(st.event.Packages)
	}
	if pr, ok := a.selectedProject(); ok {
		return marked, len(pr.Packages)
	}
	return marked, 0
}
