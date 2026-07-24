package ui

import (
	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/luchrv/lazyncu/config"
	"github.com/luchrv/lazyncu/orchestrator"
)

// handleKey implements the global keybindings. It steps aside whenever a
// text input (the add-path modal) has focus.
func (a *App) handleKey(ev *tcell.EventKey) *tcell.EventKey {
	if _, editing := a.tv.GetFocus().(*tview.InputField); editing {
		return ev
	}
	switch ev.Key() {
	case tcell.KeyEscape:
		if a.pages.HasPage(pageAbout) {
			a.closeAbout()
			return nil
		}
		if a.tableFocused {
			a.setTableFocus(false)
			return nil
		}
	case tcell.KeyTab:
		if a.pages.HasPage(pageAbout) {
			return nil
		}
		a.setTableFocus(!a.tableFocused)
		return nil
	}
	switch ev.Rune() {
	case 'q':
		a.tv.Stop()
		return nil
	case 'c':
		a.copyCommand()
		return nil
	case 'v':
		a.showVulns = !a.showVulns
		a.refreshDetail()
		return nil
	case 'r':
		a.rescanSelected()
		return nil
	case 'm':
		a.toggleMessages()
		return nil
	case 'h':
		a.toggleAbout()
		return nil
	case 'a':
		a.openAddPath()
		return nil
	case 'd':
		a.removeSelectedPath()
		return nil
	case ' ':
		if a.tableFocused && !a.showVulns {
			a.toggleMarkUnderCursor()
			return nil
		}
	case 'x':
		if a.tableFocused && !a.showVulns {
			a.clearMarks()
			a.refreshDetail()
			a.refreshCommandBar()
			return nil
		}
	}
	return ev
}

// setTableFocus moves keyboard focus between the sources tree and the
// Packages table, adjusting row selectability and the contextual help.
func (a *App) setTableFocus(focused bool) {
	a.tableFocused = focused
	a.detail.SetSelectable(focused, false)
	if focused {
		a.tv.SetFocus(a.detail)
		if a.detail.GetRowCount() > 1 {
			a.detail.Select(1, 0)
		}
	} else {
		a.tv.SetFocus(a.tree)
	}
	a.refreshHelp()
}

// toggleMarkUnderCursor marks/unmarks the package on the selected table row
// and refreshes the views, keeping the cursor where it was.
func (a *App) toggleMarkUnderCursor() {
	row, _ := a.detail.GetSelection()
	idx := row - 1 // header occupies row 0
	if idx < 0 || idx >= len(a.rowPkgs) {
		return
	}
	a.toggleMark(a.rowPkgs[idx])
	a.refreshDetail()
	a.refreshCommandBar()
	a.detail.Select(row, 0)
}

// copyCommand puts the visible command on the clipboard: the fix command
// when the vulnerability view is active, the update command otherwise.
func (a *App) copyCommand() {
	update, fix := a.currentCommands()
	text := update
	if a.showVulns && fix != "" {
		text = fix
	}
	if text == "" {
		a.setStatus("nothing to copy here")
		return
	}
	if err := clipboard.WriteAll(text); err != nil {
		a.setStatus("[red]clipboard unavailable (%v) — copy manually from the command bar[-]", err)
		return
	}
	a.setStatus("[green]copied:[-] %s", tview.Escape(text))
}

// toggleFold collapses or expands a source's project list. Collapsing moves
// a selection that pointed at a now-hidden project up to the source itself.
func (a *App) toggleFold(src string) {
	st, ok := a.state[src]
	if !ok || len(st.event.Projects) == 0 {
		return
	}
	st.collapsed = !st.collapsed
	if st.collapsed && a.sel.source == src {
		a.sel = selection{source: src, projectIdx: -1}
	}
	a.refreshTree()
}

// rescanSelected rescans the selected source. Disabled while that source is
// already scanning: the guard prevents overlapping scans of the same source.
func (a *App) rescanSelected() {
	src := a.sel.source
	st, ok := a.state[src]
	if !ok {
		return
	}
	if st.loading {
		a.setStatus("[yellow]%s is still scanning — rescan is disabled until it finishes[-]", displayName(src))
		return
	}
	st.loading = true
	st.marks = nil // fresh scan invalidates the selection
	a.scanOne(src)
	a.refreshAll()
	a.setStatus("rescanning %s…", displayName(src))
}

func displayName(source string) string {
	if source == orchestrator.SourceGlobal {
		return "global packages"
	}
	return source
}

// openAddPath shows the add-path modal input.
func (a *App) openAddPath() {
	input := tview.NewInputField().SetLabel("Path to add: ").SetFieldWidth(0)
	input.SetBorder(true)
	input.SetDoneFunc(func(key tcell.Key) {
		defer a.closeAddPath()
		if key != tcell.KeyEnter {
			return
		}
		a.addPath(input.GetText())
	})
	a.pages.AddPage(pageAddPath, centered(input, modalWidth, modalHeight), true, true)
	a.tv.SetFocus(input)
}

func (a *App) closeAddPath() {
	a.pages.RemovePage(pageAddPath)
	a.tv.SetFocus(a.tree)
}

// addPath validates through the config store, persists immediately, and
// scans only the new source.
func (a *App) addPath(raw string) {
	if raw == "" {
		return
	}
	updated, err := a.cfg.AddPath(raw)
	if err != nil {
		a.setStatus("[red]%v[-]", err)
		return
	}
	if err := config.Save(a.cfgPath, updated); err != nil {
		a.setStatus("[red]could not save config: %v[-]", err)
		return
	}
	a.cfg = updated
	added := updated.Paths[len(updated.Paths)-1].Path
	a.order = append(a.order, added)
	a.state[added] = &sourceState{loading: true}
	a.scanOne(added)
	a.refreshAll()
	a.setStatus("[green]added %s — scanning[-]", added)
}

// removeSelectedPath unregisters the selected source (never the global one).
func (a *App) removeSelectedPath() {
	src := a.sel.source
	if src == orchestrator.SourceGlobal {
		a.setStatus("the global source cannot be removed")
		return
	}
	updated := a.cfg.RemovePath(src)
	if err := config.Save(a.cfgPath, updated); err != nil {
		a.setStatus("[red]could not save config: %v[-]", err)
		return
	}
	a.cfg = updated
	delete(a.state, src)
	a.order = removeString(a.order, src)
	a.sel = selection{source: orchestrator.SourceGlobal, projectIdx: -1}
	a.refreshAll()
	a.setStatus("removed %s", src)
}

func removeString(list []string, target string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		if s != target {
			out = append(out, s)
		}
	}
	return out
}
