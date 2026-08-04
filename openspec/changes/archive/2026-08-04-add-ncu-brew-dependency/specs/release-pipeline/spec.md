# release-pipeline Delta Specification

## MODIFIED Requirements

### Requirement: Homebrew cask is published to the tap on release
The release pipeline SHALL push an updated Homebrew cask to the `luchrv/homebrew-tap` repository (under `Casks/`, via goreleaser `homebrew_casks` — the `brews` formula route is deprecated) on every release, authenticating with a token provided via the `HOMEBREW_TAP_TOKEN` secret, so users can `brew install luchrv/tap/lazyncu`. The cask SHALL declare a `depends_on` dependency on the homebrew-core `npm-check-updates` formula so that installing lazyncu via Homebrew also installs the `ncu` binary it requires at runtime. The cask SHALL strip the macOS quarantine attribute post-install since binaries are unsigned.

#### Scenario: Cask updated on release
- **WHEN** a release is published by the workflow
- **THEN** `luchrv/homebrew-tap` receives a commit updating `Casks/lazyncu.rb` to the new version and checksums

#### Scenario: Fresh brew install provides ncu
- **WHEN** a user without `ncu` on PATH runs `brew install luchrv/tap/lazyncu`
- **THEN** Homebrew installs the `npm-check-updates` formula as a dependency and `lazyncu` passes its preflight check on first launch

### Requirement: Release process is documented for one-time setup and routine releases
The repository SHALL include a release guide (`docs/RELEASING.md`) with step-by-step instructions for the one-time setup (create the tap repository, create a fine-grained PAT with contents read/write on the tap repo, store it as the `HOMEBREW_TAP_TOKEN` actions secret) and for cutting a release (tag and push). The README SHALL state that the Homebrew install provides `ncu` automatically, while the release-binary and `go install` channels require installing npm-check-updates manually.

#### Scenario: New maintainer performs setup
- **WHEN** a maintainer follows `docs/RELEASING.md` from scratch
- **THEN** they can complete tap/PAT/secret setup and cut a release without external help

#### Scenario: README distinguishes install channels
- **WHEN** a user reads the README Requirements/Install sections
- **THEN** they learn that `brew install luchrv/tap/lazyncu` includes ncu, and that other channels need `npm install -g npm-check-updates`
