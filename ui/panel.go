package ui

import (
	"fmt"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/orchestrator"
	"github.com/luchrv/lazyncu/scanner"
	"github.com/luchrv/lazyncu/semver"
)

// refreshTree rebuilds the sources panel from current state, preserving the
// selection when its node still exists.
func (a *App) refreshTree() {
	root := a.tree.GetRoot()
	root.ClearChildren()

	var selectedNode *tview.TreeNode
	spinner := spinnerGlyph(a.spinFrame)
	for _, src := range a.order {
		st := a.state[src]
		node := tview.NewTreeNode(sourceText(src, st, spinner)).
			SetReference(selection{source: src, projectIdx: -1})
		if node.GetReference() == a.sel {
			selectedNode = node
		}
		for i, pr := range st.event.Projects {
			child := tview.NewTreeNode(projectText(pr)).
				SetReference(selection{source: src, projectIdx: i})
			if child.GetReference() == a.sel {
				selectedNode = child
			}
			node.AddChild(child)
		}
		if len(st.event.Projects) > 0 {
			node.SetText(foldIndicator(st.collapsed) + node.GetText())
			node.SetExpanded(!st.collapsed)
		}
		root.AddChild(node)
	}

	if len(a.cfg.Paths) == 0 {
		for _, hint := range []string{"", "No paths registered.", "Press a to add one."} {
			root.AddChild(tview.NewTreeNode(hint).
				SetSelectable(false).
				SetColor(tcell.ColorGray))
		}
	}

	if selectedNode == nil && len(root.GetChildren()) > 0 {
		selectedNode = root.GetChildren()[0]
		a.sel = selectedNode.GetReference().(selection)
	}
	a.tree.SetCurrentNode(selectedNode)
}

func foldIndicator(collapsed bool) string {
	if collapsed {
		return "▸ "
	}
	return "▾ "
}

// spinnerFrames are the braille frames cycled while a source scans.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// spinnerGlyph returns the frame for an ever-increasing tick counter.
func spinnerGlyph(frame int) string {
	return spinnerFrames[frame%len(spinnerFrames)]
}

func sourceText(src string, st *sourceState, spinner string) string {
	name := "Global (npm -g)"
	if src != orchestrator.SourceGlobal {
		name = filepath.Base(src)
	}
	switch {
	case st.loading:
		return fmt.Sprintf("%s  [gray]%s scanning…[-]", name, spinner)
	case st.event.Err != nil:
		return fmt.Sprintf("%s  [red]✗ scan failed[-]", name)
	case src == orchestrator.SourceGlobal:
		return fmt.Sprintf("%s  %s [gray]│ audit n/a[-]",
			name, updateSummary(countPackages(st.event.Packages)))
	default:
		agg := aggregateSource(st.event.Projects)
		return fmt.Sprintf("%s  %s [gray]│[-] %s",
			name, updateSummary(agg.updates), aggregateAuditText(agg))
	}
}

// sourceAggregate sums a source's projects so the source row (and its
// folded form) carries the same signal as the expanded list.
type sourceAggregate struct {
	updates semver.Counters
	vulns   audit.Counters // summed over successfully audited projects only
	audited int            // projects with a usable audit
	failed  int            // projects whose audit failed
}

func aggregateSource(projects []orchestrator.ProjectResult) sourceAggregate {
	var agg sourceAggregate
	for _, pr := range projects {
		agg.updates.Major += pr.Counters.Major
		agg.updates.Minor += pr.Counters.Minor
		agg.updates.Patch += pr.Counters.Patch
		switch pr.Audit.Status {
		case audit.StatusOK:
			agg.audited++
			agg.vulns.Critical += pr.Audit.Counters.Critical
			agg.vulns.High += pr.Audit.Counters.High
			agg.vulns.Moderate += pr.Audit.Counters.Moderate
			agg.vulns.Low += pr.Audit.Counters.Low
		case audit.StatusFailed:
			agg.failed++
		}
	}
	return agg
}

// aggregateAuditText renders the audit side of a source row: letter sums
// over audited projects, a ✗ marker when any project's audit failed, and
// the n/a state when nothing produced a usable audit.
func aggregateAuditText(agg sourceAggregate) string {
	switch {
	case agg.audited == 0 && agg.failed == 0:
		return "[gray]audit n/a[-]"
	case agg.vulns.Total() == 0 && agg.failed > 0:
		return "[red]audit ✗[-]"
	case agg.vulns.Total() == 0:
		return "[green]0 vulns[-]"
	}
	out := vulnCounterText(agg.vulns)
	if agg.failed > 0 {
		out += " [red]✗[-]"
	}
	return out
}

func projectText(pr orchestrator.ProjectResult) string {
	return fmt.Sprintf("%s  %s [gray]│[-] %s",
		pr.Label, updateSummary(pr.Counters), auditSummary(pr.Audit))
}

// updateSummary renders semver counters with shape glyphs, like
// "[red]▲3[-] [yellow]●5[-] [green]▪2[-]". Shapes keep the semver side on a
// different alphabet than the audit letters (C/H/M/L), so `M` can never
// mean two things on one line, and they survive without color.
func updateSummary(c semver.Counters) string {
	if c.Total() == 0 {
		return "[green]up to date[-]"
	}
	out := ""
	if c.Major > 0 {
		out += fmt.Sprintf("[red]▲%d[-] ", c.Major)
	}
	if c.Minor > 0 {
		out += fmt.Sprintf("[yellow]●%d[-] ", c.Minor)
	}
	if c.Patch > 0 {
		out += fmt.Sprintf("[green]▪%d[-] ", c.Patch)
	}
	return out[:len(out)-1]
}

// auditSummary renders vulnerability counters, keeping the three non-OK
// states visually distinct as the spec requires.
func auditSummary(res audit.Result) string {
	switch res.Status {
	case audit.StatusNotAvailable:
		return "[gray]audit n/a[-]"
	case audit.StatusFailed:
		return "[red]audit ✗[-]"
	}
	if res.Counters.Total() == 0 {
		return "[green]0 vulns[-]"
	}
	return vulnCounterText(res.Counters)
}

// vulnCounterText renders non-zero audit letter counters like
// "[red::b]C1[-:-:-] [red]H2[-]". Callers guarantee Total() > 0.
func vulnCounterText(c audit.Counters) string {
	out := ""
	if c.Critical > 0 {
		out += fmt.Sprintf("[red::b]C%d[-:-:-] ", c.Critical)
	}
	if c.High > 0 {
		out += fmt.Sprintf("[red]H%d[-] ", c.High)
	}
	if c.Moderate > 0 {
		out += fmt.Sprintf("[yellow]M%d[-] ", c.Moderate)
	}
	if c.Low > 0 {
		out += fmt.Sprintf("[gray]L%d[-] ", c.Low)
	}
	return out[:len(out)-1]
}

func countPackages(pkgs []scanner.Package) semver.Counters {
	severities := make([]semver.Severity, len(pkgs))
	for i, p := range pkgs {
		severities[i] = p.Severity
	}
	return semver.Count(severities)
}
