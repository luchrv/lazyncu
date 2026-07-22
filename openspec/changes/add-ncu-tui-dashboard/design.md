# Design: add-ncu-tui-dashboard

## Context

Greenfield Go project (repository contains only README/LICENSE). The application wraps the locally installed `npm-check-updates` CLI (`ncu`, v18+) and presents its results in a terminal dashboard. All decisions below were settled during exploration with the user:

- Read-only dashboard; it suggests update commands but never runs them.
- Paths persisted in TOML (SQLite discarded — no cache/history requirement, YAGNI).
- Scan strategy auto-detected per path.
- Full parallel scan on launch.
- Severity (major/minor/patch) computed and color-coded.
- `tview` chosen as the TUI framework (user preference, validated below).

## Goals / Non-Goals

**Goals:**

- Single view answering "which projects need updates, and how urgent are they?".
- Zero per-path configuration: scan mode and package manager are inferred.
- Fast perceived startup: UI renders immediately; results stream in as scans finish.
- Business logic isolated in pure, unit-testable packages (80% coverage target).

**Non-Goals:**

- Applying updates (no `ncu -u`, no `npm install` execution).
- Scan-result caching, history, or scheduling.
- Watching the filesystem for new projects.
- Supporting package managers beyond npm/pnpm/yarn lockfile detection.
- Windows support beyond what tview/clipboard provide out of the box (target is macOS/Linux).

## Decisions

### D1 — TUI framework: tview over Bubble Tea

`tview` (with `tcell`) provides ready-made `Table` (fixed headers, row selection, scrolling), `Flex` layout and `Pages` — the entire UI of this app is tables and panes, so tview yields the UI nearly for free. Bubble Tea's advantages (Elm architecture, pure `Update` easy to test, immutability by design) are recovered through two compensating rules:

1. **Single choke point for async updates**: goroutines never touch widgets directly; they send results through one function that wraps `app.QueueUpdateDraw`. This confines tview's data-race footgun to one reviewed location.
2. **Thin UI layer**: all logic lives in pure packages (`config`, `detect`, `scanner`, `semver`, `command`); `ui/` only renders structs it is handed. Coverage targets are met in the pure packages; `ui/` holds no business logic.

### D2 — Persistence: TOML file, not SQLite

Requirements reduced to "a list of paths". A TOML file at `~/.config/ncu-tui/config.toml` (respecting `XDG_CONFIG_HOME`) is human-editable, diffable, and dependency-light. SQLite was considered and rejected: it only pays off with cached scan results or history, both out of scope. Library: `pelletier/go-toml/v2`.

Config shape:

```toml
timeout_ms = 30000   # optional, passed to ncu --timeout

[[paths]]
path = "~/VCProj/repos"    # tilde expanded at load
```

### D3 — Scan-mode auto-detection

Decision tree evaluated per registered path at scan time (never persisted, so it self-heals when a folder gains/loses projects):

```
path
 ├─ no package.json ──────────────────────────→ ncu --deep   (folder of projects)
 └─ package.json present
     ├─ "workspaces" field OR pnpm-workspace.yaml → ncu --deep   (monorepo)
     └─ otherwise ────────────────────────────→ ncu           (single project)
```

`--deep` is `--packageFile '**/package.json'` (node_modules excluded by default), so it also covers the mixed case of a folder containing monorepos — recursion is total.

### D4 — ncu invocation and parsing

- All scans use `--jsonUpgraded` for machine-readable output.
- Global: `ncu -g --jsonUpgraded` → flat map `{pkg: "new-version"}`. Current versions obtained via `npm ls -g --depth=0 --json` (ncu's upgraded map alone lacks the installed version needed for the semver diff).
- Single project: `ncu --jsonUpgraded --packageFile <path>/package.json` → flat map; current versions read from that `package.json`.
- Deep: `ncu --deep --jsonUpgraded` (cwd = path) → map keyed by each discovered `package.json` path; each key becomes a child project row in the dashboard.
- `exec.CommandContext` with configurable timeout; `ncu` exit codes and malformed JSON surface as per-source error states in the UI, never crash the app.

### D5 — Concurrency model

One goroutine per source (global + each registered path), fanned out at launch via `errgroup`/`sync.WaitGroup`. Each goroutine produces an immutable `ScanResult` struct and delivers it through the D1 choke point. UI shows a per-source spinner until its result (or error) arrives. No shared mutable state between scanner and UI; results are passed by value/ownership transfer.

### D6 — Semver severity

Computed in Go with `Masterminds/semver/v3`: parse current vs. new, classify jump as major/minor/patch (fallback bucket `other` for non-semver ranges/tags like `latest` or git URLs). Colors: red=major, yellow=minor, green=patch, gray=other. Per-project counters ("3 major, 5 minor, 2 patch") aggregate these for the project panel — the at-a-glance answer to "what must I update?".

### D7 — Update-command construction

Read-only dashboard shows the exact command for the selected context in a bottom command bar, copyable with a keybinding (`atotto/clipboard`):

- Global source: `npm install -g pkg1@v1 pkg2@v2 ...` rebuilt from scan results.
- Project row: `ncu -u && <install>` prefixed with the project directory, where `<install>` is inferred from the lockfile: `package-lock.json` → `npm install`, `pnpm-lock.yaml` → `pnpm install`, `yarn.lock` → `yarn`. No lockfile → default `npm install`.

### D8 — Package layout

```
main.go
config/    load/save TOML, path add/remove, validation      (pure)
detect/    scan-mode tree (D3) + package-manager detection  (pure)
scanner/   ncu/npm exec + JSON parsing → ScanResult          (pure logic; exec injected for tests)
semver/    severity classification + per-project counters    (pure)
command/   update-command builders (D7)                      (pure)
audit/     npm/pnpm audit exec + JSON parsing → AuditResult  (pure logic; exec injected for tests)
orchestrator/  scan+audit fan-out per source → event channel  (scanner/auditor injected for tests)
ui/        tview app: panels, table, command bar, choke point
```

### D9 — Vulnerability audit via npm/pnpm audit

ncu carries no vulnerability data (it only compares versions), so audit is a separate, complementary signal. Per project entry, the orchestrator (D5) launches an audit goroutine alongside the ncu scan:

- npm projects (`package-lock.json` or no lockfile): `npm audit --json` in the project directory.
- pnpm projects: `pnpm audit --json` (output format compatible with npm's).
- yarn projects: **deferred** — yarn classic emits NDJSON with a different shape; v1 shows an "audit not available" state instead of parsing a third format.
- Global source: **excluded** — `npm audit` requires a lockfile and does not support global installs; the UI states this explicitly.

Parsed output yields per-project counters by npm severity (critical/high/moderate/low) plus a detail list (package, severity, vulnerable range, fix availability, dependency chain). The chain — which direct dependency drags the vulnerable package in — is reconstructed from the audit report's `via` (cause) and `effects` (impact) graph: walk `effects` upward from the vulnerable package to a root-level dependency; direct dependencies are labeled "direct". Depth is bounded (chain truncated with `…` past 4 hops) to keep parsing cycle-safe and rows readable. Suggested fix command in the command bar: `npm audit fix` or `pnpm audit --fix` (displayed/copyable only — read-only rule D7 applies). Audit failure never poisons the version scan: the project still shows its ncu results with an "audit failed" badge. Rationale for including it: the dashboard's core question is "which project needs attention?", and a critical vulnerability outranks any major version jump.

`scanner` takes a command-runner interface so tests inject canned ncu JSON without executing binaries.

## Risks / Trade-offs

- [tview UI race conditions] → All widget mutations funneled through a single `QueueUpdateDraw` wrapper (D1); code review gate on any direct widget access from goroutines.
- [ncu absent or too old] → Startup preflight: check `ncu` on PATH and version >= 18; show actionable error screen ("npm install -g npm-check-updates") instead of failing mid-scan.
- [Slow scans on large monorepos (10–30s per --deep)] → Parallel fan-out + per-source spinners keep the UI responsive; configurable `timeout_ms` bounds worst case; a slow source never blocks others.
- [`ncu -g` upgraded map lacks installed versions] → Pair with `npm ls -g --depth=0 --json`; if that fails, degrade gracefully: show new version with severity `unknown` instead of erroring the whole source.
- [Non-semver version specs (git URLs, tags, `*`)] → `other` bucket, gray color, excluded from counters; never a parse crash.
- [tview hard to unit test] → Accepted trade-off; mitigated by thin-UI rule (D1.2) — coverage measured on pure packages.
- [Clipboard unavailable (headless/SSH)] → Copy failure shows non-fatal status message; command remains visible for manual copy.
- [Audit needs network + registry advisory endpoint] → Same timeout budget as scans; failure degrades to "audit failed" badge per project, ncu results unaffected.
- [`npm audit` exits non-zero when vulnerabilities exist] → Exit code 1 with parseable JSON is a *successful* audit, not an error; only unparseable output or exec failure counts as failure.
- [Audit coverage gaps (yarn, global) may read as "no vulnerabilities"] → Explicit "audit not available" state, visually distinct from "0 vulnerabilities".

## Open Questions

- None blocking. Deferred ideas (explicitly out of scope): `--target` selection (minor/patch-only views), result caching, yarn audit parsing (NDJSON format). Manual per-source rescan was initially deferred but promoted to a requirement during implementation (`r` key, guarded against overlapping scans of the same source).
