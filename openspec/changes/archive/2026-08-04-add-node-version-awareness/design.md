# Design: add-node-version-awareness

## Context

See proposal.md — Why. Relevant current state:

- `scanner.scanSingle` runs `ncu --jsonUpgraded --packageFile <pkg>`; `scanner.scanDeep` runs `ncu --deep --jsonUpgraded`; `scanner.ScanGlobal` runs `ncu -g --jsonUpgraded` (`scanner/scanner.go`). All go through the injected `Runner`, so unit tests assert exact argv.
- `scanner.Project` (`scanner/types.go`) is embedded in `orchestrator.ProjectResult`, so new fields flow to the UI with no orchestrator changes.
- `detect` is the stateless per-path filesystem-inspection package (scan mode, package manager) — the natural home for reading `.nvmrc`/`engines.node`.
- `ui.detailTitle(showVulns, mode, filter, marked, total)` (`ui/detail.go`) is the single composer of the detail panel title; `refreshDetail` is its only call site and can resolve the selected project via `a.selectedProject()`.
- ncu >= 18 is guaranteed by preflight; `--enginesNode` exists in that range ("Include only packages that satisfy engines.node as specified in the package file").

## Goals / Non-Goals

**Goals:**
- Suggestions a project can actually install, automatically, with zero configuration.
- Node context visible where the user already looks (detail title), following the project's TUI conventions: titles that teach, no color-only encoding, graceful omission over empty placeholders.

**Non-Goals:**
- Running ncu under a project-specific node runtime (nvm/fnm/asdf/volta) — rejected in exploration.
- Per-package incompatibility markers in the table.
- Showing node context in the tree panel — project rows are already the densest line in the UI (UX-REVIEW O-18).

## Decisions

**D1 — Pass `--enginesNode` unconditionally on project scans (single and deep).**
ncu resolves the constraint from each package file itself; a manifest without `engines.node` makes the flag a no-op. Passing it always avoids a pre-read of the manifest just to decide the argv, and deep mode allows no per-project conditioning anyway (one invocation covers many manifests). Observable behavior matches the "automatic when declared" decision from exploration.
Alternative considered: conditional flag for single mode — more code, identical behavior.
Verification: task includes a manual smoke test against a fixture with a constraining `engines.node` (behavior of the flag is ncu's, not unit-testable through the injected Runner).

**D2 — Node context reading lives in `detect`.**
`detect.NodeContext(dir) (nvmrc, engines string)`: trimmed `.nvmrc` content; `engines.node` from `package.json`. Both degrade to empty on any read/parse error, matching the package's existing tolerance (`manifestVersions` never fails a scan). `scanner.buildProject` fills two new `Project` fields (`Nvmrc`, `EnginesNode`); `ProjectResult` inherits them via embedding.
Alternative considered: reading inside `scanner` — spreads filesystem inspection across packages; `detect`'s charter is exactly this.

**D3 — Display as a low-priority title segment, `.nvmrc` first.**
`detailTitle` gains a `nodeCtx` parameter appended after the existing segments: `· node 18.19.0 (.nvmrc)` or `· node >=18 (engines)`. `.nvmrc` wins for display when both exist (the explicit pin is what a developer switches to); filtering always uses `engines.node` regardless (that is what ncu reads). `.nvmrc` content is shown verbatim (handles `v18`, `lts/gallium`, `18.19.0`). Shown in both packages and vulnerabilities views — it is selection context, not view state.
Alternative considered: tree row — rejected (density, Non-Goals); dedicated info row in the table — shifts row indexes that marking (`rowPkgs`) depends on.

## Risks / Trade-offs

- [ncu `--deep` + `--enginesNode` interaction unverified against a real tree] → Manual smoke test in tasks before considering done; if ncu applies a root constraint globally, fall back to conditional flag on single mode only and drop the deep clause from the spec (artifact update).
- [Fewer suggestions may surprise users who expected latest] → The filtered suggestion is the installable one; README already frames lazyncu around actionable commands. No UI noise added.
- [Long `.nvmrc`/engines strings could crowd the title] → Values are short in practice (`18`, `>=18 <21`, `lts/iron`); segment is last, so tview truncates it first on narrow terminals.
- [`engines.node` may use full semver ranges ncu can't satisfy exactly] → ncu owns that resolution; lazyncu only passes the flag.

## Migration Plan

Pure additive behavior; no config, no data migration. Rollback = revert commit.

## Open Questions

None.
