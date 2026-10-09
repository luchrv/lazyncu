# Tasks

## 1. Registry client (`registry`)

- [x] 1.1 Add a `registry` package with an `npm view` transport that retrieves versions, per-version `engines`, dist-tags, `deprecated`, and `time` for a single package; verify with fixture tests parsing both npm-12 array output and a single-object response.
- [x] 1.2 Implement target selection (`dist-tags.latest` with greatest fallback) and the filters (prerelease unless current is prerelease, deprecated included by default, `engines.node` compatibility); verify with table-driven unit tests covering the `registry-query` scenarios.
- [x] 1.3 Implement non-registry spec skipping (`file:`/`link:`/`workspace:`/`catalog:`/`portal:`/`owner/repo`) and per-dependency fetch-failure isolation; verify with tests that a failing dependency is omitted and the rest succeed.

## 2. Scanner integration

- [x] 2.1 Introduce a `Registry` interface in `scanner` and inject it, keeping `Runner` for `npm ls`/`npm audit`; verify scanner tests compile and pass with a fake registry.
- [x] 2.2 Rewrite `ScanGlobal` to use `npm ls -g` plus registry lookups; verify the global scenarios, including installed-version lookup failure.
- [x] 2.3 Rewrite `scanSingle` to read manifest dependencies including the `packageManager` field and resolve them through the registry; verify the single-project scenarios including the engines-incompatible fallback.
- [x] 2.4 Remove the `ncu` availability/version preflight from `main.go` and `scanner`; verify `lazyncu` launches with no `ncu` on PATH and `--version` still prints.

## 3. Deep discovery

- [x] 3.1 Add the recursive `**/package.json` walk in `detect` (include root and nested manifests, skip `node_modules`/`.pnpm-store`, deterministic order, per-manifest label); verify against the deep-manifest discovery scenarios.
- [x] 3.2 Rewire the deep scan to walk manifests and resolve each through the registry, applying `engines.node` per manifest; verify the deep scanning scenarios.

## 4. `.ncurc` configuration (`ncurc`)

- [x] 4.1 Add a `ncurc` reader for `.ncurc`, `.ncurc.json`, `.ncurc.yaml`, `.ncurc.yml` (nearest file; per manifest directory in deep mode); verify parse and precedence tests.
- [x] 4.2 Apply the supported keys (`filter`, `reject`, `filterVersion`, `rejectVersion`, `pre`, `deprecated`, dist-tag `target`) and warn on out-of-scope keys or executable configs; verify a rejection list is applied and a warning is emitted for `peer: true` and for `.ncurc.js`.
- [x] 4.3 Apply the `min-release-age` cooldown using npm config and the `time` field; verify versions newer than the configured age are excluded.

## 5. Update commands (`command`)

- [x] 5.1 Rewrite the project update command to `cd <dir> && <add> <pkg>@<ver> …` (`npm install`/`pnpm add`/`yarn add`); verify the per-package-manager scenarios.
- [x] 5.2 Rewrite the selection-filtered command to the same form restricted to marked packages; verify the filtered-project and clearing-selection scenarios.
- [x] 5.3 Update the read-only guarantee tests to the new command set (`npm view`, `npm ls`, `npm audit`); verify no non-read-only process is spawned.

## 6. Distribution and docs

- [x] 6.1 Remove `depends_on: npm-check-updates` from the `.goreleaser.yaml` cask and resolve the design Open Question about `depends_on: node`; verify the rendered cask no longer declares the npm-check-updates dependency.
- [x] 6.2 Update `README.md` Requirements/Install and the suggested-commands table (no npm-check-updates; npm/pnpm); verify the documented commands match the spec.
- [x] 6.3 Update `docs/` and website references that instruct installing `ncu`; verify with a search that no stale npm-check-updates install instructions remain.

## 7. Integration verification

- [x] 7.1 Run `make check` (gofmt, vet, race tests, coverage) and verify it passes.
- [x] 7.2 Mandatory before release: scan a project whose dependency resolves from a scoped private registry and verify the HTTP client resolves it with the credentials from npmrc (no `npm view` fallback in the log) (manual).
- [x] 7.3 End-to-end: with no `ncu` installed, scan a fixture single project, monorepo, folder, and global source; verify entries and suggested commands match the specced behavior.

## 8. Scan performance

- [x] 8.1 Drop `time` from the ranged `npm view` query and fetch publish times only under a cooldown; memoize registry metadata per package across repositories (`registry.Cache`); verify with the registry and cache tests.
- [x] 8.2 Defer `npm audit`/`pnpm audit` to selection time (`audit.StatusPending`/`StatusRunning`, `orchestrator.AuditProject`, UI `ensureAudit`); verify a scan spawns no audit and selecting a project audits it once.

## 9. Native registry client and cache

- [x] 9.1 Add `registry.Resolver` (one `npm config list --json` per process, npmrc credential parsing with `${VAR}` expansion, project `.npmrc` overlay, nerf-dart matching) and `registry.HTTPFetcher` (abbreviated packument, bounded by `max_requests`, `npm view` fallback); verify with resolver and httptest-based fetcher tests.
- [x] 9.2 Persist `registry.Cache` to `registry-cache.json` with `cache_ttl`, bypassed by rescans through `registry.WithRefresh`; verify persistence, expiry, refresh and malformed-file tests.
- [x] 9.3 Source the cooldown (`min-release-age`) from the resolved npm config instead of one `npm config get` per project; add `max_requests` and `cache_ttl` to `config`; update README and the config-store, registry-query, package-scanning and update-commands deltas.

## 10. Deferred audit phase and audit cache

- [x] 10.1 Extract `memo.Store` from `registry.Cache` (singleflight, TTL, JSON persistence, `memo.WithRefresh`); verify with the store tests and the unchanged registry cache tests.
- [x] 10.2 Audit from the lockfile (`npm audit --json --package-lock-only`) and cache results in `audit-cache.json` keyed by `audit.Fingerprint`; verify fingerprint, reuse, invalidation and never-stored-failure tests.
- [x] 10.3 Queue pending audits in the UI when a scan stream closes (bounded by `max_parallel`, selection jumps the queue); verify the queue, priority and stale-result tests. Update the vulnerability-audit and config-store deltas and the README.

## Workflow follow-up

- Archive the change after the project's review requirements are satisfied.
- Verify the archived result.
