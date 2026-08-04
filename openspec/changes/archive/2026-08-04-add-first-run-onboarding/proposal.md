# Proposal: add-first-run-onboarding

## Why

A first-time user lands on a dashboard whose only meaningful next step is registering a folder, but discovering that requires reading the passive onboarding text and finding the `a` key. The folder browser shipped in v0.9.0 makes the step itself trivial — opening it automatically on the very first launch removes the last bit of friction and completes the flow with zero reading.

## What Changes

- On a true first run — the config file was just created — the app opens the add-path folder browser automatically, once, with a welcome title ("Welcome — pick a folder to scan…"). Every other launch is unchanged.
- The trigger is config-file creation, not "zero paths registered": users who intentionally run with only the global source are never nagged.
- Closing the browser without adding a path shows a status hint (`press a to add a path, ? for all keys`); the existing passive onboarding panel remains the fallback. Adding a path behaves exactly as today.
- `config.Load` reports whether it created the file (internal plumbing for the trigger; on-disk behavior unchanged).

Out of scope:
- Auto-opening the `?` cheat sheet (rejected in exploration: information dump with no action) or chaining any second modal at startup.
- Any change to the passive onboarding panel or to the browser's behavior once open.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `dashboard-ui`: the "First run shows an empty-state call to action" requirement gains the active layer — auto-opened browser with welcome title on first launch only, and the close-without-adding hint.

## Impact

- `config`: `Load` additionally returns a created flag; `main.go` threads it into the UI.
- `ui`: startup hook in `Run` (open browser before the event loop when first run), welcome title variant, close-hint on the first-run browser.
- Specs: `dashboard-ui` delta only — the config file's on-disk behavior is untouched, so `config-store` requirements do not change.
