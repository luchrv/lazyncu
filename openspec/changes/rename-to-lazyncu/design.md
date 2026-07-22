# Design: rename-to-lazyncu

## Context

Mechanical rename following the GitHub repository rename `ncu-tui` → `lazyncu`. One real decision was settled during exploration: how to treat the existing config directory.

## Goals / Non-Goals

**Goals:**

- Single identity everywhere: module path, binary, config dir, docs, remote.
- `go install github.com/luchrv/lazyncu@latest` works.

**Non-Goals:**

- Config migration from the old directory (decision below).
- Rewriting archived OpenSpec artifacts — history keeps the name of its time.
- Renaming Go source files (none embed the repo name).
- Renaming the local working directory (user's personal choice, outside the repo).

## Decisions

### D1 — No config migration (option A)

`appDirName` changes to `lazyncu`; an existing `~/.config/ncu-tui/` is simply ignored. Alternatives considered: (B) one-time automatic migration copying the old file, (C) read-fallback to the old path writing to the new one. Rejected: the tool currently has a single known user, re-registering paths takes seconds, and both alternatives add code plus tests for a one-time event (YAGNI). If external users appear later, B can ship as its own change.

### D2 — Module path rename via go.mod + import sweep

`module github.com/luchrv/lazyncu` in go.mod and a mechanical update of every `github.com/luchrv/ncu-tui/...` import. No `replace` directive, no major-version games — the repo has no published tags yet, so consumers (none known) are unaffected.

### D3 — Binary name follows the module

`go build` derives the binary name from the module's last path element, so the rename is automatic; only `.gitignore` (`/lazyncu`) and docs need touching. The stderr error prefix in `main.go` changes to `lazyncu:` to match what the user actually typed.

## Risks / Trade-offs

- [User launches new binary and finds an empty dashboard] → Expected consequence of D1; README notes the breaking change; re-add paths with `a`.
- [Stale GitHub redirect hides a missed reference] → Final `grep -r ncu-tui` sweep (excluding `openspec/changes/archive/`) must come back empty before closing.
- [Old binary `ncu-tui` lingers in PATH/GOBIN] → Note in tasks to remove the previously installed binary if present.
