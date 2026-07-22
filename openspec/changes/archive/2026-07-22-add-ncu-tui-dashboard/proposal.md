# Proposal: add-ncu-tui-dashboard

## Why

Keeping npm packages up to date across many projects (and globally installed tools) requires manually running `ncu` in each location and mentally aggregating the results. There is no single view that answers "which of my projects need updates, and how urgent are they?". This change creates a read-only terminal dashboard that runs `npm-check-updates` across all registered locations at once and presents the results in a single, color-coded view.

## What Changes

- New Go TUI application (`ncu-tui`) built with `tview`, acting as a dashboard for the locally installed `ncu` (npm-check-updates) CLI.
- Two package sources displayed side by side:
  - **Global packages**: results of `ncu -g`.
  - **Project packages**: results of running `ncu` against user-registered filesystem paths.
- Registered paths are stored in a TOML config file (`~/.config/ncu-tui/config.toml`); paths can be added and removed.
- Per-path auto-detection of the correct scan mode:
  - No `package.json` at the path → folder of multiple projects → `ncu --deep`.
  - `package.json` with a `workspaces` field or a `pnpm-workspace.yaml` present → monorepo → `ncu --deep`.
  - Plain `package.json` → single project → `ncu`.
- All sources are scanned in parallel on launch, with per-source loading feedback; `--deep` results expand into one dashboard entry per discovered `package.json`.
- Output is parsed from `ncu --jsonUpgraded`; the semver jump (major/minor/patch) is computed per package and color-coded (red/yellow/green), with per-project counters (e.g. "3 major, 5 minor, 2 patch").
- The dashboard is **read-only**: it never modifies any project. Instead it displays (and lets the user copy) the exact update command for the current context:
  - Global: `npm install -g pkg@x.y.z ...` reconstructed from scan results.
  - Project: `ncu -u && <install>` where `<install>` is chosen from the detected lockfile (`npm install`, `pnpm install`, or `yarn`).
- Vulnerability audit per project, run in parallel with the version scan: `npm audit --json` (npm projects) or `pnpm audit --json` (pnpm projects). Per-project vulnerability counters (critical/high/moderate/low), a detail view, and a suggested fix command (`npm audit fix` / `pnpm audit --fix`). Yarn projects and the global source show an "audit not available" state (v1 limitation: yarn's audit format differs; npm cannot audit global installs).

## Capabilities

### New Capabilities

- `config-store`: TOML-based persistence of user-registered project paths (load, add, remove, validate).
- `scan-detection`: auto-detection of the scan strategy for a path (single project, monorepo, folder of projects) and of the package manager from lockfiles.
- `package-scanning`: executing `ncu` (`-g`, plain, `--deep`) with JSON output, in parallel across sources, and parsing results into per-project package lists.
- `semver-analysis`: computing the upgrade severity (major/minor/patch) for each package and aggregating per-project counters.
- `update-commands`: building the context-appropriate, copyable update command (global npm install vs. per-project `ncu -u` + package-manager install).
- `vulnerability-audit`: running `npm audit`/`pnpm audit` per project, parsing severity counts and vulnerable-package details, and exposing them to the dashboard with a suggested fix command.
- `dashboard-ui`: tview-based terminal UI — source/project panel, package table with severity colors, command bar, loading states, and clipboard copy.

### Modified Capabilities

_None — greenfield project, no existing specs._

## Impact

- New Go module: `go.mod`, `main.go`, packages `config/`, `detect/`, `scanner/`, `semver/`, `command/`, `audit/`, `ui/`.
- New external dependencies: `rivo/tview` (+ `gdamore/tcell`), `pelletier/go-toml/v2`, `Masterminds/semver/v3`, `atotto/clipboard`.
- Runtime dependency on locally installed `npm-check-updates` >= 18 (`ncu` on PATH) and `npm`; `pnpm` required only to audit pnpm projects. Audit queries the npm registry's advisory endpoint (network required).
- New user-owned file: `~/.config/ncu-tui/config.toml`.
- No existing code affected (repository currently contains only README/LICENSE).
