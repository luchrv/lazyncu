# Proposal: add-version-and-release

## Why

lazyncu has no version identity: the binary cannot report what version it is, users cannot tell releases apart, and there is no distribution channel beyond `go install` from source. To ship v0.1.0 to end users we need version stamping, a way to see it (CLI and TUI), and an automated release pipeline.

## What Changes

- Add version metadata (version, commit, build date) resolved via a hybrid strategy: ldflags injection for `make build` and goreleaser builds, with `debug.ReadBuildInfo()` fallback for `go install @latest`.
- Add `lazyncu --version` CLI flag printing the full version line (version, commit, date).
- Add an About modal in the TUI, opened with `h`, showing app name, version, commit, build date, repo URL, and license. Closes with `Esc` or `h`; `q` remains global quit.
- Widen the bottom help bar and add the `[h] about` hint.
- Add goreleaser config: linux/darwin/windows × amd64/arm64, archives, checksums, auto-changelog from conventional commits, Homebrew tap (`luchrv/homebrew-tap`).
- Add GitHub Actions workflow that runs goreleaser on `v*` tag push.
- Add step-by-step release guide (`docs/RELEASING.md`) covering Homebrew tap repo creation, PAT, and secret setup.
- Update Makefile `build` target to inject version via ldflags; update README.

## Capabilities

### New Capabilities

- `app-version`: version metadata resolution (ldflags + BuildInfo fallback) and the `--version` CLI flag.
- `release-pipeline`: goreleaser configuration, GitHub Actions release workflow, Homebrew tap publishing, and the documented release process.

### Modified Capabilities

- `dashboard-ui`: new About modal on `h` (content, open/close behavior, `q` stays global quit) and the `[h] about` hint added to the help bar.

## Impact

- **Code**: `main.go` (flag parsing, version package wiring), new `version/` package, `ui/` (modal, input handling, help bar width).
- **Build**: `Makefile` (ldflags), new `.goreleaser.yaml`, new `.github/workflows/release.yml`.
- **Docs**: new `docs/RELEASING.md`, README install/version sections.
- **External**: new GitHub repo `luchrv/homebrew-tap`, fine-grained PAT stored as `HOMEBREW_TAP_TOKEN` secret (user-performed, guided).
- **Process constraint**: no commit/push/tag until the user confirms the Homebrew tap setup guide is completed, so the first tagged release works in one shot.
