package ui

import (
	"context"
	"errors"
	"testing"

	"github.com/luchrv/lazyncu/config"
	"github.com/luchrv/lazyncu/orchestrator"
	"github.com/luchrv/lazyncu/scanner"
)

// projectWithDir builds the minimal scanner.Project a pending selection
// compares against.
func projectWithDir(dir string) scanner.Project {
	return scanner.Project{Dir: dir}
}

// newLaunchApp builds an App over cfg with the given launch intent, using
// the same fake scanner as newTestApp.
func newLaunchApp(t *testing.T, cfg config.Config, l *Launch) *App {
	t.Helper()
	return New(context.Background(), cfg, t.TempDir()+"/config.toml",
		fakeScanner{}, nil, false, l)
}

func pathsConfig(paths ...string) config.Config {
	cfg := config.Config{}
	for _, p := range paths {
		cfg.Paths = append(cfg.Paths, config.Path{Path: p})
	}
	return cfg
}

func TestApplyLaunchSelectsSource(t *testing.T) {
	cfg := pathsConfig("/projects", "/other")

	a := newLaunchApp(t, cfg, &Launch{Source: "/projects"})

	expected := selection{source: "/projects", projectIdx: -1}
	if a.sel != expected {
		t.Errorf("sel = %+v, want %+v", a.sel, expected)
	}
	if a.pendingProject != nil {
		t.Errorf("pendingProject = %+v, want nil without ProjectDir", a.pendingProject)
	}
}

func TestApplyLaunchNilKeepsDefaultSelection(t *testing.T) {
	a := newLaunchApp(t, pathsConfig("/projects"), nil)

	expected := selection{source: orchestrator.SourceGlobal, projectIdx: -1}
	if a.sel != expected {
		t.Errorf("sel = %+v, want %+v", a.sel, expected)
	}
}

func TestApplyLaunchUnknownSourceIsIgnored(t *testing.T) {
	a := newLaunchApp(t, pathsConfig("/projects"), &Launch{Source: "/not-registered"})

	expected := selection{source: orchestrator.SourceGlobal, projectIdx: -1}
	if a.sel != expected {
		t.Errorf("sel = %+v, want %+v", a.sel, expected)
	}
}

func TestPendingSelectionMatchesProject(t *testing.T) {
	cfg := pathsConfig("/projects")
	a := newLaunchApp(t, cfg, &Launch{Source: "/projects", ProjectDir: "/projects/api"})

	a.applyEvent(orchestrator.Event{
		Source: "/projects",
		Projects: []orchestrator.ProjectResult{
			{Project: projectWithDir("/projects/web")},
			{Project: projectWithDir("/projects/api")},
		},
	})

	expected := selection{source: "/projects", projectIdx: 1}
	if a.sel != expected {
		t.Errorf("sel = %+v, want %+v", a.sel, expected)
	}
	if a.pendingProject != nil {
		t.Errorf("pendingProject = %+v, want cleared", a.pendingProject)
	}
}

func TestPendingSelectionNoMatchKeepsSource(t *testing.T) {
	cfg := pathsConfig("/projects")
	a := newLaunchApp(t, cfg, &Launch{Source: "/projects", ProjectDir: "/projects/api"})

	a.applyEvent(orchestrator.Event{
		Source:   "/projects",
		Projects: []orchestrator.ProjectResult{{Project: projectWithDir("/projects/web")}},
	})

	expected := selection{source: "/projects", projectIdx: -1}
	if a.sel != expected {
		t.Errorf("sel = %+v, want %+v", a.sel, expected)
	}
	if a.pendingProject != nil {
		t.Errorf("pendingProject = %+v, want cleared after its source's event", a.pendingProject)
	}
}

func TestPendingSelectionScanErrorKeepsSource(t *testing.T) {
	cfg := pathsConfig("/projects")
	a := newLaunchApp(t, cfg, &Launch{Source: "/projects", ProjectDir: "/projects/api"})

	a.applyEvent(orchestrator.Event{Source: "/projects", Err: errors.New("scan failed")})

	expected := selection{source: "/projects", projectIdx: -1}
	if a.sel != expected {
		t.Errorf("sel = %+v, want %+v", a.sel, expected)
	}
	if a.pendingProject != nil {
		t.Errorf("pendingProject = %+v, want cleared", a.pendingProject)
	}
}

func TestPendingSelectionIgnoresOtherSources(t *testing.T) {
	cfg := pathsConfig("/projects", "/other")
	a := newLaunchApp(t, cfg, &Launch{Source: "/projects", ProjectDir: "/projects/api"})

	a.applyEvent(orchestrator.Event{
		Source:   "/other",
		Projects: []orchestrator.ProjectResult{{Project: projectWithDir("/projects/api")}},
	})

	if a.pendingProject == nil {
		t.Errorf("pendingProject cleared by an unrelated source's event")
	}
}

func TestConsolidateChildrenRemovesAndPersists(t *testing.T) {
	cfg := pathsConfig("/projects/api", "/projects/web", "/other", "/projects")
	cfgPath := t.TempDir() + "/config.toml"
	a := New(context.Background(), cfg, cfgPath, fakeScanner{}, nil, false,
		&Launch{Source: "/projects", CoveredChildren: []string{"/projects/api", "/projects/web"}})

	a.consolidateChildren([]string{"/projects/api", "/projects/web"})

	if len(a.cfg.Paths) != 2 || a.cfg.Paths[0].Path != "/other" || a.cfg.Paths[1].Path != "/projects" {
		t.Errorf("cfg.Paths = %+v, want [/other /projects]", a.cfg.Paths)
	}
	for _, gone := range []string{"/projects/api", "/projects/web"} {
		if _, ok := a.state[gone]; ok {
			t.Errorf("state still holds removed source %s", gone)
		}
	}
	loaded, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reloading config: %v", err)
	}
	if len(loaded.Paths) != 2 {
		t.Errorf("persisted Paths = %+v, want the two survivors", loaded.Paths)
	}
	expected := selection{source: "/projects", projectIdx: -1}
	if a.sel != expected {
		t.Errorf("sel = %+v, want cursor kept on parent %+v", a.sel, expected)
	}
}

func TestMaybeOfferConsolidationOpensConfirm(t *testing.T) {
	cfg := pathsConfig("/projects/api", "/projects")
	a := newLaunchApp(t, cfg,
		&Launch{Source: "/projects", CoveredChildren: []string{"/projects/api"}})

	a.maybeOfferConsolidation()

	if !a.pages.HasPage(pageConfirm) {
		t.Errorf("confirm modal not shown for covered children")
	}
}

func TestMaybeOfferConsolidationNoChildrenNoModal(t *testing.T) {
	a := newLaunchApp(t, pathsConfig("/projects"), &Launch{Source: "/projects"})

	a.maybeOfferConsolidation()

	if a.pages.HasPage(pageConfirm) {
		t.Errorf("confirm modal shown without covered children")
	}
}
