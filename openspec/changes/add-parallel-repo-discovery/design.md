## Context

See proposal.md — Why. Current state that shapes the approach:

- `detect.ScanMode` has two outcomes: no `package.json` → `ModeDeep`, otherwise `ModeSingle`/`ModeDeep` by workspaces. `scanner.ScanPath` maps `ModeDeep` to one `ncu --deep` process over the whole path (scanner/scanner.go:71). ncu 23.1.0 follows symlinks in `--deep`, ignores `node_modules`, and expands link cycles to depth 32 (verified in a scratch folder: 2 repositories, 65 entries).
- `scanner.ExecRunner` already applies `Timeout` per command (scanner/execrunner.go:17). The timeout is "per folder" today only because the folder is one command.
- `orchestrator.Run` fans out one goroutine per source and delivers exactly one `Event` per source; `scanAndAuditPath` then runs one audit goroutine per project with no bound (orchestrator/orchestrator.go:95). Rescans reuse `RunOne`/`RunGlobal` synchronously from the UI (ui/app.go:186).
- `ui.sourceState` holds a single `orchestrator.Event`; 26 call sites read `st.event.Projects`/`Packages`/`Err`, and `selection.projectIdx` and the marks map index into `Event.Projects` (ui/app.go:19-34). Tree rows are rebuilt from state on every refresh (ui/panel.go:17).
- `detect.HasNodeProject` already implements the "walk, skip `node_modules`/hidden, bounded depth 3" rules with `filepath.WalkDir`, which does not follow symlinks (detect/node_target.go). `launch.Comparable` is the existing symlink-resolving comparison helper.
- Every widget mutation goes through `tview.Application.QueueUpdateDraw`; goroutines never touch widgets.

## Goals / Non-Goals

**Goals:**

- One external command never covers more than one repository, so the existing per-command timeout becomes the per-repository timeout with no new timer code.
- Bound total process concurrency in one place that every command path (scan, global, audit, rescan) goes through.
- Keep `detect`, `scanner` and `audit` contracts intact for single and monorepo paths; add, do not rewrite.
- Keep the UI's "one snapshot per source" reading model so the 26 consumers change shape, not logic.

**Non-Goals:**

- Cross-source deduplication (a folder that links to another registered path). Out of scope per proposal.
- Persisting discovery results or caching between launches; detection stays stateless.
- Making `ncu --deep` itself symlink-safe for monorepos: a workspaces monorepo is scanned with `--deep` scoped to the repository as today.
- Changing the bottom-bar `scanning N/M` semantics (sources); folder progress lives on the folder row.

## Decisions

**D1 — Third detection mode `ModeFolder`, discovery lives in `detect`.**
`ScanMode` returns `ModeFolder` when no root `package.json` exists; `ModeDeep` is now reserved for workspaces monorepos. Discovery (`detect.Repos(root) ([]Repo, error)`, `Repo{Dir, Label}`) sits next to `HasNodeProject` and shares its skip rules (`node_modules`, dot-dirs) but not its depth bound: discovery descends without limit, as `ncu --deep` did, so nested groups and monorepos at any depth keep appearing; termination is guaranteed by the visited set (D2) and by stopping at each repository root. Rationale: `detect` is already "stateless filesystem inference per path"; a target accepted by `lazyncu .` validation (depth ≤ 3) is by construction also found by discovery. Alternative — keep `ModeDeep` and branch inside `scanner` on "has package.json" — rejected: it hides a behavioral fork the spec now names, and `ScanMode` tests already enumerate modes.

**D2 — Discovery is a hand-rolled recursive walk, not `filepath.WalkDir`.**
`WalkDir` reports a symlink as a non-directory entry and never follows it. The walker uses `os.ReadDir`, and for `ModeSymlink` entries calls `os.Stat` to learn whether the target is a directory. A `visited` set keyed by `filepath.EvalSymlinks(path)` is checked before entering any directory (including the root), which both deduplicates repositories reachable twice and terminates cycles. Entries are processed in `ReadDir`'s sorted order so output is deterministic; the first path that reaches a real directory wins. `Repo.Dir` is the path *as reached* under the root (a link path stays a link path) so `cd`-style update commands and `npm audit` run where the user sees them; `Repo.Label` is `filepath.Rel(root, Dir)`. Unreadable directories and dangling links (`Stat` error) are skipped, never fatal; only an unreadable root returns an error. Alternative — `filepath.WalkDir` + manual recursion on symlinks — rejected: two traversal mechanisms for one walk, and cycle handling would still need the visited set.

**D3 — Concurrency bound as a `Runner` decorator, acquired before the timeout starts.**
`scanner.LimitedRunner{Inner Runner, slots chan struct{}}` implements `Runner`: acquire a slot (or return `ctx.Err()` if the context ends while waiting), then delegate to `Inner.Run`, release on return. `main.go` builds `LimitedRunner{Inner: ExecRunner{Timeout}, ...}` from `cfg.MaxParallel` and hands it to both `scanner.New` and the auditor, so `ncu`, `npm ls -g` and `npm audit` share one bound with zero changes in `scanner`, `audit` or `orchestrator` fan-out. Because `ExecRunner` creates its timeout context inside `Run`, waiting for a slot happens before the timer starts — the spec's "waiting does not eat the timeout" falls out of the layering. Alternative — semaphore inside `orchestrator` around `ScanPath`/auditor calls — rejected: it would need an `Orchestrator` struct threaded through the UI's rescan paths and would still miss `npm ls -g`; the decorator covers every process by construction. Alternative — bound goroutines instead of processes — rejected: goroutines are cheap, processes are what hit the registry and the CPU.

**D4 — Folder sources emit incremental *snapshot* events; the UI keeps one `Event` per source.**
`orchestrator.Event` gains `Done bool`. For a folder source the orchestrator emits: (1) after discovery, a snapshot with one `ProjectResult` per repository flagged `Pending: true`, sorted by label, `Done: false`; (2) after each repository finishes, a new snapshot (cloned slice, that index replaced by the real result or by `Pending: false, Err: <reason>`); (3) the last replacement carries `Done: true`. Single/deep/global sources emit one event with `Done: true`, exactly as today. `ProjectResult` gains `Pending bool` and `Err error`; a repository failure is a project-level `Err`, never the source's. The UI's `applyEvent` becomes `st.event = ev; st.loading = !ev.Done` — the 26 consumers keep reading `st.event.Projects`, and `projectIdx`/marks stay valid because placeholder positions are fixed at discovery time. Rationale: snapshots keep the immutable-per-event contract the orchestrator package documents and turn a risky UI refactor into shape changes (`Pending`/`Err` on entries). Cost — cloning a slice of ≤ a few hundred entries per event — is negligible next to a process spawn. Alternative — fine-grained delta events (`Discovered`, `ProjectDone`, `SourceDone`) — rejected: forces the UI to own accumulation state and index bookkeeping across goroutine boundaries; the snapshot moves that into the orchestrator goroutine that already owns the source.

**D5 — Per-repository scan and audit are one unit; rescan of an entry reuses it.**
The orchestrator learns about folders through an injected `Discoverer func(dir string) (repos []detect.Repo, isFolder bool, err error)` (production: `detect.ScanMode` + `detect.Repos`; tests: canned lists), mirroring how `Auditor` is injected, so `orchestrator` tests keep running without a filesystem. `orchestrator.ScanProject(ctx, sc, auditor, repo) []ProjectResult` runs `sc.ScanPath(repo.Dir)` (so a repository that is a monorepo yields `ncu --deep` scoped to it) followed by the audit of each resulting project, and is what the folder fan-out calls per repository. A repository detected as `deep` may yield several projects; they are spliced in place of the single placeholder (label-prefixed `repo/workspace`), so folder entries remain a flat list under the source. UI rescan on a project entry (`projectIdx >= 0` inside a folder source) marks that entry `Pending`, calls `ScanProject` in a goroutine and splices the result via `QueueUpdateDraw`; rescan on the folder row calls `RunOne` (rediscover + full snapshot stream). Overlap guard: a pending entry refuses rescan with the existing "still scanning" message. Alternative — retry only failed entries, not successful ones — rejected: one rule ("r rescans what is selected") is easier to explain and matches the existing keymap description.

**D6 — Folder row wording is derived, not stored.**
`sourceText` derives `scanning done/total` from counting non-pending entries while `!Done`, `N failed` from entries with `Err`, and `no projects found` from `Done && len(Projects) == 0`. Aggregates (`aggregateSource`) skip pending and failed entries. No new state fields beyond `Done`/`Pending`/`Err`. `anyLoading` stays `!Done` per source, so the spinner and bottom progress bar are untouched.

**D7 — `max_parallel` mirrors `timeout_ms` in `config`.**
`Config.MaxParallel int \`toml:"max_parallel,omitempty"\``, `DefaultMaxParallel = 4`, defaulted in `Load` when `<= 0`, written on first launch next to `timeout_ms`. Alternative — derive from `runtime.NumCPU()` — rejected: the bottleneck is registry I/O and rate limiting, not CPU; a fixed, documented default is predictable.

## Risks / Trade-offs

- [Unbounded discovery over a folder with large non-Node subtrees (build outputs, vendored sources) walks many directories before finding nothing] → the walk stops at every repository root, skips `node_modules` and dot-dirs, and does only `ReadDir`/`Stat` (no file reads) — the same cost `ncu --deep`'s glob already paid; a pathological folder is a registration mistake visible in the tree, not a hang, because discovery runs in the source goroutine and the UI stays responsive.
- [A symlink whose target is inside another registered source gets scanned under both] → accepted, out of scope; label makes the origin visible.
- [Placeholder positions are fixed at discovery, but a monorepo repository expands into several projects when its scan lands, shifting later indices that `projectIdx` and marks point at] → the splice happens inside the orchestrator goroutine before the snapshot is emitted, so the UI only ever sees consistent snapshots; when a snapshot changes the entry count, the UI clears that source's marks (same rule as rescan) and re-anchors the selection by label. Marks placed while a folder is still scanning are rare enough that this is acceptable.
- [50 snapshot events in quick succession flood `QueueUpdateDraw`] → each event is one redraw at ≤ 50 events per folder; tview coalesces draws; no batching needed at this scale.
- [Dedupe by `EvalSymlinks` fails on permission-restricted paths] → fall back to the cleaned path as `launch.Comparable` does; worst case a duplicate entry, same as today.
- [Behavioral change for users whose "folder" was actually relying on `ncu --deep` for root-level `package.json`-less monorepos (e.g. pnpm workspaces without root manifest)] → `pnpm-workspace.yaml` without `package.json` is not a valid pnpm workspace; no realistic loss.
- [Default 4 too low for fast networks] → configurable; README shows the knob.

## Migration Plan

Purely additive config (`max_parallel` optional); existing `config.toml` files load unchanged. One release, minor bump. Rollback is the previous binary; config written by the new version still parses on the old one because `max_parallel` is an unknown key that go-toml ignores.

## Open Questions

None. Wording of the folder row segments and the exact error text per repository can be tuned during implementation without touching specs or tasks.
