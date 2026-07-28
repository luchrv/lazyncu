package ui

import (
	"strings"
	"testing"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/semver"
)

func TestUpdateSummaryShapes(t *testing.T) {
	got := updateSummary(semver.Counters{Major: 3, Minor: 5, Patch: 2})

	want := "[red]▲3[-] [yellow]●5[-] [green]▪2[-]"
	if got != want {
		t.Errorf("updateSummary = %q, want %q", got, want)
	}
}

func TestUpdateSummarySkipsZeroCounters(t *testing.T) {
	got := updateSummary(semver.Counters{Major: 1})

	if got != "[red]▲1[-]" {
		t.Errorf("updateSummary = %q, want only the major glyph", got)
	}
}

func TestUpdateSummaryUpToDate(t *testing.T) {
	got := updateSummary(semver.Counters{})

	if got != "[green]up to date[-]" {
		t.Errorf("updateSummary = %q, want up-to-date text", got)
	}
}

func TestUpdateSummaryUsesNoAuditLetters(t *testing.T) {
	got := updateSummary(semver.Counters{Major: 2, Minor: 1, Patch: 4})

	for _, letter := range []string{"M", "m", "p", "C", "H", "L"} {
		if strings.Contains(got, letter) {
			t.Errorf("semver summary must not use letter %q (audit alphabet): %q", letter, got)
		}
	}
}

func TestAuditSummaryLetterCounters(t *testing.T) {
	res := audit.Result{Status: audit.StatusOK, Counters: audit.Counters{
		Critical: 1, High: 2, Moderate: 3, Low: 4,
	}}

	got := auditSummary(res)

	want := "[red::b]C1[-:-:-] [red]H2[-] [yellow]M3[-] [gray]L4[-]"
	if got != want {
		t.Errorf("auditSummary = %q, want %q", got, want)
	}
}

func TestAuditSummaryNonOKStates(t *testing.T) {
	if got := auditSummary(audit.Result{Status: audit.StatusNotAvailable}); got != "[gray]audit n/a[-]" {
		t.Errorf("n/a summary = %q", got)
	}
	if got := auditSummary(audit.Result{Status: audit.StatusFailed}); got != "[red]audit ✗[-]" {
		t.Errorf("failed summary = %q", got)
	}
	if got := auditSummary(audit.Result{Status: audit.StatusOK}); got != "[green]0 vulns[-]" {
		t.Errorf("clean summary = %q", got)
	}
}
