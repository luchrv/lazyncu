package ui

import (
	"fmt"

	"github.com/rivo/tview"

	"github.com/luchrv/lazyncu/version"
)

const (
	pageAbout   = "about"
	aboutWidth  = 56
	aboutHeight = 12
)

// toggleAbout opens the About modal, or closes it when already open.
func (a *App) toggleAbout() {
	if a.pages.HasPage(pageAbout) {
		a.closeAbout()
		return
	}
	info := version.Get()
	text := fmt.Sprintf(
		"                    [yellow::b]lazyncu[-:-:-]\n\n"+
			" Version  %s\n"+
			" Commit   %s\n"+
			" Built    %s\n\n"+
			" Repo     https://github.com/luchrv/lazyncu\n"+
			" License  Apache-2.0\n\n"+
			"               [yellow]Esc[-] / [yellow]h[-] close",
		info.Version, info.Commit, info.Date)
	view := tview.NewTextView().SetDynamicColors(true)
	view.SetText(text)
	view.SetBorder(true)
	view.SetTitle(" About ")
	a.pages.AddPage(pageAbout, centered(view, aboutWidth, aboutHeight), true, true)
	a.tv.SetFocus(view)
}

func (a *App) closeAbout() {
	a.pages.RemovePage(pageAbout)
	a.tv.SetFocus(a.tree)
}
