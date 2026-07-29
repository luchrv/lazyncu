# Proposal: add-table-density

## Why

Fourth batch of the UX review (`docs/UX-REVIEW.md`): density and non-color accessibility. Tables render in scan order with no way to see majors first or jump to a name (O-12); every column expands equally, so the `Via` dependency chain — the longest field — clips first and without a marker (O-23); and severity in table rows is carried by color plus a word, with no glyph channel (O-24).

## What Changes

- **Sort** (UX-14): `s` cycles the table order — scan order → severity (most severe first) → name — in both the packages and vulnerability views. The active order is shown in the panel title. Session-only state.
- **Incremental filter** (UX-15): `/` turns the bottom-left message zone into an inline filter input; rows filter by case-insensitive substring on the package name as you type. `Enter` keeps the filter and returns focus to the table; `Esc` clears and exits. With a filter active, `Esc` in the table clears the filter first, then a second `Esc` returns to the tree. The active filter shows in the panel title. Marks and generated commands are unaffected by visibility.
- **Column sizing** (UX-17): only identifying fields expand (`Package`, `Via`); versions, severity, range, and fix stay at their natural width. The `Via` chain gets a middle-ellipsis (`lodash ← … ← express`) when it exceeds its budget instead of clipping silently.
- **Non-color severity in rows** (UX-21, partial): the Severity cell carries the shape glyph (`▲ major`, `● minor`, `▪ patch`) and vulnerability rows carry the letter (`C critical`, …), completing the non-color channel across the UI (counters and message icons already have one). Full `NO_COLOR` handling is deferred to UX-22 (theming), which gives it a single home.

## Capabilities

### New Capabilities

<!-- none — everything lands in the existing dashboard-ui capability -->

### Modified Capabilities

- `dashboard-ui`: adds two requirements (table sorting; incremental filtering) and modifies three — dashboard layout (column expansion and Via middle-ellipsis), severity encoding (glyph prefix in table cells), and context-scoped keybindings (`s` and `/` join the table contexts; `Esc` clears an active filter before leaving the table).

## Impact

- **Code**: `ui/` only — `detail.go` (sort/filter application, column sizing, glyph cells, title composition), `keymap.go` (`s`, `/` bindings), `layout.go`/`app.go` (inline filter input, state), `input.go` (Esc two-stage). Core packages untouched.
- **Specs**: `dashboard-ui` delta (2 added, 3 modified).
- **Tests**: pure — sort comparators, filter predicate, middle-ellipsis, glyph labels, title composition, Esc staging.
- **Docs**: README keybindings (`s`, `/`); UX-REVIEW items UX-14/15/17/21 marked shipped (UX-21 partial, noted).
- **No breaking changes**.
