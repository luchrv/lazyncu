package ui

import (
	"slices"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/config"
	"github.com/luchrv/lazyncu/orchestrator"
)

// auditKey identifies one project's audit inside a source.
type auditKey struct{ source, dir string }

// auditQueue runs the deferred vulnerability audits once every version scan
// has landed: projects are audited in panel order, at most limit at a time
// (the process bound also applies), and the selected project always jumps
// the queue. Only ever touched on the UI thread.
type auditQueue struct {
	pending []auditKey
	running int
	// gen counts audit requests per project directory, so a result that
	// lands after a rescan restarted the audit is discarded.
	gen map[string]int
}

// auditLimit is how many audits the queue keeps in flight.
func (a *App) auditLimit() int {
	if a.cfg.MaxParallel < 1 {
		return config.DefaultMaxParallel
	}
	return a.cfg.MaxParallel
}

// enqueuePendingAudits queues every auditable project whose audit is still
// pending, in panel order, then starts as many as the limit allows. Called
// when a scan stream closes and after a folder entry is rescanned.
func (a *App) enqueuePendingAudits() {
	for _, source := range a.order {
		st, ok := a.state[source]
		if !ok {
			continue
		}
		for _, pr := range st.event.Projects {
			key := auditKey{source: source, dir: pr.Dir}
			if auditable(pr) && !slices.Contains(a.audits.pending, key) {
				a.audits.pending = append(a.audits.pending, key)
			}
		}
	}
	a.pumpAudits()
}

// auditable reports whether an entry is waiting for its first audit.
func auditable(pr orchestrator.ProjectResult) bool {
	return !pr.Pending && pr.Err == nil && pr.Audit.Status == audit.StatusPending
}

// pumpAudits starts queued audits while slots remain.
func (a *App) pumpAudits() {
	for a.audits.running < a.auditLimit() && len(a.audits.pending) > 0 {
		key := a.audits.pending[0]
		a.audits.pending = a.audits.pending[1:]
		a.startAudit(key)
	}
}

// ensureAudit audits the selected project right away when its audit is
// still pending, ahead of the queue. Called on selection changes and after
// every scan snapshot.
func (a *App) ensureAudit() {
	pr, ok := a.selectedProject()
	if !ok || !auditable(pr) {
		return
	}
	key := auditKey{source: a.sel.source, dir: pr.Dir}
	a.audits.pending = slices.DeleteFunc(a.audits.pending, func(k auditKey) bool { return k == key })
	a.startAudit(key)
}

// startAudit marks the project running and launches its audit; the result
// lands through the UI choke point. A key whose project is no longer
// pending (rescanned, removed) is skipped.
func (a *App) startAudit(key auditKey) {
	st, ok := a.state[key.source]
	if !ok {
		return
	}
	idx := slices.IndexFunc(st.event.Projects, func(pr orchestrator.ProjectResult) bool {
		return pr.Dir == key.dir && auditable(pr)
	})
	if idx < 0 {
		return
	}
	pr := st.event.Projects[idx]
	a.setAudit(key.source, key.dir, audit.Result{Status: audit.StatusRunning})
	if a.audits.gen == nil {
		a.audits.gen = map[string]int{}
	}
	a.audits.gen[key.dir]++
	gen := a.audits.gen[key.dir]
	a.audits.running++
	go func() {
		res := orchestrator.AuditProject(a.ctx, a.deps, pr)
		a.tv.QueueUpdateDraw(func() { a.applyAudit(key, gen, res) })
	}()
	a.refreshAll()
}

// applyAudit records a finished audit unless a newer request for the same
// project superseded it, then lets the queue advance.
func (a *App) applyAudit(key auditKey, gen int, res audit.Result) {
	a.audits.running--
	if a.audits.gen[key.dir] == gen && a.setAudit(key.source, key.dir, res) {
		a.refreshAll()
	}
	a.pumpAudits()
}

// setAudit replaces one project's audit inside a fresh snapshot of the
// source's projects (snapshots are never mutated in place). It reports
// whether the project was found.
func (a *App) setAudit(source, dir string, res audit.Result) bool {
	st, ok := a.state[source]
	if !ok {
		return false
	}
	idx := slices.IndexFunc(st.event.Projects, func(pr orchestrator.ProjectResult) bool {
		return pr.Dir == dir && !pr.Pending
	})
	if idx < 0 {
		return false
	}
	projects := slices.Clone(st.event.Projects)
	projects[idx].Audit = res
	st.event.Projects = projects
	return true
}
