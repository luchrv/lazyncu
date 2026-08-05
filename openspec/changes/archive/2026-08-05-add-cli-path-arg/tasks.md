## 1. Path resolution and containment helpers

- [x] 1.1 Add a resolve helper (tilde expansion + `filepath.Abs` + `filepath.Clean` + `filepath.EvalSymlinks` with cleaned-path fallback) with table-driven tests covering `.`, `..`, `~/x`, absolute, and symlinked inputs
- [x] 1.2 Add a segment-wise containment helper (`equal` / `contains` on resolved paths) with tests including the `/projects-old` vs `/projects` raw-prefix trap and symlinked equality

## 2. Node-target validation

- [x] 2.1 Add `detect.HasNodeProject(dir)` — bounded `WalkDir` (depth ≤ 3 constant), skipping `node_modules` and hidden dirs — with tests: root `package.json`, nested project, monorepo layout, empty dir, depth-exceeded miss

## 3. Launch classification

- [x] 3.1 Implement launch-target classification against `cfg.Paths` producing one of equal/contained/parent/new (with covered children listed for parent), with table-driven tests over all four outcomes plus multiple-children parent case
- [x] 3.2 Wire `main.go`: keep `--version` precedence, accept 0–1 positional args, usage error + non-zero exit on 2+, pre-TUI validation errors on stderr with non-zero exit; tests for arg parsing and error paths

## 4. UI startup behavior

- [x] 4.1 Extend `ui.New`/`App` with a launch intent: initial source selection for equal/contained/new, and pending-selection state (`source`, target dir)
- [x] 4.2 Resolve pending selection in `applyEvent`: on the covering source's event, select the project entry matching the target (resolved-path comparison against `Project.Dir`), clear pending; degrade to source selection on scan error or no match — with tests for match, no-match, and scan-error cases
- [x] 4.3 New-path intent: persist via `AddPath` + `Save` before TUI start (or reuse the existing addPath flow at startup), select the new source, ensure its scan is part of the launch fan-out; tests for persistence and selection
- [x] 4.4 Parent consolidation modal: after persisting the parent, show confirm listing covered children; accept removes them (`RemovePath` loop + one `Save`), decline keeps them; cursor on parent either way; guard ordering vs first-run welcome browser; tests for accept/decline paths

## 5. Verification and docs

- [x] 5.1 Run `go vet ./...`, `go build ./...`, `go test -race ./...` and confirm coverage of new code ≥ 80%
- [x] 5.2 Update README usage section with `lazyncu [path]` (`lazyncu .`) and the containment/consolidation behavior
