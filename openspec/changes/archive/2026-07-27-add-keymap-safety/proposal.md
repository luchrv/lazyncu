# Proposal: add-keymap-safety

## Why

The UX review (`docs/UX-REVIEW.md`, findings O-06/O-07/O-18) identified the dashboard's three worst defects: every key fires regardless of focus (`d` deletes a registered path while the Packages table — or even the About modal — is on screen, without being listed in the contextual help), destructive actions run with no confirmation (`d` removes a path instantly; `r` silently discards package marks), and severity counters use one glyph with two meanings on the same line (`3M … 3M` = major vs moderate — which also drifts from what `dashboard-ui` spec already promises: "3 major, 5 minor, 2 patch"). The help bar is a hand-written string that can lie about what keys do; today it does.

## What Changes

- **Context-scoped keybindings**: keys are dispatched against five contexts — `global`, `tree`, `table-packages`, `table-vulns`, `modal` — instead of unconditionally. `a`/`d`/`r`-confirm targets/`Enter` belong to the tree; `Space`/`x` to the packages table; `q`/`c`/`v`/`m`/`h`/`r`/`Tab` stay global. Modals are inert except `q` (About only) and their own close keys.
- **Declarative keymap as single source of truth**: one data table drives key dispatch, the focus-dependent help bar, and out-of-context behavior. Hand-written help strings are removed; the help bar can no longer drift from actual behavior.
- **Teaching hints**: pressing a key that exists in another context shows a transient status hint (e.g., "d only works in Sources — Tab to go back") that auto-expires after ~5s. Unbound keys stay silent.
- **Confirmation modals for destructive actions**: `d` (remove path) always asks; `r` (rescan) asks only when the source has marks in any of its projects, stating the aggregate loss ("discards 8 marks across 3 projects"). Modals default focus to Cancel, `Esc` cancels, `q` is inert inside confirmations.
- **Unambiguous severity counters**: semver counters switch to shapes (`▲3 ●5 ▪2`), audit counters keep letters (`C1 H2 M3 L4`) — two disjoint alphabets, no collision, same width as today. Shapes double as a non-color severity channel.
- **Legend in the About modal**: the counter legend (shapes + letters) is added to the existing About modal.
- **Width-aware help bar**: the hardcoded 106-column help zone is replaced by variants chosen by terminal width, so the generated help never clips the message zone.

## Capabilities

### New Capabilities

<!-- none — everything lands in the existing dashboard-ui capability -->

### Modified Capabilities

- `dashboard-ui`: adds two requirements (context-scoped keybindings driven by a declarative keymap; destructive actions require confirmation) and modifies six — path management (`d` confirmation), rescan (`r` mark-loss confirmation), package marks (no silent discard), severity counters (shape/letter encoding, fixing the "3 major, 5 minor, 2 patch" drift), help bar (generated from the keymap, width-aware), and About modal (hosts the legend).

## Impact

- **Code**: `ui/` only — `input.go` (rewritten around the keymap table), `layout.go` (generated help, width variants), `panel.go` (counter rendering), `about.go` (legend), new `keymap.go` + `confirm.go`. Core packages (`command`, `scanner`, `orchestrator`, `semver`, `audit`, `config`) untouched.
- **Specs**: `dashboard-ui` delta (2 added, 6 modified requirements).
- **Tests**: first tests for the `ui` package — keymap dispatch, confirm guards, and counter summaries are pure functions testable without tview.
- **Docs**: README keybindings table updated (counter legend, confirmation behavior).
- **No breaking changes**: read-only invariant preserved; no config format changes; existing keys keep their meaning, they only stop firing in contexts where they were never advertised.
