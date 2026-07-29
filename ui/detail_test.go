package ui

import (
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
		if got := detailTitle(tc.vulns, tc.mode, tc.filter); got != tc.want {
			t.Errorf("detailTitle(%v,%d,%q) = %q, want %q", tc.vulns, tc.mode, tc.filter, got, tc.want)
		}
	}
}
