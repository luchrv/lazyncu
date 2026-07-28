package ui

import "testing"

func TestMarkCountAggregatesAcrossProjects(t *testing.T) {
	// Arrange
	st := &sourceState{marks: map[int]map[string]bool{
		0: {"axios": true, "chalk": true},
		1: {},
		2: {"lodash": true},
	}}

	// Act
	marks, projects := markCount(st)

	// Assert
	if marks != 3 {
		t.Errorf("marks = %d, want 3", marks)
	}
	if projects != 2 {
		t.Errorf("projects = %d, want 2 (empty sets do not count)", projects)
	}
}

func TestMarkCountNoMarks(t *testing.T) {
	marks, projects := markCount(&sourceState{})

	if marks != 0 || projects != 0 {
		t.Errorf("markCount = (%d, %d), want (0, 0)", marks, projects)
	}
}

func TestConfirmRescanTextPlural(t *testing.T) {
	got := confirmRescanText("work/api", 8, 3)

	want := "Rescanning work/api discards 8 marks across 3 projects."
	if got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
}

func TestConfirmRescanTextSingular(t *testing.T) {
	got := confirmRescanText("api", 1, 1)

	want := "Rescanning api discards 1 mark across 1 project."
	if got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
}

func TestConfirmRemoveTextMentionsDiskSafety(t *testing.T) {
	got := confirmRemoveText("/Users/me/work/api")

	if got != "Stop tracking /Users/me/work/api?\n\nThe folder on disk is not touched." {
		t.Errorf("unexpected removal text: %q", got)
	}
}

func TestClearHintIfCurrentGenerationGuard(t *testing.T) {
	// Arrange
	a := newTestApp(t)
	a.setStatus("old hint")
	staleGen := a.msgGen
	a.setStatus("newer message")

	// Act — a stale hint timer fires after a newer message was set.
	a.clearHintIfCurrent(staleGen)

	// Assert
	if got := a.statusMsg.GetText(true); got != "newer message" {
		t.Errorf("stale timer cleared a newer message: %q", got)
	}

	// Act — the timer matching the visible message clears it.
	a.clearHintIfCurrent(a.msgGen)

	// Assert
	if got := a.statusMsg.GetText(true); got != "" {
		t.Errorf("current hint was not cleared: %q", got)
	}
}
