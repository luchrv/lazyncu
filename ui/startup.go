package ui

import (
	"fmt"
	"strings"

	"github.com/luchrv/lazyncu/config"
	"github.com/luchrv/lazyncu/launch"
	"github.com/luchrv/lazyncu/orchestrator"
)

// Launch is the already-classified positional CLI argument the App acts on
// at startup. Config effects (registering a new or parent path) happened
// before the App was built; only selection and consolidation remain.
type Launch struct {
	// Source is the panel source to select on startup.
	Source string
	// ProjectDir, when set, is the project directory to select once
	// Source's scan delivers its event.
	ProjectDir string
	// CoveredChildren, when non-empty, are registered paths now covered by
	// Source; the App offers to remove them.
	CoveredChildren []string
}

// applyLaunch sets the startup selection and pending project state from the
// launch intent. Called during New, before the first render.
func (a *App) applyLaunch(l *Launch) {
	if l == nil {
		return
	}
	if _, ok := a.state[l.Source]; !ok {
		return // classified against a config the App did not receive
	}
	a.sel = selection{source: l.Source, projectIdx: -1}
	a.launch = l
	if l.ProjectDir != "" {
		a.pendingProject = &pendingSelect{source: l.Source, dir: launch.Comparable(l.ProjectDir)}
	}
}

// pendingSelect is a deferred selection: once source's scan event arrives,
// the project whose directory matches dir gets selected.
type pendingSelect struct {
	source string
	dir    string // comparable (symlink-resolved) form
}

// resolvePendingSelection moves the selection to the project entry matching
// the pending target when its source's event arrives. Scan errors and
// missing matches degrade to keeping the source selected. One-shot: the
// first event for the source consumes the pending state either way.
func (a *App) resolvePendingSelection(ev orchestrator.Event) {
	p := a.pendingProject
	if p == nil || ev.Source != p.source {
		return
	}
	a.pendingProject = nil
	if ev.Err != nil {
		return
	}
	for i, pr := range ev.Projects {
		if launch.Comparable(pr.Dir) == p.dir {
			a.sel = selection{source: p.source, projectIdx: i}
			return
		}
	}
}

// maybeOfferConsolidation shows the one-time confirm that offers removing
// registered paths now covered by the launch target. Runs before the
// first-run welcome; both can never trigger on the same launch.
func (a *App) maybeOfferConsolidation() {
	if a.launch == nil || len(a.launch.CoveredChildren) == 0 {
		return
	}
	children := a.launch.CoveredChildren
	a.confirm(consolidateText(a.launch.Source, children),
		func() { a.consolidateChildren(children) })
}

// consolidateChildren unregisters the covered paths in one persisted step;
// the folders on disk are never touched.
func (a *App) consolidateChildren(children []string) {
	updated := a.cfg
	for _, child := range children {
		updated = updated.RemovePath(child)
	}
	if err := config.Save(a.cfgPath, updated); err != nil {
		a.setStatus(msgError, "could not save config: %v", err)
		return
	}
	a.cfg = updated
	for _, child := range children {
		delete(a.state, child)
		a.order = removeString(a.order, child)
	}
	a.refreshAll()
	a.setStatus(msgOK, "removed %s now covered by %s", plural(len(children), "path"), a.launch.Source)
}

// consolidateText words the consolidation offer, listing every covered path.
func consolidateText(parent string, children []string) string {
	return fmt.Sprintf("%s now covers %s already registered:\n\n%s\n\nStop tracking %s? The folders on disk are not touched.",
		parent, plural(len(children), "path"), strings.Join(children, "\n"),
		pluralPronoun(len(children)))
}

func pluralPronoun(n int) string {
	if n == 1 {
		return "it"
	}
	return "them"
}
