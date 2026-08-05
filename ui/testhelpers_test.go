package ui

import (
	"context"
	"testing"

	"github.com/luchrv/lazyncu/config"
	"github.com/luchrv/lazyncu/orchestrator"
	"github.com/luchrv/lazyncu/scanner"
)

// fakeScanner satisfies orchestrator.Scanner with canned empty results so
// rescans triggered from tests never touch exec or the network.
type fakeScanner struct{}

func (fakeScanner) ScanGlobal(context.Context) ([]scanner.Package, error) { return nil, nil }
func (fakeScanner) ScanPath(context.Context, string) ([]scanner.Project, error) {
	return nil, nil
}

// newTestApp builds an App with the real layout but a fake scanner and a
// throwaway config path. The tview event loop is never started: handlers
// and widget state are exercised directly.
func newTestApp(t *testing.T) *App {
	t.Helper()
	return New(context.Background(), config.Config{}, t.TempDir()+"/config.toml",
		fakeScanner{}, nil, false, nil)
}

// registerPath adds a source to the app state the way a config entry would,
// bypassing config persistence.
func registerPath(a *App, path string) {
	a.order = append(a.order, path)
	a.state[path] = &sourceState{}
	a.sel = selection{source: path, projectIdx: -1}
}

// globalSel points the selection back at the global source.
func globalSel(a *App) {
	a.sel = selection{source: orchestrator.SourceGlobal, projectIdx: -1}
}
