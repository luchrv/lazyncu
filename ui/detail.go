package ui

import (
	"fmt"
	"slices"
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

// sortMode orders detail-table rows; session-only, cycled with s.
type sortMode int

const (
	sortScan sortMode = iota
	sortSeverity
	sortName
)

func (m sortMode) String() string {
	switch m {
	case sortSeverity:
		return "severity"
	case sortName:
		return "name"
	default:
		return "scan"
	}
}

// severityRank orders semver severities most-severe-first.
func severityRank(s semver.Severity) int {
	switch s {
	case semver.Major:
		return 0
	case semver.Minor:
		return 1
	case semver.Patch:
		return 2
	default:
		return 3
	}
}

// vulnRank orders audit severities most-severe-first.
func vulnRank(s audit.Severity) int {
	switch s {
	case audit.Critical:
		return 0
	case audit.High:
		return 1
	case audit.Moderate:
		return 2
	case audit.Low:
		return 3
	default:
		return 4
	}
}

// sortPackages returns a sorted copy; scan results are never mutated and
// ties keep scan order.
func sortPackages(pkgs []scanner.Package, mode sortMode) []scanner.Package {
	out := slices.Clone(pkgs)
	switch mode {
	case sortSeverity:
		slices.SortStableFunc(out, func(a, b scanner.Package) int {
			return severityRank(a.Severity) - severityRank(b.Severity)
		})
	case sortName:
		slices.SortStableFunc(out, func(a, b scanner.Package) int {
			return strings.Compare(a.Name, b.Name)
		})
	}
	return out
}

// sortVulns returns a sorted copy, mirroring sortPackages.
func sortVulns(vulns []audit.Vulnerability, mode sortMode) []audit.Vulnerability {
	out := slices.Clone(vulns)
	switch mode {
	case sortSeverity:
		slices.SortStableFunc(out, func(a, b audit.Vulnerability) int {
			return vulnRank(a.Severity) - vulnRank(b.Severity)
		})
	case sortName:
		slices.SortStableFunc(out, func(a, b audit.Vulnerability) int {
			return strings.Compare(a.Name, b.Name)
		})
	}
	return out
}

// filterPackages keeps rows whose name contains the query (case-insensitive).
func filterPackages(pkgs []scanner.Package, query string) []scanner.Package {
	if query == "" {
		return pkgs
	}
	q := strings.ToLower(query)
	out := make([]scanner.Package, 0, len(pkgs))
	for _, p := range pkgs {
		if strings.Contains(strings.ToLower(p.Name), q) {
			out = append(out, p)
		}
	}
	return out
}

// filterVulns mirrors filterPackages for the vulnerability view.
func filterVulns(vulns []audit.Vulnerability, query string) []audit.Vulnerability {
	if query == "" {
		return vulns
	}
	q := strings.ToLower(query)
	out := make([]audit.Vulnerability, 0, len(vulns))
	for _, v := range vulns {
		if strings.Contains(strings.ToLower(v.Name), q) {
			out = append(out, v)
		}
	}
	return out
}

// middleEllipsis shortens s to max runes keeping both ends — for dependency
// chains, the direct dependent and the vulnerable leaf are the informative
// parts.
func middleEllipsis(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	keep := max - 1
	head := (keep + 1) / 2
	return string(runes[:head]) + "…" + string(runes[len(runes)-keep/2:])
}

// severityCell renders a package severity with its shape glyph, so the
// cell is identifiable without color.
func severityCell(s semver.Severity) string {
	switch s {
	case semver.Major:
		return "▲ major"
	case semver.Minor:
		return "● minor"
	case semver.Patch:
		return "▪ patch"
	default:
		return string(s)
	}
}

// vulnCell renders an audit severity with its letter prefix.
func vulnCell(s audit.Severity) string {
	switch s {
	case audit.Critical:
		return "C critical"
	case audit.High:
		return "H high"
	case audit.Moderate:
		return "M moderate"
	case audit.Low:
		return "L low"
	default:
		return string(s)
	}
}

// detailTitle composes the detail-panel title from view, sort, filter,
// and mark state — the single home for view indicators.
func detailTitle(showVulns bool, mode sortMode, filter string, marked, total int) string {
	title := " 2 Packages (v for vulnerabilities)"
	if showVulns {
		title = " 2 Vulnerabilities (v to go back)"
	}
	if !showVulns && marked > 0 {
		title += fmt.Sprintf(" · %d/%d marked", marked, total)
	}
	if mode != sortScan {
		title += " · sort: " + mode.String()
	}
	if filter != "" {
		title += " · /" + filter
	}
	return title + " "
}

// refreshDetail renders the right panel: the package table for the current
// selection, or the vulnerability detail when that view is toggled on.
func (a *App) refreshDetail() {
	a.detail.Clear()
	a.rowPkgs = nil // marking is only valid on the packages view
	marked, total := a.markStats()
	a.detail.SetTitle(detailTitle(a.showVulns, a.sortMode, a.filterText, marked, total))
	if a.showVulns {
		a.renderVulns()
		return
	}
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
		a.renderScanError(st.event.Err)
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

	pkgs = filterPackages(sortPackages(pkgs, a.sortMode), a.filterText)
	if len(pkgs) == 0 {
		a.detailMessage("no packages match /" + a.filterText)
		return
	}

	marks := a.currentMarks()
	a.detailHeader(pkgColumns, "Package", "Current", "New", "Severity")
	for row, p := range pkgs {
		a.rowPkgs = append(a.rowPkgs, p.Name)
		color := severityColor(p.Severity)
		a.detailRow(row+1, color, pkgColumns, p.Name, p.Current, p.New, severityCell(p.Severity))
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

	vulns := filterVulns(sortVulns(res.Vulns, a.sortMode), a.filterText)
	if len(vulns) == 0 {
		a.detailMessage("no vulnerabilities match /" + a.filterText)
		return
	}

	a.detailHeader(vulnColumns, "Package", "Severity", "Range", "Fix", "Via")
	for row, v := range vulns {
		fix := "no"
		if v.FixAvailable {
			fix = "yes"
		}
		a.detailRow(row+1, vulnColor(v.Severity), vulnColumns,
			v.Name, vulnCell(v.Severity), v.Range, fix,
			middleEllipsis(chainText(v), viaBudget))
	}
}

// refreshCommandBar shows the copyable commands for the current selection,
// sizing the bar to its content; overflow is truncated with an explicit
// indicator while c always copies the full command.
func (a *App) refreshCommandBar() {
	update, fix := a.currentCommands()
	text, lines := commandBarContent(update, fix, a.cmdBarInnerWidth())
	a.cmdBar.SetText(text)
	a.right.ResizeItem(a.cmdBar, lines+2, 0) // +2 borders
}

// cmdBarInnerWidth estimates the bar's usable columns from the observed
// screen width (the detail column takes 2/3 of the body).
func (a *App) cmdBarInnerWidth() int {
	return max(20, a.screenW*2/3-4)
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

// Column expansion budgets: only identifying fields (Package, Via) grow
// with available width; versions/severity/range/fix keep natural width.
var (
	pkgColumns  = []int{1, 0, 0, 0}
	vulnColumns = []int{1, 0, 0, 0, 1}
)

// viaBudget caps the Via dependency chain before middle-ellipsis kicks in.
const viaBudget = 60

func (a *App) detailHeader(expand []int, titles ...string) {
	for col, title := range titles {
		a.detail.SetCell(0, col, tview.NewTableCell(title).
			SetTextColor(tcell.ColorWhite).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false).
			SetExpansion(expand[col]))
	}
}

func (a *App) detailRow(row int, color tcell.Color, expand []int, cells ...string) {
	for col, text := range cells {
		a.detail.SetCell(row, col, tview.NewTableCell(text).
			SetTextColor(color).
			SetExpansion(expand[col]))
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

// wrapText word-wraps s at width w; tokens longer than w are hard-split.
func wrapText(s string, w int) []string {
	if w < 1 {
		return []string{s}
	}
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		for len([]rune(word)) > w {
			if line != "" {
				lines = append(lines, line)
				line = ""
			}
			runes := []rune(word)
			lines = append(lines, string(runes[:w]))
			word = string(runes[w:])
		}
		switch {
		case line == "":
			line = word
		case len([]rune(line))+1+len([]rune(word)) <= w:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// cmdTruncationHint marks a command cut to fit the bar; c still copies it whole.
const cmdTruncationHint = " … (c copies full)"

// cmdLineBudget is the max content lines one command may occupy in the bar.
const cmdLineBudget = 2

// commandBarContent renders the command bar body for the visible commands
// and reports how many content lines it needs. A command longer than its
// budget is truncated with an explicit indicator — the bar never hides
// part of a command silently.
func commandBarContent(update, fix string, innerW int) (string, int) {
	type entry struct{ prefix, cmd string }
	entries := make([]entry, 0, 2)
	if update != "" {
		entries = append(entries, entry{"[yellow]update:[-] ", update})
	}
	if fix != "" {
		entries = append(entries, entry{"[red]fix:[-]    ", fix})
	}
	if len(entries) == 0 {
		return "[gray]nothing to update here[-]", 1
	}

	const prefixW = 8 // printable width of both prefixes
	lines := 0
	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		capacity := cmdLineBudget*innerW - prefixW
		cmd := e.cmd
		if len([]rune(cmd)) > capacity {
			keep := capacity - len([]rune(cmdTruncationHint))
			if keep < 1 {
				keep = 1
			}
			cmd = string([]rune(cmd)[:keep]) + cmdTruncationHint
		}
		used := (prefixW + len([]rune(cmd)) + innerW - 1) / innerW
		lines += min(max(used, 1), cmdLineBudget)
		parts = append(parts, e.prefix+tview.Escape(cmd))
	}
	return strings.Join(parts, "\n"), lines
}

// allSourcesMarks totals marks and marked projects across every source —
// the blast radius of a rescan-all sweep.
func allSourcesMarks(states map[string]*sourceState) (marks, projects int) {
	for _, st := range states {
		m, p := markCount(st)
		marks += m
		projects += p
	}
	return marks, projects
}

// renderScanError fills the detail panel with the full failure reason —
// wrapped so nothing is cut off — and how to retry.
func (a *App) renderScanError(err error) {
	a.detail.SetCell(0, 0, tview.NewTableCell("✗ scan failed — press r to retry").
		SetTextColor(tcell.ColorRed).
		SetSelectable(false))
	for i, line := range wrapText(err.Error(), a.cmdBarInnerWidth()) {
		a.detail.SetCell(i+2, 0, tview.NewTableCell(line).
			SetTextColor(tcell.ColorGray).
			SetSelectable(false))
	}
}
