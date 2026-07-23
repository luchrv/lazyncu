# release-pipeline Specification

## Purpose
Automate lazyncu distribution: goreleaser builds multi-platform artifacts, a tag-triggered GitHub Actions workflow publishes tested GitHub Releases with a Homebrew cask pushed to the tap, and the whole process is documented for maintainers.
## Requirements
### Requirement: goreleaser builds multi-platform release artifacts
The release configuration SHALL build static binaries (`CGO_ENABLED=0`) for linux, darwin, and windows on amd64 and arm64, package them as tar.gz archives (zip on windows) including LICENSE and README, and publish checksums alongside.

#### Scenario: Snapshot build locally
- **WHEN** `goreleaser release --snapshot --clean` runs on a developer machine
- **THEN** all six platform archives and a checksums file are produced under `dist/` without publishing

### Requirement: Tag push triggers automated GitHub release
A GitHub Actions workflow SHALL run on pushes of tags matching `v*`, run the test suite as a gate, and on success run goreleaser to publish a GitHub Release with artifacts and an auto-generated changelog grouped by conventional-commit type (feat/fix/others; chore, docs, test, and ci excluded).

#### Scenario: Successful release
- **WHEN** tag `v0.1.0` is pushed to GitHub
- **THEN** the workflow tests, builds, and publishes a `v0.1.0` GitHub Release containing the six archives, checksums, and a grouped changelog

#### Scenario: Tests fail on tagged commit
- **WHEN** a `v*` tag is pushed and `go test ./...` fails
- **THEN** no release is published

### Requirement: Homebrew cask is published to the tap on release
The release pipeline SHALL push an updated Homebrew cask to the `luchrv/homebrew-tap` repository (under `Casks/`, via goreleaser `homebrew_casks` — the `brews` formula route is deprecated) on every release, authenticating with a token provided via the `HOMEBREW_TAP_TOKEN` secret, so users can `brew install luchrv/tap/lazyncu`. The cask SHALL strip the macOS quarantine attribute post-install since binaries are unsigned.

#### Scenario: Cask updated on release
- **WHEN** a release is published by the workflow
- **THEN** `luchrv/homebrew-tap` receives a commit updating `Casks/lazyncu.rb` to the new version and checksums

### Requirement: Release process is documented for one-time setup and routine releases
The repository SHALL include a release guide (`docs/RELEASING.md`) with step-by-step instructions for the one-time setup (create the tap repository, create a fine-grained PAT with contents read/write on the tap repo, store it as the `HOMEBREW_TAP_TOKEN` actions secret) and for cutting a release (tag and push).

#### Scenario: New maintainer performs setup
- **WHEN** a maintainer follows `docs/RELEASING.md` from scratch
- **THEN** they can complete tap/PAT/secret setup and cut a release without external help
