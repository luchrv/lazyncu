## Why

A registered folder that holds many repositories (the reporting case has 50+, several of them symlinks to repositories elsewhere) is scanned by a single `ncu --deep` process bounded by the 30 s `timeout_ms`. That one process has to resolve every dependency of every repository against the registry, blows through the timeout, gets killed, and the whole source fails with `ncu --deep failed for <dir>: signal: killed` — zero results for 50 repositories because one of them was slow. On top of that, `ncu --deep` follows symlinks blindly: a link pointing back into the folder expands `loop/loop/loop/...` up to 32 levels deep (verified with ncu 23.1.0), multiplying the work and guaranteeing the timeout; a repository reachable both by its real path and by a link is scanned twice.

## What Changes

- A registered path with no root `package.json` is now a **folder of repositories**: lazyncu discovers the repositories itself (walking the folder, stopping at the first directory that holds a `package.json`, following symlinks, skipping `node_modules` and hidden directories, deduplicating by resolved real path and never entering a directory twice) and scans each repository as its own unit through the existing single/monorepo logic. `ncu --deep` is no longer run over the whole folder; it remains the strategy for a repository that is itself a workspaces monorepo, scoped to that repository.
- The scan timeout applies **per external command** (one repository, one `ncu` run) instead of covering the whole folder, so a slow or broken repository fails alone.
- A new optional config setting `max_parallel` (default 4) bounds how many external processes (`ncu`, `npm ls`, `npm audit`) run at the same time across all sources. Waiting for a slot does not count against the command timeout.
- Folder sources report progress incrementally: the sources panel lists the discovered repositories immediately with a per-entry loading state, fills each entry in as its scan completes, shows `done/total` on the folder row while scanning, and marks an individual failed repository with an error badge and its reason, without failing the folder.
- A failed repository entry can be retried on its own with the existing rescan key; rescanning the folder row rediscovers and rescans everything.
- **Out of scope**: deduplicating a repository that is reachable from two *different* registered sources (e.g. a folder containing a symlink to another registered path); today's behavior (scanned under both) stays.

## Capabilities

### New Capabilities

_None — the change reshapes how existing capabilities behave._

### Modified Capabilities

- `scan-detection`: the decision tree gains a third mode. "No `package.json`" now means **folder** (discover repositories), "workspaces or `pnpm-workspace.yaml`" stays **deep** (monorepo), plain `package.json` stays **single**. Adds the repository-discovery rules (stop at first manifest, follow symlinks, dedupe by real path, cycle safety, skip `node_modules`/hidden, unbounded depth like `ncu --deep`).
- `package-scanning`: "Deep paths are scanned recursively" is replaced by folder scanning (one scan per discovered repository, each through single/deep detection); "Sources are scanned in parallel on launch" gains the global concurrency bound and per-repository result delivery; "Scan failures are isolated per source" gains per-repository isolation inside a folder; timeout semantics become per command.
- `config-store`: new optional `max_parallel` setting with default and validation.
- `dashboard-ui`: "Loading and error states are shown per source" extends to per-repository entries inside a folder source (placeholders, `done/total` on the folder row, per-entry error badge and reason, empty-folder message); "User can rescan the selected source" gains rescanning a single failed repository entry.

## Impact

- `detect`: new `ModeFolder`, repository discovery walker (shares the skip rules with `HasNodeProject`; unbounded depth, cycle-safe through the visited set).
- `scanner`: `ScanPath` routes `ModeFolder` to per-repository scanning; a concurrency-limiting `Runner` decorator (semaphore) wraps `ExecRunner`; per-repository timeout falls out of the existing per-command timeout.
- `orchestrator`: per-source event stream becomes incremental for folder sources (discovered placeholders, per-repository results, per-repository errors, completion); single-project rescan entry point.
- `config`: `max_parallel` field, default constant, load-time defaulting.
- `ui`: source state consumes incremental events; tree rows for pending/failed repositories; folder-row progress; per-entry retry; `sourceState.event.Projects` consumers adapt to the new shape; pending launch selection resolves on the matching repository's event.
- `main.go`: wires the limited runner with `cfg.MaxParallel`.
- README Configuration section documents `max_parallel` and the folder behavior; `docs/` unaffected.
- No new dependencies. Config file stays backward compatible (`max_parallel` optional). Discovery walks as deep as `ncu --deep` did (no depth bound; `node_modules`/hidden skipped, cycles cut by real-path tracking), so nested groups of repositories and monorepos at any depth keep appearing.
