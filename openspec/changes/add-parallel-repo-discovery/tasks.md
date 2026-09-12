## 1. Detection and discovery

- [x] 1.1 Add `detect.ModeFolder` and make `ScanMode` return it when no root `package.json` exists (workspaces stay `ModeDeep`); update the `TestScanMode` table so the "no package.json" case expects `folder` and add a case for a folder containing a workspaces repository detected independently — verify with `go test ./detect/`
- [x] 1.2 Implement `detect.Repos(root) ([]Repo, error)` as a recursive `os.ReadDir` walk that follows directory symlinks via `os.Stat`, dedupes/cycle-guards with a visited set keyed by `filepath.EvalSymlinks` (cleaned-path fallback), stops at the first `package.json` per branch, skips `node_modules` and dot-dirs, descends without a depth limit, sorts deterministically, and labels by path as reached — verify with table-driven tests built in `t.TempDir()` covering: flat folder, nested group, stop-at-repo-root, symlink to external repo (label = link name), self/ancestor link cycle, same repo via link and real path (once), dangling link, unreadable dir (skipped), unreadable root (error), `node_modules`/hidden ignored, repository nested six levels deep is found

## 2. Concurrency bound

- [x] 2.1 Add `scanner.LimitedRunner{Inner Runner}` with `NewLimitedRunner(inner, max)` (buffered-channel semaphore) that acquires before delegating, releases on return, and returns `ctx.Err()` when the context ends while waiting — verify with tests using a blocking fake inner runner: never more than `max` concurrent `Run` calls, waiters proceed as slots free, cancellation while waiting returns promptly, `max <= 1` still works
- [x] 2.2 Verify by test that wrapping `ExecRunner` keeps the timeout per command and that time spent waiting for a slot is not deducted (fake clock or a slot held longer than the timeout with a short inner command still succeeding)

## 3. Config

- [x] 3.1 Add `Config.MaxParallel` (`max_parallel`, omitempty), `DefaultMaxParallel = 4`, default it in `Load` when `<= 0`, and write it on first launch next to `timeout_ms` — verify with tests mirroring the timeout ones: default when absent, override honored, `0`/negative fall back, first-launch file contains `max_parallel = 4`, existing files without the key still load
- [x] 3.2 Wire `main.go`: build `ExecRunner{Timeout}` wrapped in `LimitedRunner(cfg.MaxParallel)` and pass the wrapped runner to both `scanner.New` and the auditor closure — verify with `go build ./...` and the existing `main_test.go` still passing

## 4. Scanner and orchestrator

- [x] 4.1 Route `scanner.ScanPath` so `ModeFolder` returns an explicit error if called directly (folders are expanded by the orchestrator) while `ModeSingle`/`ModeDeep` behave as today; add `detect.Repo` handling so a repository scanned as `deep` labels its projects relative to that repository — verify with existing `scanner` tests plus one asserting the folder guard
- [x] 4.2 Extend `orchestrator.Event` with `Done bool` and `ProjectResult` with `Pending bool` and `Err error`; make every existing single-shot path (`RunGlobal`, single/deep `RunOne`) emit `Done: true` — verify existing orchestrator tests pass with the new fields
- [x] 4.3 Add the injected `Discoverer` and implement folder scanning in `Run`/`RunOne`: discovery → snapshot of pending placeholders sorted by label → per-repository `ScanProject` goroutines → cloned snapshot per completion (result, or `Err` on failure, deep repositories spliced in place) → final snapshot `Done: true`; discovery error → source-level `Err` with `Done: true` — verify with tests using gated fakes: placeholder snapshot precedes any result, each completion yields a new snapshot with only that index changed, one failing repository never sets source `Err`, empty discovery yields `Done` with zero projects, discovery error surfaces as source `Err`, snapshots are never mutated after emission (race detector clean)
- [x] 4.4 Add `orchestrator.ScanProject(ctx, sc, auditor, repo) []ProjectResult` (scan + audits for one repository) used by the folder fan-out and by the UI's single-entry rescan — verify with a test covering success, scan failure → `Err`, and a deep repository yielding several results

## 5. UI

- [x] 5.1 Update `applyEvent` to `st.event = ev; st.loading = !ev.Done`, clear marks and re-anchor the selection by label when a snapshot changes the entry count, and resolve the pending launch selection against non-pending entries only — verify with `ui` tests for progressive snapshots, selection stability, mark clearing on splice, and pending selection resolving when the matching repository lands
- [x] 5.2 Render folder rows and entries: spinner + `scanning done/total` while `!Done`, `N failed` next to the aggregate, `no projects found` for an empty completed folder; project entries render a spinner while `Pending` and the error badge when `Err != nil`; `aggregateSource` skips pending and failed entries — verify with `panel_test.go` cases for each wording and aggregate exclusion
- [x] 5.3 Detail panel and command bar: a pending entry shows the loading message, a failed entry shows `renderScanError` with its own reason and retry hint, and commands are suppressed for pending/failed entries — verify with `detail_test.go` cases
- [x] 5.4 Rescan: `rescanSelected` on a project entry inside a folder marks it pending, runs `ScanProject` in a goroutine and splices the result through `QueueUpdateDraw`; the folder row still rescans via `RunOne`; overlap guard rejects rescanning a pending entry; mark-loss confirmation counts only that entry's marks — verify with tests for entry rescan, folder rescan, blocked-while-pending, and confirmation wording
- [x] 5.5 Confirm the spinner goroutine and bottom `scanning N/M` need no changes with folder sources (loading is `!Done` per source) — verify with the existing spinner tests plus one folder-source case

## 6. Verification and docs

- [x] 6.1 Run `make check` (`gofmt`, `go vet ./...`, `go test -race ./...`, coverage) and confirm new code coverage ≥ 80%
- [x] 6.2 Manual smoke in a real terminal: register a folder with 50+ repositories including symlinks to external repositories and a link cycle; confirm entries appear progressively, the folder row counts `done/total`, no `signal: killed`, a deliberately broken repository shows its own error and `r` retries only it
- [x] 6.3 Update README: Configuration snippet gains `max_parallel = 4` with a comment, the `folder of projects → ncu --deep` comment becomes "folder of repositories → one scan per repository", and a short note on symlinks/dedupe (followed, cycle-safe, each repository once); regenerate nothing else (demos unaffected)
