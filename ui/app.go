// Package ui is the thin tview layer of the dashboard. It holds no business
// logic: it renders immutable results produced by the core packages and
// funnels every widget mutation from goroutines through one choke point.
package ui

import (
	"context"
	"fmt"

	"github.com/rivo/tview"

	"github.com/luchrv/lazyncu/config"
	"github.com/luchrv/lazyncu/orchestrator"
)

// sourceState tracks one source's lifecycle in the panel.
type sourceState struct {
	loading   bool
	collapsed bool
	event     orchestrator.Event
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

	pages      *tview.Pages
	tree       *tview.TreeView
	detail     *tview.Table
	cmdBar     *tview.TextView
	statusMsg  *tview.TextView
	helpBar    *tview.TextView
	showVulns  bool
	msgsHidden bool
	lastMsg    string
}

// New assembles the dashboard around an already-loaded config.
func New(ctx context.Context, cfg config.Config, cfgPath string,
	sc orchestrator.Scanner, auditor orchestrator.Auditor) *App {
	a := &App{
		tv:      tview.NewApplication(),
		ctx:     ctx,
		cfg:     cfg,
		cfgPath: cfgPath,
		sc:      sc,
		auditor: auditor,
		state:   map[string]*sourceState{},
		sel:     selection{source: orchestrator.SourceGlobal, projectIdx: -1},
	}
	a.order = append(a.order, orchestrator.SourceGlobal)
	a.state[orchestrator.SourceGlobal] = &sourceState{loading: true}
	for _, p := range cfg.Paths {
		a.order = append(a.order, p.Path)
		a.state[p.Path] = &sourceState{loading: true}
	}
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
	a.refreshAll()
}

func (a *App) refreshAll() {
	a.refreshTree()
	a.refreshDetail()
	a.refreshCommandBar()
}

// setStatus shows a transient message in the left status zone; the key help
// on the right stays untouched. The last message is remembered so toggling
// the zone back on restores it.
func (a *App) setStatus(format string, args ...any) {
	a.lastMsg = fmt.Sprintf(format, args...)
	if !a.msgsHidden {
		a.statusMsg.SetText(a.lastMsg)
	}
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
