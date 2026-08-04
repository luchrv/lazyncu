package ui

import (
	"strings"
	"testing"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/scanner"
	"github.com/luchrv/lazyncu/semver"
)

func pkgsFixture() []scanner.Package {
	return []scanner.Package{
		{Name: "debug", Severity: semver.Patch},
		{Name: "axios", Severity: semver.Major},
		{Name: "chalk", Severity: semver.Minor},
		{Name: "lodash", Severity: semver.Major},
	}
}

func TestSortPackagesBySeverity(t *testing.T) {
	in := pkgsFixture()

	out := sortPackages(in, sortSeverity)

	want := []string{"axios", "lodash", "chalk", "debug"} // majors keep scan order
	for i, name := range want {
		if out[i].Name != name {
			t.Errorf("row %d = %s, want %s", i, out[i].Name, name)
		}
	}
	if in[0].Name != "debug" {
		t.Error("sortPackages must not mutate its input")
	}
}

func TestSortPackagesByName(t *testing.T) {
	out := sortPackages(pkgsFixture(), sortName)

	want := []string{"axios", "chalk", "debug", "lodash"}
	for i, name := range want {
		if out[i].Name != name {
			t.Errorf("row %d = %s, want %s", i, out[i].Name, name)
		}
	}
}

func TestSortPackagesScanPassthrough(t *testing.T) {
	in := pkgsFixture()
	out := sortPackages(in, sortScan)

	for i := range in {
		if out[i].Name != in[i].Name {
			t.Errorf("scan order must be untouched, row %d = %s", i, out[i].Name)
		}
	}
}

func TestSortVulnsBySeverity(t *testing.T) {
	in := []audit.Vulnerability{
		{Name: "a", Severity: audit.Low},
		{Name: "b", Severity: audit.Critical},
		{Name: "c", Severity: audit.Moderate},
		{Name: "d", Severity: audit.High},
	}

	out := sortVulns(in, sortSeverity)

	want := []string{"b", "d", "c", "a"}
	for i, name := range want {
		if out[i].Name != name {
			t.Errorf("row %d = %s, want %s", i, out[i].Name, name)
		}
	}
}

func TestFilterPackagesCaseInsensitive(t *testing.T) {
	out := filterPackages(pkgsFixture(), "AX")

	if len(out) != 1 || out[0].Name != "axios" {
		t.Errorf("filter = %v, want only axios", out)
	}
}

func TestFilterEmptyQueryPassthrough(t *testing.T) {
	in := pkgsFixture()
	if got := filterPackages(in, ""); len(got) != len(in) {
		t.Errorf("empty query must pass everything, got %d rows", len(got))
	}
}

func TestMiddleEllipsis(t *testing.T) {
	if got := middleEllipsis("short", 10); got != "short" {
		t.Errorf("short strings pass through, got %q", got)
	}
	got := middleEllipsis("lodash ← body-parser ← express", 20)
	if len([]rune(got)) > 20 {
		t.Errorf("result exceeds budget: %q (%d runes)", got, len([]rune(got)))
	}
	if got[:6] != "lodash" {
		t.Errorf("head must be preserved: %q", got)
	}
	if got[len(got)-len("press"):] != "press" {
		t.Errorf("tail must be preserved: %q", got)
	}
}

func TestSeverityCellLabels(t *testing.T) {
	if got := severityCell(semver.Major); got != "▲ major" {
		t.Errorf("major cell = %q", got)
	}
	if got := severityCell(semver.Patch); got != "▪ patch" {
		t.Errorf("patch cell = %q", got)
	}
	if got := vulnCell(audit.Critical); got != "C critical" {
		t.Errorf("critical cell = %q", got)
	}
	if got := vulnCell(audit.Low); got != "L low" {
		t.Errorf("low cell = %q", got)
	}
}

func TestDetailTitleComposition(t *testing.T) {
	cases := []struct {
		vulns  bool
		mode   sortMode
		filter string
		want   string
	}{
		{false, sortScan, "", " 2 Packages (v for vulnerabilities) "},
		{false, sortSeverity, "", " 2 Packages (v for vulnerabilities) · sort: severity "},
		{true, sortName, "", " 2 Vulnerabilities (v to go back) · sort: name "},
		{false, sortScan, "axi", " 2 Packages (v for vulnerabilities) · /axi "},
		{false, sortSeverity, "axi", " 2 Packages (v for vulnerabilities) · sort: severity · /axi "},
	}
	for _, tc := range cases {
		if got := detailTitle(tc.vulns, tc.mode, tc.filter, 0, 0, ""); got != tc.want {
			t.Errorf("detailTitle(%v,%d,%q) = %q, want %q", tc.vulns, tc.mode, tc.filter, got, tc.want)
		}
	}
}

func TestDetailTitleMarkCounter(t *testing.T) {
	got := detailTitle(false, sortScan, "", 2, 6, "")
	if got != " 2 Packages (v for vulnerabilities) · 2/6 marked " {
		t.Errorf("marked title = %q", got)
	}
	// Hidden at zero and in the vulns view.
	if got := detailTitle(false, sortScan, "", 0, 6, ""); got != " 2 Packages (v for vulnerabilities) " {
		t.Errorf("zero marks must hide the counter: %q", got)
	}
	if got := detailTitle(true, sortScan, "", 2, 6, ""); got != " 2 Vulnerabilities (v to go back) " {
		t.Errorf("vulns view must not show the counter: %q", got)
	}
}

func TestDetailTitleNodeContext(t *testing.T) {
	// Appended after every other segment, in both views.
	got := detailTitle(false, sortSeverity, "axi", 0, 0, "node 18.19.0 (.nvmrc)")
	want := " 2 Packages (v for vulnerabilities) · sort: severity · /axi · node 18.19.0 (.nvmrc) "
	if got != want {
		t.Errorf("node segment title = %q, want %q", got, want)
	}
	got = detailTitle(true, sortScan, "", 0, 0, "node >=18 (engines)")
	if got != " 2 Vulnerabilities (v to go back) · node >=18 (engines) " {
		t.Errorf("vulns view node segment = %q", got)
	}
	// Empty context adds nothing.
	if got := detailTitle(false, sortScan, "", 0, 0, ""); got != " 2 Packages (v for vulnerabilities) " {
		t.Errorf("empty node context must add no segment: %q", got)
	}
}

func TestNodeContextLabel(t *testing.T) {
	cases := []struct {
		name    string
		nvmrc   string
		engines string
		want    string
	}{
		{"nvmrc wins over engines", "18.19.0", ">=18", "node 18.19.0 (.nvmrc)"},
		{"engines fallback", "", ">=18 <21", "node >=18 <21 (engines)"},
		{"lts alias shown verbatim", "lts/gallium", "", "node lts/gallium (.nvmrc)"},
		{"neither yields empty", "", "", ""},
	}
	for _, tc := range cases {
		if got := nodeContextLabel(tc.nvmrc, tc.engines); got != tc.want {
			t.Errorf("%s: nodeContextLabel(%q, %q) = %q, want %q", tc.name, tc.nvmrc, tc.engines, got, tc.want)
		}
	}
}

func TestWrapText(t *testing.T) {
	if got := wrapText("short", 20); len(got) != 1 || got[0] != "short" {
		t.Errorf("short passthrough = %v", got)
	}
	got := wrapText("one two three four", 9)
	if len(got) != 3 || got[0] != "one two" || got[1] != "three" || got[2] != "four" {
		t.Errorf("word wrap = %v", got)
	}
	got = wrapText("abcdefghij", 4)
	if len(got) != 3 || got[0] != "abcd" {
		t.Errorf("long token must hard-split: %v", got)
	}
}

func TestCommandBarContentSingleLine(t *testing.T) {
	text, lines := commandBarContent("cd /x && ncu -u", "", 60)

	if lines != 1 {
		t.Errorf("lines = %d, want 1", lines)
	}
	if !strings.Contains(text, "cd /x && ncu -u") {
		t.Errorf("text = %q", text)
	}
}

func TestCommandBarContentTwoCommands(t *testing.T) {
	_, lines := commandBarContent("cd /x && ncu -u && npm install", "cd /x && npm audit fix", 60)

	if lines != 2 {
		t.Errorf("lines = %d, want 2", lines)
	}
}

func TestCommandBarContentTruncatesLoudly(t *testing.T) {
	long := strings.Repeat("pkg-name ", 40) // ~360 chars >> 2 lines of 40
	text, lines := commandBarContent("ncu -u "+long, "", 40)

	if lines != 2 {
		t.Errorf("truncated command must cap at 2 lines, got %d", lines)
	}
	if !strings.Contains(text, "… (c copies full)") {
		t.Errorf("truncation must be explicit: %q", text)
	}
}

func TestCommandBarContentEmpty(t *testing.T) {
	text, lines := commandBarContent("", "", 60)

	if lines != 1 || !strings.Contains(text, "nothing to update here") {
		t.Errorf("empty = %q (%d lines)", text, lines)
	}
}
