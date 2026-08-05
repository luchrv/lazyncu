## Context

See proposal.md — Why. Current state that shapes the approach:

- `main.go` handles only `--version` as the first argument (`wantsVersion`, main.go:23); everything else falls through to `run()`, which loads config and enters the TUI unconditionally.
- `config.Config` is immutable-by-convention: `AddPath` (config/config.go:95) validates existence and exact duplicates only; `RemovePath` is a pure filter. Persisted paths are cleaned but not symlink-resolved.
- Scanning fans out one goroutine per source (`orchestrator.Run`), delivering exactly one async `Event` per source; the UI applies them in `applyEvent` (ui/app.go:193) on the UI thread.
- Deep mode (`detect.ScanMode` → `ncu --deep`) means a registered parent already surfaces nested projects as child entries, so "target contained in a registered path" is the common case, not an edge case.
- `detect.ScanMode` cannot be reused as a launch validator: a missing `package.json` maps to `ModeDeep` ("folder of projects"), so it never says "not a Node target".
- The UI already has a confirm modal component (`ui/confirm.go`) and an add-path flow (`ui/input.go:183`) that validates via `Config.AddPath`, persists, and triggers `scanOne`.

## Goals / Non-Goals

**Goals:**

- Resolve, validate, and classify the launch target fully before the TUI starts; the UI only receives an already-classified launch intent.
- Reuse the existing immutable config operations and the existing confirm modal; no new UI primitives.
- Keep persisted config bytes untouched by comparisons (symlink resolution happens in memory only).

**Non-Goals:**

- Multiple positional paths (`lazyncu a b c`) — explicit usage error.
- Headless/no-TUI mode (`lazyncu . --report`-style output).
- Any change to `AddPath`/`RemovePath` semantics for the in-TUI add-path browser flow.
- Config migration or normalization of already-registered paths.

## Decisions

**D1 — Classify the target in `main`/launch code, pass a launch intent to the UI.**
A small launch step (new file, e.g. `launch.go` or a `launch` package) resolves the argument and classifies it against `cfg.Paths` into one of: `equal(source)`, `contained(source, target)`, `parent(covered...)`, `new`. `ui.New` receives this intent instead of re-deriving it. Rationale: validation errors must exit before the TUI (spec requirement), and the UI stays a consumer of decisions, matching the existing "config validates, UI reacts" split. Alternative — teach `Config.AddPath` containment — rejected: it would change the in-TUI add-path flow's behavior, which is out of scope.

**D2 — Containment compares symlink-resolved paths, segment-wise.**
Both sides are passed through `filepath.EvalSymlinks` (falling back to the cleaned path if resolution fails, e.g. permission errors) and compared by path segments — `target == reg` or `strings.HasPrefix(target, reg + separator)` on the resolved, cleaned forms. Never raw prefix (`/projects-old` vs `/projects`). Persisted values are never rewritten with resolved forms. Alternative — `filepath.Rel` and checking for `..` — equivalent; segment prefix is simpler to test.

**D3 — Node-target validation is a bounded `WalkDir`.**
Valid when `package.json` exists at the root, or in any subdirectory within depth ≤ 3, skipping `node_modules` and hidden directories. Depth 3 covers `folder/project/package.json` and typical monorepo `packages/*/package.json` layouts while bounding cost on huge trees. Lives beside `detect` (same package, new function, e.g. `detect.HasNodeProject(dir) bool`) since it shares the "read the filesystem, stateless" contract. Alternative — accept any existing directory (current `AddPath` behavior) — rejected by explored decision Q3: fail fast beats an empty/erroring source in the TUI.

**D4 — Cursor targeting via a pending-selection field resolved in `applyEvent`.**
`App` gains a pending target (`pendingSelect struct{source, dir string}`). At startup the covering (or new) source is selected immediately. When `applyEvent` receives the event for `pendingSelect.source`, it looks for a project whose `Dir` equals the target (compared on resolved forms), selects that node if found, and clears the pending state either way. Degradation: scan error or no match leaves the cursor on the source — no error surfaced. Rationale: events are the single UI-thread choke point already; no new synchronization. Alternative — block startup until the scan finishes — rejected: freezes the TUI for the slowest-scan duration and fights the existing async design.

**D5 — Parent consolidation is a startup confirm modal after persisting the parent.**
Order: `AddPath(parent)` + `Save` first, then show the confirm modal listing covered children ("Remove N paths now covered by <parent>?"). Accept → `RemovePath` per child on the current config + one `Save`. Decline → nothing. The parent's scan starts regardless. Rationale: the user's primary intent (register the parent) must not be hostage to the secondary question; a crash between the two saves leaves a redundant-but-valid config. Reuses `ui/confirm.go`.

**D6 — Argument parsing stays hand-rolled.**
`--version` keeps first-argument precedence; then at most one non-flag argument is accepted; more → usage error on stderr, exit non-zero. No `flag` package adoption: two cases don't justify changing the existing CLI surface (and `flag` would alter `-version` handling).

## Risks / Trade-offs

- [`EvalSymlinks` fails on restricted parents (permissions, dangling links)] → fall back to comparing cleaned unresolved paths; worst case a symlinked duplicate registers twice, same as today's behavior.
- [Bounded depth misses deeply nested monorepos (e.g. `apps/group/sub/project`)] → depth constant in one place; error message tells the user no Node project was found at the path, so the mismatch is visible, and the in-TUI add-path flow (existence-only) remains an escape hatch.
- [Pending selection races a fast scan] → events are applied on the UI thread via `QueueUpdateDraw` after `App` construction; pending state is set before `Run` starts consuming, so no event can be missed.
- [Consolidation modal at startup competes with the first-run welcome browser] → mutually exclusive in practice (first run has zero registered paths, so parent-of-registered cannot occur); guard the order explicitly anyway: consolidation wins over welcome.
- [`ncu --deep` may report project dirs in a form that differs from the resolved target (relative vs absolute, symlinks)] → compare `Project.Dir` and target through the same resolve-then-clean helper used for containment.

## Migration Plan

Purely additive CLI surface; no config format change, no rollback concerns. Ship in one release; README gains the `lazyncu .` usage line.

## Open Questions

None — depth constant (3) and modal wording can be tuned during implementation without affecting specs or tasks.
