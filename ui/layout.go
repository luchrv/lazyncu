package ui

import (
	"github.com/rivo/tview"
)

const (
	pageMain    = "main"
	pageAddPath = "add-path"
	// statusHelp uses [yellow]…[-] tags: bare key letters like "[q]" would be
	// swallowed as color tags by tview's dynamic-colors parser.
	statusHelp  = "[yellow]q[-] quit  [yellow]c[-] copy cmd  [yellow]v[-] vulns  [yellow]r[-] rescan  [yellow]a[-] add path  [yellow]d[-] del path  [yellow]↵[-] fold  [yellow]m[-] msgs"
	helpWidth   = 86
	cmdBarRows  = 4
	modalWidth  = 60
	modalHeight = 3
)

func (a *App) buildLayout() {
	a.tree = tview.NewTreeView()
	root := tview.NewTreeNode("")
	a.tree.SetRoot(root).SetTopLevel(1)
	a.tree.SetBorder(true)
	a.tree.SetTitle(" Sources ")
	a.tree.SetChangedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}
		if ref, ok := node.GetReference().(selection); ok {
			a.sel = ref
			a.refreshDetail()
			a.refreshCommandBar()
		}
	})
	// Enter on a source node folds/unfolds its project list.
	a.tree.SetSelectedFunc(func(node *tview.TreeNode) {
		if ref, ok := node.GetReference().(selection); ok && ref.projectIdx < 0 {
			a.toggleFold(ref.source)
		}
	})

	a.detail = tview.NewTable()
	a.detail.SetBorder(true)
	a.detail.SetTitle(" Packages ")
	a.detail.SetFixed(1, 0)
	a.detail.SetSelectable(false, false)

	a.cmdBar = tview.NewTextView().SetDynamicColors(true)
	a.cmdBar.SetBorder(true)
	a.cmdBar.SetTitle(" Command (copy with c) ")

	a.statusMsg = tview.NewTextView().SetDynamicColors(true)
	a.helpBar = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignRight)
	a.helpBar.SetText(statusHelp)

	right := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.detail, 0, 1, false).
		AddItem(a.cmdBar, cmdBarRows, 0, false)

	body := tview.NewFlex().
		AddItem(a.tree, 0, 1, true).
		AddItem(right, 0, 2, false)

	// Bottom line: transient messages on the left, permanent key help on the
	// right — a message must never hide the help.
	bottom := tview.NewFlex().
		AddItem(a.statusMsg, 0, 1, false).
		AddItem(a.helpBar, helpWidth, 0, false)

	outer := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(body, 0, 1, true).
		AddItem(bottom, 1, 0, false)

	a.pages = tview.NewPages().AddPage(pageMain, outer, true, true)
	a.tv.SetInputCapture(a.handleKey)
}

// centered wraps a primitive in a fixed-size centered modal frame.
func centered(p tview.Primitive, width, height int) tview.Primitive {
	column := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(p, height, 0, true).
		AddItem(nil, 0, 1, false)
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(column, width, 0, true).
		AddItem(nil, 0, 1, false)
}
