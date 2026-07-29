package ui

import (
	"strings"
	"testing"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/orchestrator"
	"github.com/luchrv/lazyncu/scanner"
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

func projWith(maj, min, pat int, res audit.Result) orchestrator.ProjectResult {
	return orchestrator.ProjectResult{
		Project: scanner.Project{Counters: semver.Counters{Major: maj, Minor: min, Patch: pat}},
		Audit:   res,
	}
}

func TestAggregateSourceSums(t *testing.T) {
	// Arrange
	projects := []orchestrator.ProjectResult{
		projWith(1, 2, 0, audit.Result{Status: audit.StatusOK, Counters: audit.Counters{High: 1}}),
		projWith(2, 3, 2, audit.Result{Status: audit.StatusOK, Counters: audit.Counters{High: 2, Low: 1}}),
	}

	// Act
	agg := aggregateSource(projects)

	// Assert
	if agg.updates != (semver.Counters{Major: 3, Minor: 5, Patch: 2}) {
		t.Errorf("updates = %+v, want 3/5/2", agg.updates)
	}
	if agg.vulns != (audit.Counters{High: 3, Low: 1}) {
		t.Errorf("vulns = %+v, want H3 L1", agg.vulns)
	}
	if agg.audited != 2 || agg.failed != 0 {
		t.Errorf("audited/failed = %d/%d, want 2/0", agg.audited, agg.failed)
	}
}

func TestAggregateSourceSkipsNotAvailableAndCountsFailed(t *testing.T) {
	projects := []orchestrator.ProjectResult{
		projWith(1, 0, 0, audit.Result{Status: audit.StatusNotAvailable, Counters: audit.Counters{Critical: 9}}),
		projWith(0, 1, 0, audit.Result{Status: audit.StatusFailed}),
	}

	agg := aggregateSource(projects)

	if agg.vulns.Total() != 0 {
		t.Errorf("n/a counters must not be summed, got %+v", agg.vulns)
	}
	if agg.audited != 0 || agg.failed != 1 {
		t.Errorf("audited/failed = %d/%d, want 0/1", agg.audited, agg.failed)
	}
}

func TestAggregateAuditText(t *testing.T) {
	cases := []struct {
		name string
		agg  sourceAggregate
		want string
	}{
		{"all n/a", sourceAggregate{}, "[gray]audit n/a[-]"},
		{"clean", sourceAggregate{audited: 2}, "[green]0 vulns[-]"},
		{"failed only", sourceAggregate{failed: 1}, "[red]audit ✗[-]"},
		{"sums", sourceAggregate{audited: 2, vulns: audit.Counters{High: 3, Low: 1}},
			"[red]H3[-] [gray]L1[-]"},
		{"sums with failure marker", sourceAggregate{audited: 1, failed: 1, vulns: audit.Counters{High: 3}},
			"[red]H3[-] [red]✗[-]"},
	}
	for _, tc := range cases {
		if got := aggregateAuditText(tc.agg); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestSourceTextShowsAggregate(t *testing.T) {
	st := &sourceState{event: orchestrator.Event{Projects: []orchestrator.ProjectResult{
		projWith(3, 0, 0, audit.Result{Status: audit.StatusOK, Counters: audit.Counters{High: 3}}),
	}}}

	got := sourceText("/tmp/work/api", st, spinnerGlyph(0))

	printable := colorTag.ReplaceAllString(got, "")
	for _, want := range []string{"api", "▲3", "H3"} {
		if !strings.Contains(printable, want) {
			t.Errorf("source row missing %q: %q", want, printable)
		}
	}
}

func TestSpinnerGlyphCycles(t *testing.T) {
	frames := map[string]bool{}
	for i := range 20 {
		frames[spinnerGlyph(i)] = true
	}
	if len(frames) != 10 {
		t.Errorf("expected 10 distinct braille frames, got %d", len(frames))
	}
	if spinnerGlyph(0) != spinnerGlyph(10) {
		t.Error("frames must wrap around")
	}
}

func TestSourceTextLoadingShowsSpinner(t *testing.T) {
	st := &sourceState{loading: true}

	got := sourceText("/tmp/x", st, spinnerGlyph(2))

	printable := colorTag.ReplaceAllString(got, "")
	if !strings.Contains(printable, spinnerGlyph(2)+" scanning…") {
		t.Errorf("loading row missing spinner: %q", printable)
	}
}

func TestAnyLoading(t *testing.T) {
	a := newTestApp(t)
	if !a.anyLoading() { // global starts loading
		t.Error("fresh app must report loading")
	}
	a.state[orchestrator.SourceGlobal].loading = false
	if a.anyLoading() {
		t.Error("nothing loading must report false")
	}
}

func TestProgressText(t *testing.T) {
	if got := progressText(2, 5); got != "[gray]scanning 2/5[-]" {
		t.Errorf("progressText = %q", got)
	}
}

func TestScanProgressCounts(t *testing.T) {
	// Arrange — global still loading, one path loading, one path done.
	a := newTestApp(t)
	registerPath(a, "/tmp/a")
	registerPath(a, "/tmp/b")
	a.state["/tmp/a"].loading = true

	// Act
	done, total := a.scanProgress()

	// Assert
	if done != 1 || total != 3 {
		t.Errorf("scanProgress = %d/%d, want 1/3", done, total)
	}
}
