# Proposal: add-scan-feedback

## Why

Third batch of the UX review (`docs/UX-REVIEW.md`): make waiting legible. Scans routinely take tens of seconds, but loading is a static string indistinguishable from a hung app (O-13), nothing reports how many sources have finished (O-14), and status messages never expire and carry no severity — ten minutes later a stale `copied: …` still sits in the corner, and on a monochrome terminal an error reads exactly like a confirmation (O-15/O-16).

## What Changes

- **Animated scan spinner** (UX-10): sources that are scanning show a braille spinner (`⠋⠙⠹…`) in the tree row and in the Packages-panel loading message, driven by a single ticker (~120 ms) that runs only while something is loading and applies frames through the existing `QueueUpdateDraw` choke point.
- **Aggregate scan progress** (UX-11): a dedicated fixed-width segment in the bottom bar (between messages and key help) shows `scanning N/M` while any source is loading and disappears on completion. Transient messages never cover it.
- **Leveled, expiring status messages** (UX-12): `setStatus` takes a level — `info` (`·` gray), `ok` (`✓` green), `warn` (`!` yellow), `error` (`✗` red) — that derives icon and color in one place; call sites drop their inline color tags. `info`/`ok`/`warn` messages auto-expire after ~5 s via the existing generation-counter mechanism (which absorbs the teaching hints); `error` persists until replaced. `m` restores only a still-live message.

## Capabilities

### New Capabilities

<!-- none — everything lands in the existing dashboard-ui capability -->

### Modified Capabilities

- `dashboard-ui`: adds two requirements (aggregate scan progress; leveled auto-expiring status messages) and modifies two — per-source loading states (animated spinner) and the hide/show messages toggle (restores only a message that has not expired).

## Impact

- **Code**: `ui/` only — `app.go` (spinner ticker, leveled `setStatus`, expiry), `panel.go`/`detail.go` (spinner frame in loading texts), `layout.go` (progress segment), `input.go` (call-site levels, hint absorbed into the message mechanism). Core packages untouched.
- **Specs**: `dashboard-ui` delta (2 added, 2 modified requirements).
- **Tests**: pure parts — frame glyph cycling, progress text/visibility, level decoration, expiry guard reuse, call-site levels.
- **Docs**: README note on message levels; UX-REVIEW items UX-10/11/12 marked shipped.
- **No breaking changes**: read-only invariant, config, and keymap untouched.
