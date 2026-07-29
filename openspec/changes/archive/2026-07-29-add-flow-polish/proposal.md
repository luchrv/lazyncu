# Proposal: add-flow-polish

## Why

Fifth batch of the UX review (`docs/UX-REVIEW.md`): the remaining P1 friction points. Mark count is invisible outside the generated command (O-22), rescanning a whole workspace means visiting each source (O-11), a long update command silently wraps out of the fixed command bar (O-02), a failed scan shows a one-line error with no retry hint (O-17), and `Esc` in the tree is the one place where "peel one layer" goes silent (O-09).

## What Changes

- **Mark counter** (UX-13): the Packages title shows ` · N/M marked` while any package of the current entry is marked, alongside the existing sort/filter indicators.
- **Rescan all** (UX-16): `R` rescans every idle source; sources already scanning are skipped by the existing guard. If any source would lose marks, one aggregate confirmation ("discards N marks across M projects") covers the whole sweep; with no marks it starts immediately.
- **Content-sized command bar** (UX-18): the command bar grows 1–4 content lines to fit the visible commands; beyond that the text is truncated with an explicit `… (c copies full)` indicator. `c` always copies the complete command.
- **Legible scan errors** (UX-19): a failed source's detail panel shows the full error text wrapped over multiple lines plus a "press r to retry" hint — no modal, the panel has the space.
- **Esc folds in the tree** (UX-20): `Esc` in the sources tree collapses the selected source (or the parent of a selected project); when nothing is foldable it says so instead of staying silent.

## Capabilities

### New Capabilities

<!-- none — everything lands in the existing dashboard-ui capability -->

### Modified Capabilities

- `dashboard-ui`: adds two requirements (rescan-all; content-sized command bar) and modifies three — package marks (title counter), per-source loading/error states (full wrapped error + retry hint), and context-scoped keybindings (`R` global, `Esc` folds in the tree).

## Impact

- **Code**: `ui/` only — `detail.go` (title counter, error rendering), `input.go`/`app.go` (rescan-all, escape-in-tree), `layout.go` (dynamic command bar), `keymap.go` (`R`, unified `Esc`). Core packages untouched.
- **Specs**: `dashboard-ui` delta (2 added, 3 modified).
- **Tests**: pure — title counter, aggregate-across-sources counting, command-bar content/truncation, error wrapping, escape staging in the tree.
- **Docs**: README (`R`, `Esc` semantics); UX-REVIEW items UX-13/16/18/19/20 marked shipped.
- **No breaking changes**.
