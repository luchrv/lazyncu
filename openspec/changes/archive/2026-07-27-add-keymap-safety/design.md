# Design: add-keymap-safety

## Context

`ui/input.go`'s `handleKey` is a flat switch installed as the tview global input capture. It dispatches every rune unconditionally: only `Esc`/`Tab` check for the About modal, and only `Space`/`x` check table focus. Consequences (UX review O-06/O-07/O-18, plus explore findings):

- `d` removes a registered path while the Packages table is focused, though `statusHelpTable` never lists it — the help bar lies.
- With About open, `a` stacks the add-path input on top of the modal (zombie state) and `d` still deletes paths blind.
- `removeSelectedPath` persists immediately; no confirmation, no undo.
- `rescanSelected` does `st.marks = nil` silently — marks across all projects of the source are lost without warning.
- Help bars are two hand-written strings (`statusHelpTree`, `statusHelpTable`) with a hardcoded 106-column allocation; on narrow terminals the message zone collapses.
- Source rows render `3M 5m 2p │ 1C 2H 3M`: `M` means major on the left and moderate on the right of the same line, and the spec text ("3 major, 5 minor, 2 patch") already disagrees with the rendering.

The `ui` package currently has zero tests. All decisions below were closed with the user during explore (Q1–Q9 + 4 follow-ups).

## Goals / Non-Goals

**Goals:**
- Keys only fire in the contexts where they are advertised; the help bar cannot drift from behavior (single source of truth).
- Destructive actions (`d` always, `r` when marks would be lost) require explicit confirmation.
- Severity counters use two disjoint alphabets (shapes for semver, letters for audit); legend lives in About.
- Out-of-context keys teach instead of surprising or staying silent.
- First real tests for the `ui` layer (pure parts: keymap, guards, summaries).

**Non-Goals:**
- No `?` cheat-sheet modal (UX-03, next batch — the keymap table makes it nearly free later).
- No general message auto-expiry (UX-12) — only the narrow TTL for context hints.
- No full responsive layout (UX-04) — only width-selected help variants.
- No sort/filter, no spinner, no per-source counters (other backlog items).
- No change to the read-only invariant, config format, or core packages.

## Decisions

### D1 — Five contexts, resolved from UI state

`global`, `tree`, `table-packages`, `table-vulns`, `modal`. A `currentContext()` function derives the context from existing state (`pages.HasPage(...)`, `tableFocused`, `showVulns`) — no new state field to keep in sync. `table-vulns` is a real context (not a bool guard inside `table`): a mode where two keys die must be a context, or the generated help lies again.
*Alternative rejected:* tree/table only with ad-hoc modal checks — that is the current design and the source of the About leak.

### D2 — Declarative keymap as single source of truth

A package-level table `[]binding{key/rune, contexts, action, helpLabel, helpOrder}` drives three consumers: dispatch (`handleKey` walks the table), help-bar generation per context, and out-of-context hint lookup. Hand-written `statusHelp*` constants are deleted. The table and its lookup functions are pure and unit-testable without tcell.
*Alternative rejected:* `if a.tableFocused` guards inside the existing switch — minimal diff but help stays hand-written and drift returns.

### D3 — Modal context is inert, with two exceptions

In `modal` context: About responds to `q` (quit — required by the existing spec scenario), `Esc`/`h` (close). Confirmation modals respond only to their own buttons/`Esc`; `q` is deliberately inert there ("quit" while a question is on screen is more likely a mistyped answer than an exit intent). Everything else is swallowed — no more stacking add-path on top of About.

### D4 — Confirmation via a reusable `tview.Modal` helper

`confirm(title, text, onAccept)` wraps `tview.NewModal` with buttons [Cancel, Confirm], default focus on Cancel, `Esc` = cancel, mouse-clickable. Used by:
- `d`: always — "Stop tracking <path>? The folder on disk is not touched."
- `r`: only when `markCount(source) > 0` — "Rescanning <name> discards N marks across M projects." Count aggregates over all `projectIdx` entries of the source's `marks` map, not just the selected project.
The helper is generic so later changes (UX-19 error modal) can reuse it.
*Alternative rejected:* press-twice-to-confirm — invisible to anyone not reading the status line.

### D5 — Teaching hints with a scoped TTL

A key bound in another context sets a status hint ("d only works in Sources — Tab to go back") that auto-clears after ~5 s via `time.AfterFunc` + `QueueUpdateDraw`, guarded by a generation counter so a newer message is never clobbered by an older timer. TTL applies only to these hints; regular status messages keep today's behavior (general expiry is UX-12). Unbound keys stay silent.

### D6 — Counter encoding: shapes for semver, letters for audit

`updateSummary` → `▲3 ●5 ▪2` (major/minor/patch; colors unchanged red/yellow/green). `auditSummary` → `C1 H2 M3 L4` (unchanged letters, digit position normalized after the letter). `▪` chosen over `·` for patch (solid, legible); `▲ ● ▪` avoids the fold indicators' `▸ ▾` triangle family confusion. Same total width as today (~22 cols) — no layout pressure. Shapes double as a non-color severity channel (advances UX-21). The delta spec fixes the existing "3 major, 5 minor, 2 patch" scenario text to match.

### D7 — Legend in About

About modal gains two lines: `▲ major  ● minor  ▪ patch` and `C critical  H high  M moderate  L low`. Modal height grows accordingly. No new widget; the full `?` cheat sheet stays in UX-03.

### D8 — Width-aware generated help

Help text is generated from the keymap per context, in two variants: full and compact (top-priority bindings + `h help`). The bottom bar picks the variant on resize (checked in a draw hook / before render) so the message zone always keeps a minimum share. The `helpWidth = 106` constant dies; the help zone is sized to the chosen variant.

## Risks / Trade-offs

- [Keymap indirection makes simple keys less greppable] → table lives in one small file (`keymap.go`) with one line per binding; dispatch stays a single loop.
- [AfterFunc timer racing a newer status message] → generation counter checked inside `QueueUpdateDraw`; timers never clear a message they didn't set.
- [`q` inert in confirmations diverges from About behavior] → deliberate (D3); documented in the spec scenario so it is a contract, not an accident.
- [Shape glyphs (`▲ ● ▪`) may render poorly on exotic terminals] → all three are in Unicode blocks already used by the app (`▸ ▾ ✓ ✗ │ …`); no wider risk than today.
- [Aggregate mark count message can be long] → fixed short template, no package names listed.
- [Help variants add a resize-dependent code path] → variant selection is a pure function of width; unit-tested without a terminal.

## Migration Plan

Pure UI refactor, no persisted state touched. Single PR; revert = revert the commit. README keybindings table updated in the same change.

## Open Questions

None — all decisions closed during explore (Q1–Q9 + glyph set, hint TTL, help variants, `q`-in-confirm).
