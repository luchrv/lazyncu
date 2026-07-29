package ui

import (
	"strings"

	"github.com/rivo/tview"
)

const pageKeys = "keys"

// toggleKeys opens the keymap cheat-sheet modal, or closes it when open.
// The content is generated from the declarative keymap, so it can never
// drift from actual behavior.
func (a *App) toggleKeys() {
	if a.pages.HasPage(pageKeys) {
		a.closeKeys()
		return
	}
	text := keysText()
	view := tview.NewTextView().SetDynamicColors(true)
	view.SetText(text)
	view.SetBorder(true)
	view.SetTitle(" Keybindings ")

	lines := strings.Split(text, "\n")
	width := 0
	for _, l := range lines {
		width = max(width, printableWidth(l))
	}
	a.pages.AddPage(pageKeys, centered(view, width+4, len(lines)+2), true, true)
	a.tv.SetFocus(view)
}

func (a *App) closeKeys() {
	a.pages.RemovePage(pageKeys)
	if a.tableFocused {
		a.tv.SetFocus(a.detail)
		return
	}
	a.tv.SetFocus(a.tree)
}
