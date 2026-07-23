# app-version Specification (delta)

## ADDED Requirements

### Requirement: Binary resolves its version metadata in every build path
The system SHALL resolve version, commit, and build date via ldflags-injected variables, and when those are absent (values still at their defaults) SHALL fall back to `debug.ReadBuildInfo()` (module version for `go install @version`, vcs settings for source builds). Resolution MUST never fail: unknown fields degrade to `dev` / `none` / `unknown`.

#### Scenario: Built with ldflags (make build or goreleaser)
- **WHEN** the binary is built with `-X` ldflags setting version, commit, and date
- **THEN** the resolved metadata reports exactly the injected values

#### Scenario: Installed via go install @version
- **WHEN** the binary is built without ldflags via `go install github.com/luchrv/lazyncu@vX.Y.Z`
- **THEN** the resolved version is `vX.Y.Z` taken from build info

#### Scenario: Bare source build
- **WHEN** the binary is built with plain `go build` and no vcs/module version is available
- **THEN** metadata degrades to `dev`, `none`, and `unknown` without error

### Requirement: --version flag prints full version line and exits
The system SHALL accept `--version` (and `-version`) as the first CLI argument, print a single line with app name, version, commit, and build date to stdout, and exit with code 0 — before loading config or running preflight checks.

#### Scenario: Version requested
- **WHEN** the user runs `lazyncu --version`
- **THEN** stdout shows `lazyncu <version> (commit <commit>, built <date>)` and the process exits 0

#### Scenario: Version works despite broken environment
- **WHEN** the user runs `lazyncu --version` with a corrupt config file or `ncu` not installed
- **THEN** the version line is still printed and the process exits 0
