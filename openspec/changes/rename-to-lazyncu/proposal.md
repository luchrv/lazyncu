# Proposal: rename-to-lazyncu

## Why

The GitHub repository was renamed from `ncu-tui` to `lazyncu` (aligning with the lazygit/lazydocker TUI family). The codebase still carries the old identity everywhere: Go module path, binary name, config directory, error prefix, docs, and local git remote. GitHub's redirect keeps things working today, but `go install github.com/luchrv/lazyncu@latest` fails until the module path matches, and every reference to the old name is now misleading.

## What Changes

- Go module path: `github.com/luchrv/ncu-tui` → `github.com/luchrv/lazyncu` (go.mod + all internal imports across the Go packages).
- Binary/app identity: built binary becomes `lazyncu`; error prefix in `main.go` and doc comments updated; `.gitignore` entry updated.
- **BREAKING**: config directory renamed `~/.config/ncu-tui/` → `~/.config/lazyncu/` with **no migration** (option A): an existing old config is ignored and a fresh empty one is created on first launch; registered paths must be re-added manually.
- README: title, install/clone URLs, usage command.
- Live spec `config-store` updated to the new config path.
- Local git remote URL updated to the renamed repository.
- Explicitly untouched: `openspec/changes/archive/` (historical record) and Go file names (none carry the repo name).

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `config-store`: the requirement "Configuration is persisted as a TOML file" changes its mandated location from `$XDG_CONFIG_HOME/ncu-tui/config.toml` to `$XDG_CONFIG_HOME/lazyncu/config.toml`, with an explicit no-migration rule for the old directory.

## Impact

- All `.go` files (import paths), `go.mod`, `main.go`, `config/config.go` (+ its tests), `README.md`, `.gitignore`, `openspec/specs/config-store/spec.md`.
- Local environment: `git remote set-url`, optional rename of the working directory (user's choice, not part of the change).
- Users with an existing `~/.config/ncu-tui/config.toml` lose their registered paths (single-user impact accepted; re-adding takes seconds).
- No dependency, behavior, or API changes beyond naming.
