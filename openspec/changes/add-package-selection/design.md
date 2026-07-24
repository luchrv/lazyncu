# Design: add-package-selection

## Context

lazyncu is a read-only tview dashboard. Selection today is tree-centric: the Packages table is display-only (`SetSelectable(false, false)`, `ui/layout.go`), all keys are global (`ui/input.go` handleKey), the help bar is a static string (`statusHelp`, `helpWidth`), and `command.ProjectUpdate`/`GlobalUpdate` build all-packages commands. `refreshDetail` rebuilds the table on every selection change and scan event. ncu accepts positional package filters (`ncu -u lodash debug` upgrades only those).

Explore decisions (2026-07-24): read-only preserved (selection shapes the copyable command only), Tab focus toggle, space mark with `✓` + yellow, `x` clears, focus-contextual help bar, per-project ephemeral selection, vulns view out of scope, demo GIF deferred.

## Goals / Non-Goals

**Goals:**
- Select a subset of packages in a project (or the global source) and get the exact filtered update command to copy.
- Zero regression to the read-only guarantee and existing keybindings.

**Non-Goals:**
- Executing updates. Selection persistence to config. Selection in the vulnerability view. Select-all helper. New demo GIF (deferred).

## Decisions

### D1 — Focus model: explicit `focusZone` field, Tab toggles

`App` gains a `tableFocused bool` (or small enum). `Tab` flips it: sets tview focus (`a.tv.SetFocus(a.detail)` / `a.tree`), enables/disables table row selectability (`SetSelectable(true, false)` only while focused), and refreshes the help bar. `Esc` in table focus returns to tree. handleKey keeps being the single input chokepoint; table-focus keys (`space`, `x`, `↑↓` handled natively by tview Table) are gated on the focus flag.

- *Alternative — rely on `tv.GetFocus()` type checks*: implicit, breaks when the About modal or input dialog is up. Explicit flag is inspectable and testable. InputField editing guard stays first, unchanged.

### D2 — Selection state: `map[string]bool` per source/project on `sourceState`

Marks key on package name, stored alongside existing per-source state (same home as `collapsed`), scoped per project entry (`selection{source, projectIdx}`). Lifecycle: preserved across focus/project switches; cleared for a source on rescan (`scanOne`) and on path removal — same places the state is already reset. Immutable-style updates (rebuild map on toggle) per team convention.

### D3 — Mark rendering inside `refreshDetail`

Marked rows render `✓ <name>` in the package cell with yellow text (plain glyph — avoids tview dynamic-color-tag pitfalls with brackets). Since `refreshDetail` already rebuilds the table from `sourceState`, marks survive rebuilds for free. Row→package mapping kept in the same order as the rendered rows so `space` toggles the package under the cursor.

### D4 — Filtered commands: new `command` functions, existing ones untouched

`command.ProjectUpdateFiltered(dir, pm, pkgs []string)` → `cd <dir> && ncu -u pkg1 pkg2 && <install>`; `command.GlobalUpdateFiltered(pkgs []scanner.Package)` → `npm install -g pkg@ver …` subset (same shape as `GlobalUpdate`, filtered input). `refreshCommandBar` picks filtered variants when the current entry has ≥1 mark. Existing functions and their specs stay intact — no marks means identical behavior to today. Package names quoted as-is (npm names never contain shell metacharacters beyond `@/-.`, scoped names `@scope/pkg` are safe unquoted).

### D5 — Contextual help bar: two constants + refresh hook

`statusHelpTree` (current string + `Tab pkgs`) and `statusHelpTable` (`q quit · ↑↓ move · ␣ mark · x clear · c copy cmd · Tab/Esc back`). `helpWidth` sized to the longer variant. Focus flips call a `refreshHelp()` that swaps the text. About modal/input dialogs don't alter the help (modal has its own close hint).

### D6 — `c` copy works in both foci

Copy already reads the current command bar content path (`currentCommands`); with marks the filtered command is what's visible, so `c` copies it with no extra logic beyond D4's command-bar wiring.

## Risks / Trade-offs

- [Tab conflicts with tview's default focus handling inside widgets] → handleKey intercepts Tab before widgets see it (global SetInputCapture already runs first); About modal open → Tab ignored.
- [Marks reference packages that disappear after rescan] → selection cleared on rescan by design (D2); stale-name risk gone.
- [Table cursor position lost on refreshTree-triggered detail rebuilds during scans] → acceptable; scans finish quickly and cursor resets to top — same behavior class as today's full rebuilds.
- [`ncu -u <pkg>` on a monorepo child project (deep scan) — filter applies per project dir] → command already targets the child project's dir (`cd <dir>`); filter semantics unchanged.
- [Global subset command with 0 upgradable marks (user marks nothing upgradable)] → marks only exist on rendered upgradable rows; empty selection falls back to full command.

## Open Questions

None — settled in explore (2026-07-24).
