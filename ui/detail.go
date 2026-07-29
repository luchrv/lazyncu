package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/command"
	"github.com/luchrv/lazyncu/orchestrator"
	"github.com/luchrv/lazyncu/scanner"
	"github.com/luchrv/lazyncu/semver"
)

const chainSeparator = " ← "

// refreshDetail renders the right panel: the package table for the current
// selection, or the vulnerability detail when that view is toggled on.
func (a *App) refreshDetail() {
	a.detail.Clear()
	a.rowPkgs = nil // marking is only valid on the packages view
	if a.showVulns {
		a.detail.SetTitle(" 2 Vulnerabilities (v to go back) ")
		a.renderVulns()
		return
	}
	a.detail.SetTitle(" 2 Packages (v for vulnerabilities) ")
	a.renderPackages()
}

func (a *App) renderPackages() {
	st, ok := a.state[a.sel.source]
	if !ok {
		return
	}
	switch {
	case st.loading:
		a.detailMessage(spinnerGlyph(a.spinFrame) + " scanning…")
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
		// Onboarding replaces only the empty view: real rows and scan
		// errors always win over it.
		if len(a.cfg.Paths) == 0 {
			a.renderOnboarding()
			return
		}
		a.detailMessage("everything up to date ✓")
		return
	}

	marks := a.currentMarks()
	a.detailHeader("Package", "Current", "New", "Severity")
	for row, p := range pkgs {
		a.rowPkgs = append(a.rowPkgs, p.Name)
		color := severityColor(p.Severity)
		a.detailRow(row+1, color, p.Name, p.Current, p.New, string(p.Severity))
		if marks[p.Name] {
			a.detail.SetCell(row+1, 0, tview.NewTableCell("✓ "+p.Name).
				SetTextColor(tcell.ColorYellow).
				SetExpansion(1))
		}
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
// A non-empty mark set narrows the update command to the marked packages;
// the fix command is never affected by marks.
func (a *App) currentCommands() (update, fix string) {
	st, ok := a.state[a.sel.source]
	if !ok || st.loading || st.event.Err != nil {
		return "", ""
	}
	marks := a.currentMarks()
	if a.sel.source == orchestrator.SourceGlobal {
		if len(marks) > 0 {
			if filtered := command.GlobalUpdateFiltered(st.event.Packages, marks); filtered != "" {
				return filtered, ""
			}
		}
		return command.GlobalUpdate(st.event.Packages), ""
	}
	pr, ok := a.selectedProject()
	if !ok {
		return "", ""
	}
	if len(pr.Packages) > 0 {
		update = command.ProjectUpdate(pr.Dir, pr.PM)
		if names := markedNames(pr.Packages, marks); len(names) > 0 {
			update = command.ProjectUpdateFiltered(pr.Dir, pr.PM, names)
		}
	}
	return update, audit.FixCommand(pr.Audit, pr.Dir, pr.PM)
}

// markedNames filters the scan's packages down to the marked ones,
// preserving scan order and dropping marks that no longer exist.
func markedNames(pkgs []scanner.Package, marks map[string]bool) []string {
	if len(marks) == 0 {
		return nil
	}
	names := make([]string, 0, len(marks))
	for _, p := range pkgs {
		if marks[p.Name] {
			names = append(names, p.Name)
		}
	}
	return names
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

// renderOnboarding fills the empty first-run Packages panel with the one
// action that matters and where to learn the rest.
func (a *App) renderOnboarding() {
	lines := []string{
		"No project paths registered yet.",
		"",
		"Press a and enter a folder — a single project, a monorepo,",
		"or a folder of projects. Detection is automatic.",
		"",
		"Press ? for all keybindings.",
	}
	for row, line := range lines {
		a.detail.SetCell(row, 0, tview.NewTableCell(line).
			SetTextColor(tcell.ColorGray).
			SetSelectable(false))
	}
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
