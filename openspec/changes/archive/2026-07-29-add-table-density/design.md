# Design: add-table-density

## Context

Batch 4 (UX-14/15/17/21) on v0.5.0. Decisions closed: Q1=A (sort cycles scan → severity → name; no version sort — YAGNI over the backlog wording), Q2=A (inline filter input in the message zone, lazygit-style), Q3=A (sort and filter apply to both table views), Q4=A (glyph prefixes in cells; full `NO_COLOR` deferred to UX-22). Minor calls: session-only state (persistence is UX-25), case-insensitive substring on package name, marks/commands ignore visibility.

## Goals / Non-Goals

**Goals:**
- Answer "what's worst?" and "where is X?" in a 40-row table without scrolling.
- Stop clipping the most informative field silently.
- Finish the non-color severity channel in the last place it was missing (table rows).

**Non-Goals:**
- No version-order sort, no multi-key sort, no per-column sort headers.
- No regex/fuzzy filter.
- No `NO_COLOR`/theming (UX-22), no persistence of sort/filter (UX-25).

## Decisions

### D1 — Sort and filter as pure view transforms

`sortPackages`/`sortVulns` return sorted copies (never mutate scan results — the immutability rule); `filterPackages`/`filterVulns` filter by case-insensitive substring on the name. Applied in `renderPackages`/`renderVulns` after project resolution, before row rendering, so `rowPkgs` (mark cursor mapping) always matches what is visible. Severity ranks: major>minor>patch (semver), critical>high>moderate>low (audit); ties keep scan order (stable sort).

### D2 — Sort state is one enum, advertised in the title

`sortMode` (scan/severity/name) on `App`; `s` (contexts: both table views) cycles it and refreshes. The title composer appends ` · sort: <mode>` when not scan order, and ` · /<query>` when filtering — one function builds the whole title, tested as data.

### D3 — Inline filter input in the bottom bar (Q2=A)

A pre-built `InputField` sits collapsed (width 0) in the bottom flex next to the message zone. `/` expands it, focuses it, and the existing InputField guard in `handleKey` keeps every dashboard key (including `q`) out of the way while typing. `SetChangedFunc` applies the filter live; `Enter` collapses the input keeping the filter and focuses the table; `Esc` in the input clears everything. The message zone is width-0 while filtering — a toast never fights the input.

### D4 — Two-stage Esc in the table

The `Esc` dispatch action becomes: active filter → clear it and stay; otherwise → back to the tree. Matches the "Esc peels one layer" convention the app already teaches with modals.

### D5 — Column budgets

`detailHeader`/`detailRow` gain per-column expansion: `Package` and `Via` expand (1), everything else natural width (0). `Via` additionally gets a max width with `middleEllipsis(s, max)` — head and tail preserved around `…` — because the ends of a dependency chain (direct dependent, vulnerable leaf) are the informative parts.

### D6 — Glyph prefixes in severity cells (Q4=A)

Packages: `▲ major` / `● minor` / `▪ patch` via the same glyph set as counters. Vulns: `C critical` / `H high` / `M moderate` / `L low`. No new alphabets — the cells reuse the two existing ones, and every severity signal in the app now survives without color. `NO_COLOR` env support lands with UX-22 where color emission has a single home.

## Risks / Trade-offs

- [Filter persists across project switches and could confuse] → the title always shows ` · /<query>`; clearing is one `Esc`.
- [`x` clears marks the filter is hiding] → documented behavior: marks/commands are independent of visibility; the mark counter arrives with UX-13.
- [Middle-ellipsis budget is static (not width-reactive)] → cap chosen generously (60 cols); full reactive columns would need a table rewrite — not worth it now.
- [Stable sort needed for predictable ties] → `sort.SliceStable` / `slices.SortStableFunc`.

## Migration Plan

Pure UI change on `feat/add-table-density`; single PR; revert = revert.

## Open Questions

None — Q1–Q4 closed.
