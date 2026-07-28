package ui

import (
	"fmt"

	"github.com/rivo/tview"
)

const (
	pageConfirm   = "confirm"
	buttonCancel  = "Cancel"
	buttonConfirm = "Confirm"
)

// confirm shows a centered confirmation modal for a destructive action.
// Cancel is the default focus, Esc cancels, and the buttons are
// mouse-clickable. q is deliberately inert while it is open (see
// handleModalKey): a stray rune must never quit mid-question.
func (a *App) confirm(text string, onConfirm func()) {
	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{buttonCancel, buttonConfirm}).
		SetDoneFunc(func(_ int, label string) {
			a.closeConfirm()
			if label == buttonConfirm {
				onConfirm()
			}
		})
	a.pages.AddPage(pageConfirm, modal, true, true)
	a.tv.SetFocus(modal)
}

// closeConfirm dismisses the modal and restores focus to whichever panel
// had it before.
func (a *App) closeConfirm() {
	a.pages.RemovePage(pageConfirm)
	if a.tableFocused {
		a.tv.SetFocus(a.detail)
		return
	}
	a.tv.SetFocus(a.tree)
}

// markCount totals a source's marks across all of its projects, returning
// the mark total and how many projects hold at least one. Rescanning
// discards them all, so the confirmation must state the aggregate.
func markCount(st *sourceState) (marks, projects int) {
	for _, set := range st.marks {
		if len(set) == 0 {
			continue
		}
		marks += len(set)
		projects++
	}
	return marks, projects
}

// confirmRemoveText words the path-removal confirmation, making explicit
// that only the registration is dropped.
func confirmRemoveText(path string) string {
	return fmt.Sprintf("Stop tracking %s?\n\nThe folder on disk is not touched.", path)
}

// confirmRescanText words the mark-loss warning shown before rescanning a
// source with marked packages.
func confirmRescanText(name string, marks, projects int) string {
	return fmt.Sprintf("Rescanning %s discards %s across %s.",
		name, plural(marks, "mark"), plural(projects, "project"))
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
