# Design: add-version-and-release

## Context

lazyncu is a Go TUI (tview) with no version identity. The binary is built three ways: `make build` (developers), `go install github.com/luchrv/lazyncu@latest` (users), and — after this change — goreleaser (GitHub Releases + Homebrew). The UI already routes everything through `tview.Pages` (`ui/app.go`), and the bottom line is split into a transient message zone (left) and a fixed-width help bar (right, `helpWidth = 86`). Repo lives on GitHub (`github.com/luchrv/lazyncu`); no tags, no CI exist yet.

Process constraint from the user: nothing is committed or pushed until they confirm the Homebrew tap setup (external repo + PAT + secret) is done, so the first tag push produces a working release in one shot.

## Goals / Non-Goals

**Goals:**
- Binary knows its version, commit, and build date in all three build paths.
- `--version` flag and an in-TUI About modal expose it.
- Tag push `v0.1.0` triggers a fully automated release: multi-platform archives, checksums, changelog, Homebrew formula.
- A written guide lets the user perform the one-time tap/PAT setup unassisted.

**Non-Goals:**
- Self-update or update-check from inside lazyncu.
- Package managers beyond Homebrew (no scoop, apt, AUR, npm).
- Signing/notarization of binaries.
- Version migration logic (config format is unversioned and unchanged).

## Decisions

### D1 — Hybrid version resolution: ldflags + `debug.ReadBuildInfo()` fallback

New `version` package exposes `version.Info() (version, commit, date string)`. Package-level vars `version = "dev"`, `commit = "none"`, `date = "unknown"` are overridden via `-ldflags "-X ..."` by both the Makefile and goreleaser (goreleaser sets them by default convention). When `version` is still `"dev"`, fall back to `debug.ReadBuildInfo()`: `Main.Version` covers `go install @vX.Y.Z`, and `vcs.revision`/`vcs.time` settings cover source builds.

- *Alternative — ldflags only*: `go install @latest` would always report "dev". Rejected.
- *Alternative — BuildInfo only*: `make build` from a working tree reports "(devel)". Rejected.
- This is the lazygit/gh pattern; ~15 lines.

### D2 — `--version` before TUI startup, full format

Manual check of `os.Args[1]` in `main.go` for `--version`/`-version` (stdlib `flag` not needed for a single flag; avoids claiming flag namespace). Prints one line and exits 0:

```
lazyncu v0.1.0 (commit abc1234, built 2026-07-22T15:04:05Z)
```

Runs before config load and preflight — version must work even with a broken config or missing `ncu`.

### D3 — About modal as a `tview.Pages` page

`h` adds an `about` page (centered flex-wrapped TextView, ~50×12) on top of the main layout — same pattern the app already uses for its input dialogs. Content: app name, version, commit, build date, repo URL, license (Apache-2.0). Close: `Esc` or `h`. `q` stays global quit (the global input capture keeps precedence). Opening while an input dialog is active is a no-op (`h` only handled in the main-page key handler).

- *Alternative — `tview.Modal`*: button-oriented, awkward for a text card. Rejected.

### D4 — Help bar gains `[h] about`; width recomputed

Append `[yellow]h[-] about` to `statusHelp` and bump `helpWidth` to fit the new text exactly (measured from the rendered string, tags stripped). Keeps the existing rule: bare `[x]` literals never written into dynamic-colors TextViews.

### D5 — goreleaser: 6 targets, archives, changelog, brew

`.goreleaser.yaml` (v2 schema):
- Builds: `linux/darwin/windows × amd64/arm64`, `CGO_ENABLED=0`, default ldflags convention (injects D1 vars).
- Archives: `tar.gz` (zip for windows), containing binary + LICENSE + README.
- `checksums`, `snapshot` naming for local dry-runs.
- Changelog: `use: github`, groups by conventional-commit prefixes (feat/fix/others), excludes `chore`, `docs`, `test`, `ci`.
- `homebrew_casks` (the `brews` formula route is deprecated in goreleaser v2): cask pushed to `luchrv/homebrew-tap` under `Casks/`, token from env `HOMEBREW_TAP_TOKEN`, post-install hook strips the quarantine attribute (unsigned binaries).
- `force_token: github` + explicit `release.github`: a stray `GITLAB_TOKEN` in a developer environment would otherwise win auto-detection and render gitlab.com URLs (bit us in local snapshot testing).

- *Alternative — hand-rolled release script + matrix Action*: more code, no brew/changelog for free. Rejected.

### D6 — GitHub Actions release workflow on `v*` tags

`.github/workflows/release.yml`: triggers on `push: tags: ['v*']`; steps: checkout (`fetch-depth: 0`, needed for changelog), setup-go (stable), `goreleaser/goreleaser-action@v6` with `GITHUB_TOKEN` (releases) + `HOMEBREW_TAP_TOKEN` (tap push). Also run `go test ./...` before goreleaser as a release gate.

### D7 — Makefile ldflags + `release-check` helper

`make build` becomes `go build -ldflags "-X ...version=$(git describe --tags --always --dirty) -X ...commit=$(git rev-parse --short HEAD) -X ...date=$(date -u +%FT%TZ)" .` producing the `lazyncu` binary. Add `make release-check` running `goreleaser release --snapshot --clean` for local pipeline validation without tagging.

### D8 — One-shot release gate (process, not code)

Implementation lands locally; `docs/RELEASING.md` documents the one-time setup: create `luchrv/homebrew-tap` repo (public, empty, default branch `main`), create fine-grained PAT scoped to that repo with `contents: read/write`, store it as `HOMEBREW_TAP_TOKEN` actions secret on `luchrv/lazyncu`. Only after the user confirms completion: single commit → tag `v0.1.0` → push branch + tag → Action releases.

## Risks / Trade-offs

- [First release fails in CI (untestable locally: tap push, GH token)] → `make release-check` snapshot run validates builds/archives/formula rendering locally; workflow gates on tests; worst case, delete tag+release, fix, re-tag.
- [PAT misconfigured → brew step fails after release published] → guide includes verification steps; goreleaser fails loudly; formula push is retryable by re-running the job.
- [`helpWidth` overflow on narrow terminals] → help bar already truncates on narrow terminals today; width bump is marginal (~+11 chars). Accepted.
- [`--version` parsing by hand misses combined flags] → only one flag exists; YAGNI. Revisit if a second flag ever appears.
- [goreleaser v2 schema drift] → pin action to major v6 and validate with `goreleaser check` in `release-check`.

## Open Questions

None — all decisions settled in explore session (2026-07-22).
