package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/command"
	"github.com/luchrv/lazyncu/orchestrator"
	"github.com/luchrv/lazyncu/semver"
)

const chainSeparator = " ← "

// refreshDetail renders the right panel: the package table for the current
// selection, or the vulnerability detail when that view is toggled on.
func (a *App) refreshDetail() {
	a.detail.Clear()
	if a.showVulns {
		a.detail.SetTitle(" Vulnerabilities (v to go back) ")
		a.renderVulns()
		return
	}
	a.detail.SetTitle(" Packages (v for vulnerabilities) ")
	a.renderPackages()
}

func (a *App) renderPackages() {
	st, ok := a.state[a.sel.source]
	if !ok {
		return
	}
	switch {
	case st.loading:
		a.detailMessage("scanning…")
		return
	case st.event.Err != nil:
		a.detailMessage("scan failed: " + st.event.Err.Error())
		return
	}

	pkgs := st.event.Packages
	if pr, ok := a.selectedProject(); ok {
		pkgs = pr.Packages
	}
	if len(pkgs) == 0 {
		a.detailMessage("everything up to date ✓")
		return
	}

	a.detailHeader("Package", "Current", "New", "Severity")
	for row, p := range pkgs {
		color := severityColor(p.Severity)
		a.detailRow(row+1, color, p.Name, p.Current, p.New, string(p.Severity))
	}
}

func (a *App) renderVulns() {
	res, ok := a.selectedAudit()
	if !ok {
		return
	}
	switch res.Status {
	case audit.StatusNotAvailable:
		a.detailMessage("audit not available for this source (yarn projects and global packages are not audited)")
		return
	case audit.StatusFailed:
		a.detailMessage("audit failed: " + res.Err)
		return
	}
	if len(res.Vulns) == 0 {
		a.detailMessage("0 vulnerabilities ✓")
		return
	}

	a.detailHeader("Package", "Severity", "Range", "Fix", "Via")
	for row, v := range res.Vulns {
		fix := "no"
		if v.FixAvailable {
			fix = "yes"
		}
		a.detailRow(row+1, vulnColor(v.Severity),
			v.Name, string(v.Severity), v.Range, fix, chainText(v))
	}
}

// refreshCommandBar shows the copyable commands for the current selection.
func (a *App) refreshCommandBar() {
	update, fix := a.currentCommands()
	lines := make([]string, 0, 2)
	if update != "" {
		lines = append(lines, "[yellow]update:[-] "+tview.Escape(update))
	}
	if fix != "" {
		lines = append(lines, "[red]fix:[-]    "+tview.Escape(fix))
	}
	if len(lines) == 0 {
		lines = append(lines, "[gray]nothing to update here[-]")
	}
	a.cmdBar.SetText(strings.Join(lines, "\n"))
}

// currentCommands resolves the update and fix commands for the selection.
func (a *App) currentCommands() (update, fix string) {
	st, ok := a.state[a.sel.source]
	if !ok || st.loading || st.event.Err != nil {
		return "", ""
	}
	if a.sel.source == orchestrator.SourceGlobal {
		return command.GlobalUpdate(st.event.Packages), ""
	}
	pr, ok := a.selectedProject()
	if !ok {
		return "", ""
	}
	if len(pr.Packages) > 0 {
		update = command.ProjectUpdate(pr.Dir, pr.PM)
	}
	return update, audit.FixCommand(pr.Audit, pr.Dir, pr.PM)
}

// selectedProject resolves the selection to a project, falling back to the
// first project of the source when the source node itself is selected.
func (a *App) selectedProject() (orchestrator.ProjectResult, bool) {
	st, ok := a.state[a.sel.source]
	if !ok || a.sel.source == orchestrator.SourceGlobal {
		return orchestrator.ProjectResult{}, false
	}
	projects := st.event.Projects
	if a.sel.projectIdx >= 0 && a.sel.projectIdx < len(projects) {
		return projects[a.sel.projectIdx], true
	}
	if len(projects) == 1 {
		return projects[0], true
	}
	return orchestrator.ProjectResult{}, false
}

func (a *App) selectedAudit() (audit.Result, bool) {
	if a.sel.source == orchestrator.SourceGlobal {
		return audit.GlobalResult(), true
	}
	if pr, ok := a.selectedProject(); ok {
		return pr.Audit, true
	}
	return audit.Result{}, false
}

func chainText(v audit.Vulnerability) string {
	if v.Direct {
		return "direct"
	}
	return strings.Join(v.Chain, chainSeparator)
}

func (a *App) detailMessage(msg string) {
	a.detail.SetCell(0, 0, tview.NewTableCell(msg).SetTextColor(tcell.ColorGray))
}

func (a *App) detailHeader(titles ...string) {
	for col, title := range titles {
		a.detail.SetCell(0, col, tview.NewTableCell(title).
			SetTextColor(tcell.ColorWhite).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false).
			SetExpansion(1))
	}
}

func (a *App) detailRow(row int, color tcell.Color, cells ...string) {
	for col, text := range cells {
		a.detail.SetCell(row, col, tview.NewTableCell(text).
			SetTextColor(color).
			SetExpansion(1))
	}
}

func severityColor(s semver.Severity) tcell.Color {
	switch s {
	case semver.Major:
		return tcell.ColorRed
	case semver.Minor:
		return tcell.ColorYellow
	case semver.Patch:
		return tcell.ColorGreen
	default:
		return tcell.ColorGray
	}
}

func vulnColor(s audit.Severity) tcell.Color {
	switch s {
	case audit.Critical, audit.High:
		return tcell.ColorRed
	case audit.Moderate:
		return tcell.ColorYellow
	default:
		return tcell.ColorGray
	}
}
