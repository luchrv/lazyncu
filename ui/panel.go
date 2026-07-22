package ui

import (
	"fmt"
	"path/filepath"

	"github.com/rivo/tview"

	"github.com/luchrv/ncu-tui/audit"
	"github.com/luchrv/ncu-tui/orchestrator"
	"github.com/luchrv/ncu-tui/scanner"
	"github.com/luchrv/ncu-tui/semver"
)

// refreshTree rebuilds the sources panel from current state, preserving the
// selection when its node still exists.
func (a *App) refreshTree() {
	root := a.tree.GetRoot()
	root.ClearChildren()

	var selectedNode *tview.TreeNode
	for _, src := range a.order {
		st := a.state[src]
		node := tview.NewTreeNode(sourceText(src, st)).
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

func sourceText(src string, st *sourceState) string {
	name := "Global (npm -g)"
	if src != orchestrator.SourceGlobal {
		name = filepath.Base(src)
	}
	switch {
	case st.loading:
		return fmt.Sprintf("%s  [gray]scanning…[-]", name)
	case st.event.Err != nil:
		return fmt.Sprintf("%s  [red]✗ scan failed[-]", name)
	case src == orchestrator.SourceGlobal:
		return fmt.Sprintf("%s  %s [gray]│ audit n/a[-]",
			name, updateSummary(countPackages(st.event.Packages)))
	default:
		return name
	}
}

func projectText(pr orchestrator.ProjectResult) string {
	return fmt.Sprintf("%s  %s [gray]│[-] %s",
		pr.Label, updateSummary(pr.Counters), auditSummary(pr.Audit))
}

// updateSummary renders semver counters like "[red]3M[-] [yellow]5m[-] [green]2p[-]".
func updateSummary(c semver.Counters) string {
	if c.Total() == 0 {
		return "[green]up to date[-]"
	}
	out := ""
	if c.Major > 0 {
		out += fmt.Sprintf("[red]%dM[-] ", c.Major)
	}
	if c.Minor > 0 {
		out += fmt.Sprintf("[yellow]%dm[-] ", c.Minor)
	}
	if c.Patch > 0 {
		out += fmt.Sprintf("[green]%dp[-] ", c.Patch)
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
	c := res.Counters
	if c.Total() == 0 {
		return "[green]0 vulns[-]"
	}
	out := ""
	if c.Critical > 0 {
		out += fmt.Sprintf("[red::b]%dC[-:-:-] ", c.Critical)
	}
	if c.High > 0 {
		out += fmt.Sprintf("[red]%dH[-] ", c.High)
	}
	if c.Moderate > 0 {
		out += fmt.Sprintf("[yellow]%dM[-] ", c.Moderate)
	}
	if c.Low > 0 {
		out += fmt.Sprintf("[gray]%dL[-] ", c.Low)
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
