# Design: add-discoverability

## Context

`add-keymap-safety` (v0.3.0) left a declarative keymap (`ui/keymap.go`) driving dispatch, the help bar, and hints. This batch builds on it: UX-03 (`?` cheat sheet), UX-06 (first-run empty state), UX-05 (per-source aggregate counters), UX-09 (focus affordance). Decisions closed with the user: Q1=B (About stays on `h`, legend moves to `?` exclusively), Q2=A (cheat sheet generated from the keymap), Q3=A (semver+audit aggregates, `✗` on any failed audit, n/a ignored), Q4=A (yellow focus styling + numbered titles + `1`/`2` jump keys), Q5=A (empty state in both panels).

## Goals / Non-Goals

**Goals:**
- In-app, drift-proof keymap reference under the universal `?` key, hosting the counter legend.
- A first launch that teaches the one action that matters (`a`), without hiding real data.
- Source rows that summarize their projects so folding aggregates instead of destroying.
- Unmistakable keyboard-focus signal plus direct panel addressing.

**Non-Goals:**
- No spinner/progress/message-expiry (batch 3: UX-10/11/12).
- No sort/filter, no mark-column rework, no theming (later batches).
- No change to scanning, config, or the read-only invariant.

## Decisions

### D1 — Cheat sheet generated from the keymap, new `bar` visibility flag

`binding` gains `barHidden bool`: shown in the `?` modal but omitted from the bottom bar (used by `1`/`2`, which would clutter the bar). `keysText()` walks the keymap once, grouping rows by their context set — `allPanels` → Global, `treeOnly` → Sources, `packagesOnly`/`tableViews` → Packages — and appends the legend (shared helper also used nowhere else after About cedes it; single home in `?`). Help-only rows (`↵ fold`, `↑↓ move`, Tab wordings) appear naturally because grouping keys off `desc != ""` rows.
*Alternative rejected:* hand-written modal text — reintroduces the drift the keymap was built to kill.

### D2 — `?` modal is a third modal page, same inertia contract

New `pageKeys` page; `currentContext()` returns `ctxModal` when it is up. `handleModalKey` gains a branch: `Esc`/`?` close, `q` quits (same as About), everything else inert. Opening `?` while About or a confirmation is up is swallowed by the modal context — no stacking.

### D3 — About stays on `h`, legend moves to `?` (Q1=B)

About reverts to version/repo/license content (pre-v0.3.0 height). The legend's single home is the `?` modal, next to the keys it explains. README's legend table stays as the out-of-app reference.

### D4 — Source aggregation rules (Q3=A)

`aggregateSource(projects)` (pure): sums `pr.Counters` (semver) across all projects; sums audit counters over projects with `StatusOK`; counts `StatusFailed` projects. Rendering for a non-loading, non-failed registered source: `name  <semver-sums> │ <audit-part>` where audit-part is the letter sums, prefixed `✗` when failedCount > 0, `0 vulns` when audited-and-clean, and `audit n/a` when no project produced a usable audit (all n/a). Single-project sources show the same aggregate as their only child — consistency over dedup. Loading and scan-failed states render as today.

### D5 — Focus affordance via border+title color, numbered titles, jump keys (Q4=A)

`applyFocusStyle()` sets `SetBorderColor`/`SetTitleColor` to yellow on the focused panel (tree or detail) and default on the other; called from `setTableFocus` and initial build. Titles become ` 1 Sources ` and ` 2 Packages … ` (the detail title keeps its packages/vulns variants). New bindings: `1` → `setTableFocus(false)`, `2` → `setTableFocus(true)`, contexts `allPanels`, `barHidden` — advertised only in `?`. The command bar is never focusable and keeps default styling.

### D6 — Empty state that never hides data (Q5=A, amended)

Predicate: `len(cfg.Paths) == 0`. Tree: after the Global node, two gray, non-selectable hint nodes ("No paths registered." / "Press a to add one."). Packages panel: the onboarding text (what `a` accepts + `?` pointer) replaces only the *empty* states — while Global is selected it renders instead of "everything up to date ✓", never instead of actual package rows; scan errors keep priority. The hints disappear on the first successful `a` (tree rebuilds from config-backed state).
*Deviation from the wireframe (§3.4):* the wireframe replaced the whole right panel; that would hide live global results. Data wins.

## Risks / Trade-offs

- [`?` modal content wider/taller than About] → fixed-size centered frame sized from the generated line count; content is bounded (keymap is small and static).
- [Aggregation double-renders info on single-project sources] → accepted (D4): consistency and fold-safety beat dedup.
- [Yellow border may collide with user terminal themes] → same yellow already used for keys/marks; one constant, trivially changeable later by UX-22 theming.
- [Hint nodes in the tree could catch selection] → non-selectable nodes with nil reference; the existing `GetReference()` type-assert guard already skips them.
- [`1`/`2` as global runes could conflict with future numeric input] → no numeric input exists in panels; add-path modal already bypasses the keymap (InputField focus check).

## Migration Plan

Pure UI change, single PR on `feat/add-discoverability`; revert = revert. README updated in the same change.

## Open Questions

None — Q1–Q5 closed; wireframe deviation for the empty state documented in D6.
