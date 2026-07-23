# Tasks: add-version-and-release

## 1. Version package (TDD)

- [x] 1.1 Write tests for `version` package: ldflags values win; fallback to BuildInfo; degraded defaults (`dev`/`none`/`unknown`); formatted one-line string
- [x] 1.2 Implement `version` package: package vars + `Info()` + `String()` with `debug.ReadBuildInfo()` fallback

## 2. --version flag (TDD)

- [x] 2.1 Write test covering `--version`/`-version` arg detection (extracted helper, testable without running the TUI)
- [x] 2.2 Implement flag handling in `main.go` before config load/preflight; print `lazyncu <version> (commit <c>, built <d>)`, exit 0

## 3. About modal in TUI

- [x] 3.1 Add About modal page (centered card: name, version, commit, date, repo URL, license) opened with `h` from the main dashboard; no-op while input dialog active
- [x] 3.2 Wire close on `Esc`/`h` with focus restore; verify `q` still quits globally with modal open
- [x] 3.3 Append `[yellow]h[-] about` to `statusHelp` and resize `helpWidth` to fit
- [x] 3.4 Manual TUI smoke test: open/close modal, q-quit from modal, help bar fully visible

## 4. Build & release pipeline

- [x] 4.1 Update Makefile: `build` injects version/commit/date via ldflags; add `release-check` target (`goreleaser check` + `goreleaser release --snapshot --clean`)
- [x] 4.2 Create `.goreleaser.yaml`: 6 targets (linux/darwin/windows × amd64/arm64, CGO off), archives with LICENSE+README, checksums, conventional-commit changelog groups, brews section → `luchrv/homebrew-tap` `Formula/` with `HOMEBREW_TAP_TOKEN`
- [x] 4.3 Create `.github/workflows/release.yml`: on `v*` tag push → checkout fetch-depth 0 → setup-go → `go test ./...` → goreleaser-action@v6 with tokens
- [x] 4.4 Validate locally: `make release-check` produces all archives + rendered cask in `dist/`

## 5. Documentation

- [x] 5.1 Write `docs/RELEASING.md`: one-time setup guide (create `luchrv/homebrew-tap` public repo, fine-grained PAT with contents read/write on tap repo, add `HOMEBREW_TAP_TOKEN` secret to lazyncu) + routine release steps (tag → push)
- [x] 5.2 Update README: `--version`, `h` about key, install via `brew install luchrv/tap/lazyncu`, GitHub Releases, `go install`

## 6. Verification & release gate

- [x] 6.1 Run `make check` (fmt, vet, race, cover ≥80% pure packages) — all green
- [x] 6.2 ⛔ GATE — user confirms `docs/RELEASING.md` one-time setup completed (tap repo + PAT + secret). NO commit/push/tag before this
- [ ] 6.3 Single shot: commit (conventional message) → tag `v0.1.0` → push branch + tag
- [ ] 6.4 Verify release: GitHub Action green, Release `v0.1.0` with 6 archives + checksums + changelog, tap formula updated, `brew install luchrv/tap/lazyncu` works
