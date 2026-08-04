package ui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// browserFixture builds a directory tree for browser tests:
//
//	root/
//	  alpha/        (with one child: nested/)
//	  beta/
//	  .hidden/
//	  file.txt      (must never appear)
func browserFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"alpha/nested", "beta", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// --- listing helper ---

func TestListDirs(t *testing.T) {
	root := browserFixture(t)

	// Act
	visible := listDirs(root, false)
	withHidden := listDirs(root, true)

	// Assert
	if !slices.Equal(visible, []string{"alpha", "beta"}) {
		t.Errorf("listDirs(hidden off) = %v, want [alpha beta]", visible)
	}
	if !slices.Equal(withHidden, []string{".hidden", "alpha", "beta"}) {
		t.Errorf("listDirs(hidden on) = %v, want [.hidden alpha beta]", withHidden)
	}
}

func TestListDirsMissingAndUnreadable(t *testing.T) {
	if got := listDirs(filepath.Join(t.TempDir(), "nope"), false); got != nil {
		t.Errorf("missing dir must yield nil, got %v", got)
	}
	locked := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	if got := listDirs(locked, false); got != nil {
		t.Errorf("unreadable dir must yield nil, got %v", got)
	}
}

// --- modal behavior ---

// childByPath finds a direct child node whose reference is the given path.
func childByPath(node *tview.TreeNode, path string) *tview.TreeNode {
	for _, c := range node.GetChildren() {
		if ref, ok := c.GetReference().(*dirNode); ok && ref.path == path {
			return c
		}
	}
	return nil
}

func TestBrowserOpensAtRootInModalContext(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	root := browserFixture(t)

	// Act
	a.openBrowserAt(root)

	// Assert
	if !a.pages.HasPage(pageAddPath) {
		t.Fatal("openBrowserAt must show the add-path page")
	}
	if a.currentContext() != ctxModal {
		t.Error("the browser modal must put the app in ctxModal")
	}
	if got := a.browser.input.GetText(); got != root {
		t.Errorf("input = %q, want the root %q", got, root)
	}
	rootNode := a.browser.tree.GetRoot()
	if childByPath(rootNode, filepath.Join(root, "alpha")) == nil ||
		childByPath(rootNode, filepath.Join(root, "beta")) == nil {
		t.Error("root children must be loaded on open")
	}
	if childByPath(rootNode, filepath.Join(root, ".hidden")) != nil {
		t.Error("hidden dirs must not be listed by default")
	}
}

func TestBrowserSelectSyncsInput(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	root := browserFixture(t)
	a.openBrowserAt(root)
	alpha := childByPath(a.browser.tree.GetRoot(), filepath.Join(root, "alpha"))

	// Act
	a.browserSelect(alpha)

	// Assert
	if got := a.browser.input.GetText(); got != filepath.Join(root, "alpha") {
		t.Errorf("input after tree selection = %q, want alpha's path", got)
	}
}

func TestBrowserExpandAndCollapse(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	root := browserFixture(t)
	a.openBrowserAt(root)
	alpha := childByPath(a.browser.tree.GetRoot(), filepath.Join(root, "alpha"))
	a.browser.tree.SetCurrentNode(alpha)

	// Act — expand loads children lazily.
	a.handleModalKey(keyEvent(tcell.KeyRight))

	// Assert
	if !alpha.IsExpanded() {
		t.Fatal("→ must expand the current node")
	}
	if childByPath(alpha, filepath.Join(root, "alpha", "nested")) == nil {
		t.Error("expansion must load child directories")
	}

	// Act — collapse.
	a.handleModalKey(keyEvent(tcell.KeyLeft))
	if alpha.IsExpanded() {
		t.Error("← must collapse the current node")
	}
}

func TestBrowserEnterOnTreeAddsSelectedDir(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	root := browserFixture(t)
	a.openBrowserAt(root)
	alpha := childByPath(a.browser.tree.GetRoot(), filepath.Join(root, "alpha"))
	a.browser.tree.SetCurrentNode(alpha)
	a.browserSelect(alpha)

	// Act
	a.handleModalKey(keyEvent(tcell.KeyEnter))

	// Assert
	if a.pages.HasPage(pageAddPath) {
		t.Error("Enter must close the browser")
	}
	want := filepath.Join(root, "alpha")
	if len(a.cfg.Paths) != 1 || a.cfg.Paths[0].Path != want {
		t.Errorf("cfg.Paths = %+v, want [%s]", a.cfg.Paths, want)
	}
	if _, ok := a.state[want]; !ok {
		t.Error("the added path must start scanning")
	}
}

func TestBrowserInputEnterUsesTypedText(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	root := browserFixture(t)
	a.openBrowserAt(root)
	typed := filepath.Join(root, "beta")
	a.browser.input.SetText(typed)

	// Act — Enter through the InputField's own handler.
	handler := a.browser.input.InputHandler()
	handler(keyEvent(tcell.KeyEnter), func(p tview.Primitive) {})

	// Assert
	if a.pages.HasPage(pageAddPath) {
		t.Error("Enter on the input must close the browser")
	}
	if len(a.cfg.Paths) != 1 || a.cfg.Paths[0].Path != typed {
		t.Errorf("cfg.Paths = %+v, want [%s]", a.cfg.Paths, typed)
	}
}

func TestBrowserEscCancelsWithoutAdding(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	root := browserFixture(t)
	a.openBrowserAt(root)

	// Act
	a.handleModalKey(keyEvent(tcell.KeyEscape))

	// Assert
	if a.pages.HasPage(pageAddPath) {
		t.Error("Esc must close the browser")
	}
	if len(a.cfg.Paths) != 0 {
		t.Errorf("Esc must not add anything, got %+v", a.cfg.Paths)
	}
}

func TestBrowserHiddenToggle(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	root := browserFixture(t)
	a.openBrowserAt(root)
	rootNode := a.browser.tree.GetRoot()

	// Act — show hidden.
	a.handleModalKey(runeEvent('.'))

	// Assert
	if childByPath(rootNode, filepath.Join(root, ".hidden")) == nil {
		t.Fatal("'.' must reveal hidden directories")
	}

	// Act — hide again; expanded state of alpha survives the re-read.
	alpha := childByPath(rootNode, filepath.Join(root, "alpha"))
	a.browser.tree.SetCurrentNode(alpha)
	a.handleModalKey(keyEvent(tcell.KeyRight))
	a.handleModalKey(runeEvent('.'))
	if childByPath(rootNode, filepath.Join(root, ".hidden")) != nil {
		t.Error("'.' must hide dot directories again")
	}
	alpha = childByPath(rootNode, filepath.Join(root, "alpha"))
	if alpha == nil || !alpha.IsExpanded() {
		t.Error("hidden toggle must preserve expansion of surviving nodes")
	}
}

func TestBrowserDashboardKeysInert(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	root := browserFixture(t)
	a.openBrowserAt(root)
	a.tv.SetFocus(a.browser.tree)

	// Act & Assert — dashboard keys die at the modal boundary.
	for _, r := range []rune{'q', 'a', 'd', 'v', 'r', 'h'} {
		if got := a.handleKey(runeEvent(r)); got != nil {
			t.Errorf("%q must be swallowed while the browser is open", r)
		}
	}
	if a.pages.HasPage(pageConfirm) || a.pages.HasPage(pageAbout) {
		t.Error("dashboard keys acted underneath the browser")
	}
}
