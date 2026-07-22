# Tasks: add-ncu-tui-dashboard

TDD throughout: for every pure package, write the failing tests first (table-driven), then implement to green, then refactor. Coverage target 80%+ on `config`, `detect`, `scanner`, `semver`, `command`, `audit`.

## 1. Project Setup

- [x] 1.1 Initialize Go module (`go mod init`), add dependencies: `rivo/tview`, `gdamore/tcell/v2`, `pelletier/go-toml/v2`, `Masterminds/semver/v3`, `atotto/clipboard`
- [x] 1.2 Create package skeleton: `main.go`, `config/`, `detect/`, `scanner/`, `semver/`, `command/`, `audit/`, `ui/`
- [x] 1.3 Set up lint/test tooling: `gofmt`, `go vet`, `golangci-lint` config; `go test -race` wired into a Makefile or task runner

## 2. Config Store (pure)

- [x] 2.1 Tests: load/save TOML round-trip, XDG_CONFIG_HOME fallback, first-launch creation, malformed TOML error, `timeout_ms` default (30000) and override
- [x] 2.2 Tests: add path (tilde expansion, path cleaning, non-existent rejection, duplicate rejection), remove path, immediate persistence
- [x] 2.3 Implement `config` package to green (immutable ops: load/add/remove return new Config values)

## 3. Detection (pure)

- [x] 3.1 Tests: scan-mode tree — no package.json → deep; package.json + `workspaces` field → deep; package.json + pnpm-workspace.yaml → deep; plain package.json → single; re-evaluation per scan
- [x] 3.2 Tests: package-manager detection — pnpm-lock.yaml → pnpm, yarn.lock → yarn, package-lock.json → npm, none → npm, multiple lockfiles precedence (pnpm > yarn > npm)
- [x] 3.3 Implement `detect` package to green

## 4. Semver Analysis (pure)

- [x] 4.1 Tests: severity classification — major/minor/patch, range-prefix stripping (^, ~, >=), non-semver (git URL, tag, `*`) → other, unknown current → other
- [x] 4.2 Tests: per-project counters — mixed severities, zero counters when up to date, `other` excluded
- [x] 4.3 Implement `semver` package to green (wrap `Masterminds/semver/v3`)

## 5. Scanner (pure logic, injected exec)

- [x] 5.1 Define command-runner interface and `ScanResult`/`Project`/`Package` immutable structs; fake runner for tests with canned JSON fixtures
- [x] 5.2 Tests: ncu preflight — missing binary, version < 18, version >= 18 OK
- [x] 5.3 Tests: global scan — parse `ncu -g --jsonUpgraded` + `npm ls -g --depth=0 --json` merge; `npm ls` failure degrades to unknown severity without failing the source
- [x] 5.4 Tests: single scan — parse flat `--jsonUpgraded` map, current versions from package.json, up-to-date project yields zero packages
- [x] 5.5 Tests: deep scan — parse path-keyed JSON into child entries with relative labels; nested monorepo workspaces appear as entries
- [x] 5.6 Tests: failure isolation — non-zero exit, malformed JSON, timeout via context → per-source error result, never panic
- [x] 5.7 Implement `scanner` package to green (`exec.CommandContext`, configurable timeout)

## 6. Update Commands (pure)

- [x] 6.1 Tests: global command `npm install -g pkg@ver ...` ordering and formatting; empty scan → no command
- [x] 6.2 Tests: project command `cd <dir> && ncu -u && <install>` per package manager (npm/pnpm/yarn), default npm
- [x] 6.3 Implement `command` package to green

## 7. Vulnerability Audit (pure logic, injected exec)

- [x] 7.1 Define `AuditResult`/`Vulnerability` immutable structs; canned npm/pnpm audit JSON fixtures for tests
- [x] 7.2 Tests: parse `npm audit --json` and `pnpm audit --json` — severity counters (critical/high/moderate/low), detail list (package, severity, range, fixAvailable), zero-vulnerability result
- [x] 7.3 Tests: dependency-chain reconstruction from `via`/`effects` — transitive chain (`lodash ← express`), direct dependency labeled "direct", cycle safety, truncation past 4 hops
- [x] 7.4 Tests: exit code 1 with valid JSON → successful audit; unparseable output or exec failure/timeout → audit-failed result (never fails the source)
- [x] 7.5 Tests: coverage gaps — yarn project and global source → "not available" result, no command executed
- [x] 7.6 Tests: fix-command builders — `cd <dir> && npm audit fix` / `cd <dir> && pnpm audit --fix`; none when zero vulnerabilities
- [x] 7.7 Implement `audit` package to green

## 8. Concurrency Orchestration

- [x] 8.1 Tests: fan-out — one goroutine per source, results delivered as they complete, slow source does not block fast ones, all results delivered exactly once
- [x] 8.2 Tests: per-project audit goroutine runs alongside the ncu scan; audit failure delivers audit-failed without affecting the scan result
- [x] 8.3 Implement scan+audit orchestrator (WaitGroup/errgroup) producing immutable results on a channel

## 9. UI (tview, thin layer)

- [x] 9.1 Build layout: Flex with sources/projects panel, package table (package/current/new/severity columns), command bar, status line
- [x] 9.2 Implement single dispatch choke point wrapping `QueueUpdateDraw`; connect orchestrator channel; per-source spinner → results/error states
- [x] 9.3 Severity rendering: row colors (red/yellow/green/gray) and per-project counters in the panel
- [x] 9.4 Selection wiring: panel selection drives package table and command bar (global vs project commands)
- [x] 9.5 Vulnerability rendering: per-project vuln counters next to semver counters; detail view (package, severity, range, fixAvailable, dependency chain / "direct"); distinct states for "0 vulnerabilities", "audit not available", "audit failed"; fix command in command bar
- [x] 9.6 Clipboard copy keybinding with confirmation message; non-fatal error when clipboard unavailable
- [x] 9.7 Path management UI: add-path input (validate via config, scan new path immediately), remove-path keybinding
- [x] 9.9 Rescan keybinding (`r`) for the selected source (global or path), disabled with a message while that source is scanning
- [x] 9.10 Fold/unfold project lists per source (Enter), fold state persisted across refreshes, ▸/▾ indicator, selection moved up on collapse
- [x] 9.11 Toggle status-message zone visibility (`m`), last message restored on re-show, help zone unaffected
- [x] 9.8 Preflight error screen for missing/old ncu with install instructions

## 10. Integration & Polish

- [x] 10.1 Wire `main.go`: preflight → config load → orchestrator → UI run
- [x] 10.2 Manual end-to-end pass against real projects: single project, pnpm monorepo, folder of projects, global; verify update commands, audit counters, and fix commands shown are correct
- [x] 10.3 Verify coverage >= 80% on pure packages; `go test -race ./...` clean; `go vet` and lint clean
- [x] 10.4 Update README: install, usage, config format, keybindings, audit coverage notes (yarn/global), screenshots
