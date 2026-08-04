# Proposal: add-ncu-brew-dependency

## Why

lazyncu requires `ncu` (npm-check-updates) >= 18 on PATH at runtime, but no install channel provides it: a fresh `brew install luchrv/tap/lazyncu` produces a binary that immediately aborts at preflight with "ncu is not available". Homebrew can fix this for its channel — homebrew-core ships an `npm-check-updates` formula (currently 23.0.0, satisfying the >= 18 requirement) and casks support `depends_on formula:`, which GoReleaser's cask template already knows how to emit.

## What Changes

- The GoReleaser cask configuration (`.goreleaser.yaml`, `homebrew_casks` section) declares a dependency on the homebrew-core `npm-check-updates` formula, so `brew install luchrv/tap/lazyncu` installs ncu automatically.
- README Requirements/Install sections note that the Homebrew install brings ncu automatically, while release binaries and `go install` still require the manual `npm install -g npm-check-updates` step.
- No Go code changes. The preflight check and its install-hint error message remain as the safety net for non-Homebrew channels.

Out of scope (explicitly rejected during exploration):
- `npx npm-check-updates` runtime fallback in the scanner (unneeded complexity).
- Auto-installing ncu from the app (breaks the read-only invariant).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-pipeline`: the "Homebrew cask is published to the tap on release" requirement gains a dependency clause — the cask SHALL declare `depends_on` on the homebrew-core `npm-check-updates` formula so installing lazyncu via brew also installs ncu.

## Impact

- `.goreleaser.yaml`: add `dependencies` to the `homebrew_casks` entry.
- `README.md`: Requirements and Install sections.
- `luchrv/homebrew-tap` `Casks/lazyncu.rb`: regenerated with `depends_on` on the next release (no manual edit; takes effect when the next version is tagged).
- Side effect for users: brew pulls in `node` transitively via the formula. Harmless duplicate for users managing node via nvm/fnm.
