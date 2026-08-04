# Tasks: add-node-version-awareness

## 1. detect: node context reader

- [x] 1.1 Write table-driven tests for `detect.NodeContext`: `.nvmrc` present (trimmed, `v`-prefixed, `lts/*`), `engines.node` present, both, neither, malformed `package.json`
- [x] 1.2 Implement `detect.NodeContext(dir)` returning `(nvmrc, engines string)`, degrading to empty on any error

## 2. scanner: engines filtering and node fields

- [x] 2.1 Extend scanner tests: single and deep scan argv now include `--enginesNode`; `Project` carries `Nvmrc`/`EnginesNode` from fixtures; global scan argv unchanged
- [x] 2.2 Add `Nvmrc` and `EnginesNode` fields to `scanner.Project`; fill them in `buildProject` via `detect.NodeContext`
- [x] 2.3 Append `--enginesNode` to the `scanSingle` and `scanDeep` ncu invocations

## 3. ui: node context in detail title

- [x] 3.1 Extend `detailTitle` tests: node segment rendering (`.nvmrc` precedence, engines fallback, omitted when empty), position after existing segments
- [x] 3.2 Add node-context parameter to `detailTitle` and compose it in `refreshDetail` from the selected project (empty for global source)

## 4. Verification

- [x] 4.1 Run the gate: `go vet ./...`, `go build ./...`, `go test ./...`
- [x] 4.2 Manual smoke test: fixture project with `engines.node` constraining a dependency below latest — confirm filtered suggestion in single mode and in a deep tree (mixed manifests with and without engines); confirm title segment renders
- [x] 4.3 Update README if behavior description mentions suggestion semantics (check Features section wording)
