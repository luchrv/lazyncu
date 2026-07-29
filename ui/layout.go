package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	pageMain    = "main"
	pageAddPath = "add-path"
	cmdBarRows  = 4
	modalWidth  = 60
	modalHeight = 3
)

func (a *App) buildLayout() {
	a.tree = tview.NewTreeView()
	root := tview.NewTreeNode("")
	a.tree.SetRoot(root).SetTopLevel(1)
	a.tree.SetBorder(true)
	a.tree.SetTitle(" 1 Sources ")
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
	a.detail.SetTitle(" 2 Packages ")
	a.detail.SetFixed(1, 0)
	a.detail.SetSelectable(false, false)

	a.cmdBar = tview.NewTextView().SetDynamicColors(true)
	a.cmdBar.SetBorder(true)
	a.cmdBar.SetTitle(" Command (copy with c) ")

	a.statusMsg = tview.NewTextView().SetDynamicColors(true)
	a.progress = tview.NewTextView().SetDynamicColors(true)
	a.helpBar = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignRight)

	right := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.detail, 0, 1, false).
		AddItem(a.cmdBar, cmdBarRows, 0, false)

	body := tview.NewFlex().
		AddItem(a.tree, 0, 1, true).
		AddItem(right, 0, 2, false)

	// Bottom line: transient messages on the left, scan progress in the
	// middle, permanent key help on the right — a message must never hide
	// the progress or the help. Progress and help zones are resized to
	// their content (refreshProgress / renderHelp).
	a.bottom = tview.NewFlex().
		AddItem(a.statusMsg, 0, 1, false).
		AddItem(a.progress, 0, 0, false).
		AddItem(a.helpBar, 0, 0, false)

	outer := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(body, 0, 1, true).
		AddItem(a.bottom, 1, 0, false)

	a.pages = tview.NewPages().AddPage(pageMain, outer, true, true)
	a.applyFocusStyle()
	a.tv.SetInputCapture(a.handleKey)
	// The terminal width drives the help variant; re-rendered on resize.
	a.tv.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		if w, _ := screen.Size(); w != a.screenW {
			a.screenW = w
			a.renderHelp()
		}
		return false
	})
}

// progressText renders the aggregate scan progress segment.
func progressText(done, total int) string {
	return fmt.Sprintf("[gray]scanning %d/%d[-]", done, total)
}

// refreshProgress shows `scanning N/M` in its own bottom-bar segment while
// any source is loading, and collapses the segment when idle — transient
// messages can never cover it.
func (a *App) refreshProgress() {
	done, total := a.scanProgress()
	if done == total {
		a.progress.SetText("")
		a.bottom.ResizeItem(a.progress, 0, 0)
		return
	}
	text := progressText(done, total)
	a.progress.SetText(text)
	a.bottom.ResizeItem(a.progress, printableWidth(text)+2, 0)
}

// applyFocusStyle highlights the border and title of the panel that owns
// the keyboard, so focus is visible at a glance and not only implied by
// the help bar. The command bar is never focusable and keeps defaults.
func (a *App) applyFocusStyle() {
	focused, blurred := a.tree.Box, a.detail.Box
	if a.tableFocused {
		focused, blurred = a.detail.Box, a.tree.Box
	}
	focused.SetBorderColor(tcell.ColorYellow).SetTitleColor(tcell.ColorYellow)
	blurred.SetBorderColor(tview.Styles.BorderColor).SetTitleColor(tview.Styles.TitleColor)
}

// renderHelp regenerates the help bar from the keymap for the current
// context, picking the widest variant the terminal width allows so the
// message zone always keeps room.
func (a *App) renderHelp() {
	ctx := a.currentContext()
	if ctx == ctxModal {
		return // the dashboard help is irrelevant while a modal is up
	}
	text, width := helpForWidth(ctx, a.screenW)
	a.helpBar.SetText(text)
	a.bottom.ResizeItem(a.helpBar, width, 0)
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
