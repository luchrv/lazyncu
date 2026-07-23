# Proposal: add-readme-demos

## Why

The README is text-only: a new visitor cannot see what lazyncu looks like or why it beats running `ncu` by hand. Animated demos (lazygit-style) plus status badges communicate the value in seconds and make the project look maintained. Captures must be reproducible — hand-recorded GIFs rot the moment the UI changes.

## What Changes

- Add status badges to the README top: GitHub release, Go Report Card, license (Apache-2.0), release downloads, Homebrew tap version (shields.io).
- Add three animated GIF demos to the README: hero (launch + parallel scan streaming in) after the intro, vulnerability view (`v` + dependency chain) in the audit context, add-path (`a` + immediate scan) in the configuration context.
- Add scriptable demo recording with charmbracelet VHS: versioned `assets/tapes/*.tape` files rendering to `assets/demo/*.gif`, regenerable with `make demos`.
- Add synthetic demo fixtures under `demo/fixtures/`: npm projects with pinned outdated dependencies (mixed major/minor/patch) and known-vulnerable packages to populate the vulnerability view.
- Isolate recordings from the real user config: tapes run with a temporary `XDG_CONFIG_HOME` and a pre-seeded config pointing at the fixtures.
- Add `docs/DEMOS.md`: step-by-step guide — install VHS, tape anatomy, regenerate all demos, add a new demo.

## Capabilities

### New Capabilities

- `demo-recording`: reproducible VHS-based demo capture — tapes, fixtures, config isolation, `make demos` regeneration, contributor guide, and how the README presents demos and badges.

### Modified Capabilities

(none — no existing spec's requirements change; README additions are presentation covered by `demo-recording`.)

## Impact

- **Repo**: new `assets/tapes/`, `assets/demo/` (GIF binaries committed), `demo/fixtures/`, `docs/DEMOS.md`; `README.md` and `Makefile` modified.
- **Tooling**: VHS becomes a dev-only dependency (`brew install vhs`); recording needs network (real `ncu` + `npm audit` runs against fixtures).
- **Known trade-off**: GIF content ages as registries move on — accepted, regenerated on demand via `make demos`.
- **No application code changes.**
