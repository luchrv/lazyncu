# Design: add-flow-polish

## Context

Batch 5 (UX-13/16/18/19/20) on v0.6.0. Decisions closed: Q1=A (mark counter in the panel title with the other view indicators), Q2=A (`R` global with one aggregate confirmation when marks would be lost), Q3=A (content-sized command bar 1–4 lines + explicit truncation indicator — the backlog's fixed 3 rows would regress the update+fix case), Q4=A (full wrapped error in the detail panel + retry hint; no modal — better altitude than the backlog's suggestion).

## Goals / Non-Goals

**Goals:**
- Every stateful thing the user created (marks) or is waiting on (rescans, errors) is visible and actionable where they look.
- The command bar never hides part of a command without saying so.
- `Esc` peels one layer everywhere, including the tree.

**Non-Goals:**
- No modal for errors (Q4), no per-source rescan queue/parallelism changes.
- No theming/persistence/mouse (later batches).

## Decisions

### D1 — Mark counter in the title (Q1=A)

`detailTitle` gains marked/total: ` · N/M marked` appears only in the packages view when N > 0; M is the project's package count before filtering. The title remains the single home for view state (sort, filter, marks).

### D2 — `R` = the `r` semantics, aggregated (Q2=A)

`rescanAll` collects idle sources (loading ones are skipped — the existing overlap guard, silently), sums marks and marked projects across all sources, and asks once via the existing `confirm()` when the sum is positive. `doRescanAll` flips every idle source to loading, clears marks, launches `scanOne` per source, and reports one status line. All-scanning edge → warn message. `R` sits in the keymap as a global row (bar-visible, non-compact).

### D3 — Command bar sizes to content, truncates loudly (Q3=A)

Pure `commandBarContent(update, fix, innerW)` returns the rendered text and its content-line count: each present command gets up to 2 wrapped lines; anything longer is cut with `… (c copies full)`. `refreshCommandBar` resizes the bar (stored `right` flex) to lines+2 borders, clamped to 3–6 total rows. Inner width derives from the observed screen width (the detail column is 2/3 of the body); the resize hook re-renders on width change. Copying always uses the untruncated command.

### D4 — Error rendering with room to breathe (Q4=A)

`renderScanError` fills the detail panel: a red header row "✗ scan failed — press r to retry", a blank row, then the error text wrapped at the inner width via a pure `wrapText` helper. The tree row keeps its compact `✗ scan failed` badge.

### D5 — One `Esc` binding, context-routed (UX-20)

The two `Esc` dispatch rows collapse into one (`allPanels`) routing through `escape()`: table contexts keep the existing filter-then-tree staging; the tree folds the selected source (a selected project folds its parent — `toggleFold` already moves the selection up), and when nothing is foldable an info hint replaces the silence. Avoids making `lookupDispatch` context-aware for a single key.

## Risks / Trade-offs

- [Inner-width estimate for wrapping is approximate (flex math)] → conservative floor; worst case a line wraps one column early. Exactness would need draw-time hooks for marginal gain.
- [`R` skipping in-flight sources may surprise] → the status line reports how many sources were launched; skipped ones are already visibly scanning.
- [Title accumulating indicators (`· sort · /q · 2/6 marked`)] → each appears only when active; all three at once is rare and still short.

## Migration Plan

Pure UI change on `feat/add-flow-polish`; single PR — **stops for user review (branch protection)**; tag `v0.7.0` only after verifying the merge commit.

## Open Questions

None — Q1–Q4 closed; UX-20 pre-agreed.
