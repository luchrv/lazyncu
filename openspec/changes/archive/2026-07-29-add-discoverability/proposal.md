# Proposal: add-discoverability

## Why

Second batch of the UX review (`docs/UX-REVIEW.md`): make the app self-explanatory. Today the full keymap only exists in the README (O-08), a first launch with no registered paths is a dead end with no call to action (O-25), registered source nodes show a bare folder name so folding destroys all information while only the global node summarizes (O-19), and nothing marks which panel has keyboard focus beyond a help-bar swap (O-03/O-05). The declarative keymap shipped in `add-keymap-safety` makes the cheat sheet and any new bindings nearly free and drift-proof.

## What Changes

- **`?` keymap cheat-sheet modal**: generated from the declarative keymap, grouped by context (Global / Sources / Packages), including help-only rows (`↵ fold`, `↑↓ move`) and the counter legend. Closes on `Esc`/`?`; `q` quits. About stays on `h` but hands the legend over to `?` (single legend helper, one home).
- **First-run empty state**: when no paths are registered, the sources tree shows a hint ("No paths registered — press a to add one") and the Packages panel explains what `a` accepts (single project, monorepo, folder of projects; detection automatic) plus a `?` pointer — without ever hiding real global-scan results.
- **Aggregate counters on every source node**: registered sources render summed semver counters across their projects plus summed audit counters (only successfully audited projects; a `✗` marker when any project's audit failed; all-n/a renders as audit n/a). Folding a source no longer destroys information.
- **Focus affordance**: the focused panel gets a yellow border and title; panel titles are numbered (`1 Sources`, `2 Packages`); new `1`/`2` keys jump focus directly (advertised in `?`, not in the bottom bar).

## Capabilities

### New Capabilities

<!-- none — everything lands in the existing dashboard-ui capability -->

### Modified Capabilities

- `dashboard-ui`: adds three requirements (keymap cheat-sheet modal; first-run empty state; focused panel visually highlighted) and modifies three — severity counters (per-source aggregates with audit sums), context-scoped keybindings (`?`, `1`, `2` join the global set; bar-hidden bindings surface in `?`), and the About modal (legend moves to `?`).

## Impact

- **Code**: `ui/` only — `keymap.go` (new bindings, bar-hidden flag, cheat-sheet text generator), new `keys.go` (the `?` modal), `panel.go` (source aggregation), `detail.go`/`layout.go` (numbered titles, focus styling, empty state), `about.go` (legend removal), `input.go`/`app.go` (modal wiring). Core packages untouched.
- **Specs**: `dashboard-ui` delta (3 added, 3 modified requirements).
- **Tests**: extend the pure `ui` tests — cheat-sheet generation, aggregation rules, empty-state predicates, focus-jump dispatch.
- **Docs**: README keybindings table gains `?`, `1`, `2`.
- **No breaking changes**: read-only invariant preserved; no config changes; existing keys keep their meaning.
