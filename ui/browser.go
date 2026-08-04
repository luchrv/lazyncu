// The add-path browser: a hybrid picker modal pairing an editable path
// input with a lazy directory tree, so a scan source can be typed, pasted,
// or navigated to. Browsing only reads directory listings — the read-only
// invariant holds.
package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// browserWidthPct / browserHeightPct size the modal proportionally: the
// tree needs real estate, unlike the old one-line input.
const (
	browserWidthPct  = 60
	browserHeightPct = 70
)

// pathBrowser bundles the modal's widgets and view state for the lifetime
// of one open add-path dialog.
type pathBrowser struct {
	input      *tview.InputField
	tree       *tview.TreeView
	showHidden bool
}

// dirNode is a tree node's reference: its absolute path and whether its
// children have been read from disk (lazy loading happens on expand).
type dirNode struct {
	path   string
	loaded bool
}

// listDirs returns the names of dir's subdirectories, sorted; dot-prefixed
// entries are filtered unless showHidden. Any read error yields nil — an
// unreadable directory simply shows no children.
func listDirs(dir string, showHidden bool) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !showHidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// browserRoot picks the tree's starting directory: $HOME, or / when the
// home directory cannot be resolved.
func browserRoot() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "/"
	}
	return home
}

// openAddPath shows the add-path browser rooted at the home directory.
func (a *App) openAddPath() {
	a.openBrowserAt(browserRoot())
}

// openBrowserAt builds and shows the hybrid picker rooted at root.
func (a *App) openBrowserAt(root string) {
	b := &pathBrowser{
		input: tview.NewInputField().SetLabel("Path: ").SetFieldWidth(0),
		tree:  tview.NewTreeView(),
	}
	a.browser = b

	b.input.SetText(root)
	b.input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			text := b.input.GetText()
			a.closeAddPath()
			a.addPath(text)
		case tcell.KeyEscape:
			a.closeAddPath()
		case tcell.KeyTab, tcell.KeyBacktab:
			a.tv.SetFocus(b.tree)
		}
	})

	rootNode := tview.NewTreeNode(root).SetReference(&dirNode{path: root})
	b.tree.SetRoot(rootNode).SetCurrentNode(rootNode)
	b.tree.SetChangedFunc(func(node *tview.TreeNode) { a.browserSelect(node) })
	a.populateBrowserNode(rootNode)
	rootNode.SetExpanded(true)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(b.input, 1, 0, false).
		AddItem(b.tree, 0, 1, true)
	flex.SetBorder(true)
	flex.SetTitle(" Add path (Tab input/tree · →← expand · ↵ select · . hidden · Esc) ")

	a.pages.AddPage(pageAddPath, centeredPct(flex, browserWidthPct, browserHeightPct), true, true)
	a.tv.SetFocus(b.tree)
}

func (a *App) closeAddPath() {
	a.pages.RemovePage(pageAddPath)
	a.browser = nil
	a.tv.SetFocus(a.tree)
}

// browserSelect mirrors the tree selection into the input field — the
// one-way tree→input sync that keeps both Enter routes agreeing.
func (a *App) browserSelect(node *tview.TreeNode) {
	if node == nil || a.browser == nil {
		return
	}
	if ref, ok := node.GetReference().(*dirNode); ok {
		a.browser.input.SetText(ref.path)
	}
}

// populateBrowserNode reads a node's subdirectories from disk and rebuilds
// its children.
func (a *App) populateBrowserNode(node *tview.TreeNode) {
	ref, ok := node.GetReference().(*dirNode)
	if !ok {
		return
	}
	node.ClearChildren()
	for _, name := range listDirs(ref.path, a.browser.showHidden) {
		child := tview.NewTreeNode(name).
			SetReference(&dirNode{path: filepath.Join(ref.path, name)}).
			SetExpanded(false)
		node.AddChild(child)
	}
	ref.loaded = true
}

// refreshBrowserNode re-reads a loaded node's children (after the hidden
// toggle), preserving expansion and loaded state of surviving descendants.
func (a *App) refreshBrowserNode(node *tview.TreeNode) {
	ref, ok := node.GetReference().(*dirNode)
	if !ok || !ref.loaded {
		return
	}
	previous := map[string]*tview.TreeNode{}
	for _, c := range node.GetChildren() {
		if cref, ok := c.GetReference().(*dirNode); ok {
			previous[cref.path] = c
		}
	}
	a.populateBrowserNode(node)
	for _, c := range node.GetChildren() {
		cref := c.GetReference().(*dirNode)
		old, existed := previous[cref.path]
		if !existed {
			continue
		}
		if oldRef, ok := old.GetReference().(*dirNode); ok && oldRef.loaded {
			c.SetChildren(old.GetChildren())
			cref.loaded = true
			a.refreshBrowserNode(c)
		}
		c.SetExpanded(old.IsExpanded())
	}
}

// handleBrowserKey owns the modal's keys while the tree has focus: expand,
// collapse, confirm, cancel, focus swap, hidden toggle, and navigation
// passthrough. Everything else dies at the modal boundary.
func (a *App) handleBrowserKey(ev *tcell.EventKey) *tcell.EventKey {
	b := a.browser
	if b == nil {
		return nil
	}
	switch ev.Key() {
	case tcell.KeyEscape:
		a.closeAddPath()
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		a.tv.SetFocus(b.input)
		return nil
	case tcell.KeyEnter:
		if node := b.tree.GetCurrentNode(); node != nil {
			if ref, ok := node.GetReference().(*dirNode); ok {
				a.closeAddPath()
				a.addPath(ref.path)
			}
		}
		return nil
	case tcell.KeyRight:
		if node := b.tree.GetCurrentNode(); node != nil {
			if ref, ok := node.GetReference().(*dirNode); ok && !ref.loaded {
				a.populateBrowserNode(node)
			}
			node.SetExpanded(true)
		}
		return nil
	case tcell.KeyLeft:
		if node := b.tree.GetCurrentNode(); node != nil {
			node.SetExpanded(false)
		}
		return nil
	case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyHome, tcell.KeyEnd:
		return ev // the TreeView's own navigation
	}
	if ev.Key() == tcell.KeyRune && ev.Rune() == '.' {
		b.showHidden = !b.showHidden
		a.refreshBrowserNode(b.tree.GetRoot())
		return nil
	}
	return nil
}

// centeredPct wraps a primitive in a centered frame sized as a percentage
// of the screen — for modals that need proportional space, unlike the
// fixed-cell centered().
func centeredPct(p tview.Primitive, widthPct, heightPct int) tview.Primitive {
	side := (100 - widthPct) / 2
	topBottom := (100 - heightPct) / 2
	column := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, topBottom, false).
		AddItem(p, 0, heightPct, true).
		AddItem(nil, 0, topBottom, false)
	return tview.NewFlex().
		AddItem(nil, 0, side, false).
		AddItem(column, 0, widthPct, true).
		AddItem(nil, 0, side, false)
}
