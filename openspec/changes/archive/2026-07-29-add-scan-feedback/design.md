# Design: add-scan-feedback

## Context

Batch 3 of the UX review (UX-10/11/12) on top of v0.4.0. Decisions closed with the user: Q1=A (spinner in tree rows and the Packages loading message), Q2=A (dedicated fixed-width progress segment in the bottom bar), Q3=A (`info`/`ok`/`warn` expire ~5 s, `error` persists), Q4=A (level-derived icon + color, call sites drop inline tags). The generation counter (`msgGen`) and TTL mechanics from `add-keymap-safety` are the base for expiry.

## Goals / Non-Goals

**Goals:**
- A scanning app that visibly ticks: per-source spinner plus `scanning N/M` aggregate.
- One place that decides how a message looks (level → icon + color), surviving monochrome terminals.
- No stale messages: everything but errors clears itself.

**Non-Goals:**
- No per-source elapsed time or ETA.
- No scan cancellation.
- No message history/log view.
- No changes to scanning, keymap contexts, or the read-only invariant.

## Decisions

### D1 — One self-stopping spinner goroutine

`ensureSpinner()` (idempotent via a `spinning` flag) starts a goroutine with a ~120 ms ticker only when some source is loading. Each tick applies `spinFrame++` and refreshes tree/detail/progress inside `QueueUpdateDraw`; when nothing is loading anymore the closure clears the flag and the goroutine exits. Frames: `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`. Called from every path that sets a source loading (launch, rescan, add-path) via `refreshAll`.
*Alternative rejected:* per-source tickers — one clock is enough; frames are global anyway.

### D2 — Loading texts take the frame as data

`sourceText` gains the spinner glyph parameter (pure, testable); the Packages-panel loading message uses the same glyph. No widget knows about the ticker.

### D3 — Progress is a third bottom-bar segment

`bottom` flex becomes messages | progress | help. `refreshProgress()` computes `done/total` from source states: hidden (width 0) when nothing is loading, otherwise `scanning N/M` sized to its printable width. Independent of the message zone, so a `copied:` toast never hides it (Q2=A).

### D4 — Levels replace inline tags at call sites

`setStatus(level, format, args...)` with `msgInfo`/`msgOK`/`msgWarn`/`msgError`. Decoration in one function: `· text` gray, `✓` green + plain text, `! text` yellow, `✗ text` red. Call sites lose their `[red]…[-]` strings. Teaching hints become plain `msgWarn` messages — the separate `showHint` path dissolves into the general mechanism.

### D5 — Expiry generalizes the hint TTL (Q3=A)

`setStatus` schedules the existing generation-guarded `AfterFunc` clear for every level except `msgError`, which persists until replaced. `clearHintIfCurrent` is renamed to `clearIfCurrent`; expiry also clears `lastMsg`, so `m` restores only a message that is still live (an expired message does not resurrect).

## Risks / Trade-offs

- [Ticker goroutine leaks in tests (queue never drained on a non-running app)] → tview's update channel is buffered; leaked goroutines are inert and die with the test binary. Production exit path is the self-stop closure.
- [Full tree refresh every 120 ms] → the tree is a handful of nodes; rendering is O(sources+projects) string formatting. Negligible.
- [Errors persisting could go stale too] → they are replaced by any newer message; a disappearing error is worse than a lingering one.
- [Hint tests asserting exact text] → hints now carry the `!` prefix; assertions use `Contains` on the substance, unaffected.

## Migration Plan

Pure UI change on `feat/add-scan-feedback`; single PR; revert = revert.

## Open Questions

None — Q1–Q4 closed; minor calls (frames, 120 ms cadence, `N/M` = done/total, `m` semantics) fixed in this design.
