# Design: add-first-run-onboarding

## Context

See proposal.md — Why. Current mechanics:

- `config.Load` (`config/config.go:55`) creates the file on first launch and returns `(Config, error)` — creation is silent, so callers cannot distinguish first run from any other.
- `main.run()` loads the config and hands it to `ui.New(ctx, cfg, cfgPath, sc, auditor)`; `App.Run` starts the scan fan-out, refreshes, and enters `tv.Run()` (`ui/app.go:150`). Pages and focus set before the loop starts render on the first draw — the existing initial page proves the pattern.
- The browser (`ui/browser.go`) exposes `openAddPath` → `openBrowserAt(root)`; the modal title is set on the local Flex. Esc closes via `handleBrowserKey` → `closeAddPath`; adding goes through `addPath`, which emits its own status.

## Goals / Non-Goals

**Goals:**
- One-time active onboarding that costs an Esc to dismiss and nothing thereafter.
- Zero behavior change for existing users and for global-only users.

**Non-Goals:**
- Persisting any "onboarding seen" state — the config file's existence *is* the state.
- Touching the passive onboarding panel, the tree hints, or normal browser behavior.

## Decisions

**D1 — `config.Load` returns `(Config, bool, error)`; the bool is true only when the file was just created.**
The file's existence is already the natural first-run marker — no new sentinel file, no config field. Signature change touches `main.go` and config tests only.
Alternative considered: `cfg.Paths == 0` per launch — rejected in exploration (nags global-only users). Separate `WasCreated(path)` probe — racy and redundant.

**D2 — First-run flag threads `main` → `ui.New` param → `App` field; the hook lives at the top of `App.Run`.**
Before `tv.Run()` starts, when `firstRun` is true, open the browser with the welcome title. Pre-loop `AddPage`/`SetFocus` is safe (same mechanism as the initial page). No draw-queue dance needed.
Alternative considered: exported setter on `App` — one more way to configure the same thing; a constructor param is explicit and compile-checked at the single call site.

**D3 — `openBrowserAt` gains a title parameter (or a thin `openWelcomeBrowser` wrapper); the browser records `firstRun` to emit the dismiss hint.**
`pathBrowser` gets a `welcome bool`; `closeAddPath` (or the Esc path in `handleBrowserKey`) emits `press a to add a path, ? for all keys` via the existing status system only when the welcome browser closes without having added. Adding a path already emits `added X — scanning`, which supersedes any hint.

**D4 — Hint fires on dismissal, not on close-after-add.**
`addPath` runs after `closeAddPath` in both confirm routes, so the browser cannot know the add succeeded at close time. Instead: the Esc route emits the hint; the confirm routes never do. Input-focus Esc (via the InputField DoneFunc) counts as dismissal too.

## Risks / Trade-offs

- [User deletes config.toml and gets the welcome again] → Correct behavior, not a bug: recreating the file is a first run by definition.
- [First-run modal covers the initial scan feedback] → Global scan proceeds underneath; results render on close. The modal is dismissible instantly with Esc. Accepted.
- [`ui.New` signature grows] → One boolean at one call site (plus `newTestApp`); cheaper than a setter API.

## Migration Plan

No stored-data change. Existing users have a config file → flag is always false → identical behavior. Rollback = revert commit.

## Open Questions

None.
