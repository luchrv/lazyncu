# Spec Delta

## REMOVED Requirements

### Requirement: Homebrew cask is published to the tap on release
**Reason**: The cask no longer installs the runtime scanning tool; npm-check-updates is no longer a dependency of lazyncu.
**Migration**: Homebrew users rely on `npm` (provided by the `node` formula) being present. No other action is required.

## MODIFIED Requirements

### Requirement: Release process is documented for one-time setup and routine releases
The repository SHALL include a release guide (`docs/RELEASING.md`) with step-by-step instructions for the one-time setup (create the tap repository, create a fine-grained PAT with contents read/write on the tap repo, store it as the `HOMEBREW_TAP_TOKEN` actions secret) and for cutting a release (tag and push). The README SHALL state that `lazyncu` requires `npm` (and `pnpm` for pnpm projects) but not npm-check-updates, for every install channel.

#### Scenario: New maintainer performs setup
- **WHEN** a maintainer follows `docs/RELEASING.md` from scratch
- **THEN** they can complete tap/PAT/secret setup and cut a release without external help

#### Scenario: README distinguishes install channels
- **WHEN** a user reads the README Requirements/Install sections
- **THEN** they learn that npm-check-updates is not required, and that `npm` (plus `pnpm` for pnpm projects) is needed for scanning and auditing

## ADDED Requirements

### Requirement: Homebrew cask is published to the tap without npm-check-updates
The release pipeline SHALL push an updated Homebrew cask to the `luchrv/homebrew-tap` repository (under `Casks/`, via goreleaser `homebrew_casks` — the `brews` formula route is deprecated) on every release, authenticating with a token provided via the `HOMEBREW_TAP_TOKEN` secret, so users can `brew install luchrv/tap/lazyncu`. The cask SHALL NOT depend on the `npm-check-updates` formula. The cask SHALL strip the macOS quarantine attribute post-install since binaries are unsigned.

#### Scenario: Cask updated on release
- **WHEN** a release is published by the workflow
- **THEN** `luchrv/homebrew-tap` receives a commit updating `Casks/lazyncu.rb` to the new version and checksums

#### Scenario: Fresh brew install does not pull npm-check-updates
- **WHEN** a user runs `brew install luchrv/tap/lazyncu`
- **THEN** Homebrew does not install the `npm-check-updates` formula as a dependency
