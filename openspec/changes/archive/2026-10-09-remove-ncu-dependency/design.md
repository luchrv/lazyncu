# Design

## Context

See `proposal.md` for motivation. Current state that shapes the approach:

- `scanner/scanner.go` invokes `ncu` through an injected `Runner` interface
  (`scanner/types.go`, `scanner/execrunner.go`) and parses `--jsonUpgraded`
  output. `npm ls -g` and `npm audit` already go through the same `Runner`.
- `detect` already distinguishes `single`/`deep`/`folder`, discovers repositories
  in a folder (`detect/repos.go`), and reads `engines.node`/`.nvmrc`.
- `command` builds `ncu -u`-based strings; the specs for `package-scanning`,
  `scan-detection`, `update-commands`, and `release-pipeline` name `ncu`.
- `lazyncu` already requires `npm` on PATH.

Verified against the sibling `../npm-check-updates` (v23.1.0) and npm 12.2.0:
each registry fact `lazyncu` needs (versions, per-version `engines`, dist-tags,
`deprecated`, `time`) is obtainable from one `npm view` call per package, and
npm resolves registry/auth from the same configuration `ncu` uses.

## Goals / Non-Goals

**Goals:**

- Remove every runtime dependency on `ncu` (scan, preflight, suggested command,
  Homebrew) while keeping scan results equivalent for the options `lazyncu`
  actually uses.
- Preserve private/scoped registry support by delegating registry and auth
  resolution to npm.

**Non-Goals:**

- Reproducing `ncu` features `lazyncu` never invokes: `--peer`, `--minimal`,
  target levels (`-t minor|patch`), `--doctor`, interactive mode, filters beyond
  `.ncurc`'s `filter`/`reject`, catalogs, `--workspaces`/`-w`, and writing
  `package.json`.
- Implementing npm config parsing or a native registry client.
- Changing the read-only guarantee.

## Decisions

### D1 — Registry transport is a native HTTP client with `npm view` fallback (revised)

**Revision.** The first version of this design chose `npm view` as the only
transport (option A) and rejected a native client (B) and a hybrid (C) for
parity risk. Measurements on a 52-repository folder reversed that call: with
results memoized per package, one `npm view` process per unique dependency
(294 processes) still cost 91 s of CPU — npm's own startup dominates, not the
network — and a 4-slot bound serialized it into ~55 s. The hybrid is now the
primary transport, with option A kept as the fallback that bounds parity risk.

`registry.HTTPFetcher` requests the abbreviated packument
(`GET <registry>/<name>`, `Accept: application/vnd.npm.install-v1+json`: versions
with `engines` and `deprecated`, plus `dist-tags`; 3–5× smaller than the full
document) over a keep-alive pool bounded by `max_requests` (default 32). The
full packument is requested only when a release cooldown needs `time`.
Registry selection and credentials come from npm's configuration
(`registry.Resolver`): one `npm config list --json` per process supplies the
global/user/environment layers (`registry`, `@scope:registry`, `strict-ssl`,
`cafile`, `proxy`, `https-proxy`, `min-release-age`, and the `userconfig`/
`globalconfig` paths); credentials — which npm never prints — are read from
those npmrc files, the project's own `.npmrc` (overlaid locally, so no npm
process runs per project) and `npm_config_//…` variables, matched by
nerf-dart prefix (`_authToken` → Bearer, `_auth` or `username`/`_password` →
Basic). Any request failure (resolution error, network, non-2xx, unparseable
body) falls back to `registry.NpmViewFetcher`, which runs
`npm view '<pkg>@>=0.0.0-0' version engines dist-tags deprecated --json` in
the project directory (and `npm view '<pkg>' time --json` for cooldowns).
`time` MUST NOT be requested on the ranged `npm view` query: npm repeats the
package-level map on every version entry, so the output grows quadratically
(257 MB for `@types/node`).

- Evidence: 52 repositories, 294 unique dependencies, `max_parallel = 4`:
  `npm view` only ≈ 55 s (≈10 s with 24 slots, CPU-bound); HTTP client cold
  3.2 s; relaunch within the cache TTL 1.3 s. Direct packument latency 0.4–0.8 s,
  the same as `npm view`'s, so concurrency — not process count — is the lever
  once CPU is out of the way.
- Known parity gaps, all covered by the fallback: `noproxy`, `ca` (inline PEM),
  client certificates, `always-auth`, and `npm_config_*` overrides of
  non-credential keys set only for the project layer.

### D2 — Query and selection model

One request per dependency returns all versions with their `engines`, plus the
dist-tags map. Selection:
target = `dist-tags.latest`; if excluded by any filter, fall back to the highest
version passing all filters ("greatest"). Filters: prerelease excluded unless the
declared current version is a prerelease; `engines.node` compatibility when the
project declares `engines.node` (candidate with no `engines` allowed); deprecated
included by default. Non-registry specs (`file:`, `link:`, `workspace:`,
`catalog:`, `portal:`, `owner/repo`) are skipped.

### D3 — Injection seam and package layout

Introduce a `Registry` interface in `scanner` (mirroring today's `Runner`) that
returns per-dependency metadata for a project directory and npm's
`min-release-age`; production wires `registry.Cache` → `registry.HTTPFetcher`
→ `registry.NpmViewFetcher`, tests use a fake. `npm ls`/`npm audit` keep using
`Runner`. New packages: `registry` (npm config resolver, HTTP client, npm view
fallback, cache, version selection) and `ncurc` (config reader).
`detect` gains the deep recursive walk and `packageManager` parsing.

### D4 — Deep scanning is a recursive `**/package.json` walk (4A)

`ncu --deep` is an alias of `--packageFile '**/package.json'`, not workspace
expansion. Replicate as a recursive walk that includes the root manifest and
nested manifests, skipping `node_modules` and `.pnpm-store`, with per-manifest
labels relative to the scanned path. This differs from `detect.Repos` (folder
mode stops at the first manifest per branch); keep the two walks distinct.

### D5 — Update commands install direct versions (3A)

Project command becomes `cd <dir> && <add> <pkg>@<newVersion> …`, `<add>` =
`npm install` / `pnpm add` / `yarn add`. Global stays `npm install -g …`. Marks
narrow the same command. Accept npm's default `save-prefix` (`^`) rather than
reproducing `upgradeDependencyDeclaration` byte-for-byte.

### D6 — `.ncurc` handling (2B + 1B)

Load the nearest supported data file (`.ncurc`, `.ncurc.json`, `.ncurc.yaml`,
`.ncurc.yml`), per manifest directory in deep mode. Apply only `filter`,
`reject`, `filterVersion`, `rejectVersion`, `pre`, `deprecated`, and `target`
when it names a dist-tag. Warn (UI status) when the file contains keys outside
this set or when only an executable `.ncurc.js/.cjs/.mjs` exists.

### D7 — Cooldown and `packageManager` parity (2B)

Read the effective `min-release-age` from npm config per project (via
`npm config get`, run in the project directory) and, when set, exclude versions
published more recently than that, using the `time` field from the query. Include
the manifest's `packageManager` field (`name@version`) as a dependency, matching
`ncu`'s default sections (`prod`, `dev`, `optional`, `packageManager`; note:
`peer` is excluded).

### D8 — Audits run on demand

`npm audit` costs ~8 s per repository (arborist tree load plus the bulk
advisory request), which dominated a 62-repository folder scan (62 × 8 s over
4 slots ≈ 2 min) once the registry queries were cached. Scans now emit
`audit.StatusPending` for npm/pnpm projects; the UI runs
`orchestrator.AuditProject` once for the selected project and applies the
result through its existing event choke point, discarding results superseded
by a rescan. Source aggregates show `audit pending` until a project is
audited.

- Alternative: audit every project after all scans finish (background
  phase). Rejected: still spawns one `npm audit` per repository the user never
  opens.
- Not pursued: batching `npm view` across packages. npm 12 honors only the
  first spec of a multi-package `npm view` (verified with `lodash` + `is-odd`:
  only lodash entries returned), so one process per unique package remains.

### D9 — Registry metadata is cached on disk with a TTL

`registry.Cache` memoizes per package name: within a process an entry is reused
until an explicit refresh; entries persisted to `registry-cache.json` (next to
`config.toml`) are reused across launches while younger than `cache_ttl`
(default `1h`; `"0"` disables persistence). Rescans (`r`, `R`, per-entry retry)
run with `registry.WithRefresh`, bypassing cached entries; adding a path reuses
them. Writes are debounced and atomic (temp file + rename); a malformed file is
ignored and overwritten. Failures are never cached. Entries are keyed by package
name only, so a name served by two different registries from two projects
shares one entry — accepted: scoped private packages map to one registry per
organization, and the fallback path is unaffected.

- Alternative: SQLite. Rejected: adds a cgo or pure-Go driver for ~300 rows
  (5 MB of JSON for 294 packages), and a relational store buys nothing over a
  map serialized once per scan.
- Alternative: ETag revalidation instead of a TTL. Rejected: a 304 round trip
  costs 0.37 s, the same as a fresh fetch, so it saves bandwidth, not time.

### D10 — Audits run as a phase after the version scans, from the lockfile, cached by fingerprint

Audits stay on `npm audit` / `pnpm audit` for exact parity, but three changes
take them off the critical path:

1. **Phase, not part of the scan.** Scans emit `audit.StatusPending`; when the
   scan stream of a launch (or of a rescan) closes, the UI queues every pending
   project in panel order and keeps at most `max_parallel` audits in flight
   (`ui.auditQueue`). Selecting a project starts its audit immediately, ahead
   of the queue — the on-demand audit of D8 is the queue's priority path, not
   a second mechanism. Results stream in as they land.
2. **`npm audit --json --package-lock-only`.** Arborist audits the lockfile
   instead of `node_modules`: same report (verified on `bff-auth`: 89
   vulnerabilities either way), no tree read, and projects that are not
   installed can be audited. pnpm already audits from `pnpm-lock.yaml`.
3. **`audit-cache.json`** (`memo.Store[audit.Result]`, next to `config.toml`):
   key = directory + SHA-256 of the lockfile (`package-lock.json`,
   `npm-shrinkwrap.json` or `pnpm-lock.yaml`; `package.json` when none),
   same `cache_ttl`, same refresh rule as the registry cache. A changed
   lockfile misses the cache by construction. Failed and not-available
   results are never stored.

- Evidence: with npm's cache warm, `npm audit --json` costs 1.1–2.4 s per
  repository (1.9 s user CPU); the first measurement of 8.5 s was with a cold
  npm cache (arborist downloads packuments to compute fixes). 52 repositories
  at `max_parallel = 4` ≈ 26 s in the background after the versions land;
  relaunch within the TTL with unchanged lockfiles spawns no audit.
- Alternative rejected: a native client on
  `POST /-/npm/v1/security/advisories/bulk` (0.68 s for 784 names). It returns
  31 vulnerable packages for `bff-auth` where `npm audit` reports 89: arborist
  also flags every dependent whose range admits only vulnerable versions
  ("metavulns"), recursively, using the lockfile graph and packuments.
  Reproducing that means lockfile v1/v2/v3 and pnpm v9 parsers, graph
  resolution, metavuln propagation and `fixAvailable` semantics — ~1000 lines
  that drift from npm, to gain ~3 s versus ~26 s once per TTL.
- The generic store (`memo.Store`) was extracted from the registry cache so
  both caches share one singleflight/TTL/persistence mechanism.

## Risks / Trade-offs

- **Process count/latency** (one `npm view` per dependency vs one `ncu` per
  project) → bounded by the existing `max_parallel`; a per-scan cache can be
  added later without changing the specs.
- **`npm view` output shape is version-dependent** (npm 12 wraps results in
  arrays; `*` does not expand) → use `>=0.0.0-0`, parse both array and object
  forms, and cover with fixtures.
- **Private-registry auth edge cases** → the HTTP client reads credentials from
  npmrc files itself; any mismatch (401/403/404, TLS, proxy) falls back to
  `npm view`, which applies npm's full resolution. The manual spike (task 7.2)
  is mandatory before release: it verifies the fast path, not only the result.
- **Homebrew loses the transitive `node` that `npm-check-updates` provided** →
  see Open Questions; `npm` is now the only runtime tool.
- **`.ncurc` `filter`/`reject` glob semantics** differ from `ncu`'s picomatch
  (braces, `**`) → document the supported subset, use a `**`-capable matcher,
  and warn on out-of-scope config.
- **Behavioral drift from `ncu`** as either tool evolves → lock the subset with
  spec scenarios and fixture tests; the `update-config` warning makes remaining
  gaps visible.

## Migration Plan

1. Add the `registry`/`ncurc` packages behind new interfaces; wire `scanner` to
   them; delete the ncu preflight from `main.go`.
2. Switch deep discovery and `packageManager` parsing in `detect`.
3. Switch `command` to the direct-install form; update `command` tests.
4. Update `.goreleaser.yaml`, `README.md`, and `docs/` to drop ncu.
5. Rollback: revert the change (ncu remains installable but unused). No data or
   config migration on disk; existing `lazyncu` config files are unaffected.

## Open Questions

- **Homebrew `depends_on`**: with `npm-check-updates` removed, `brew install
  luchrv/tap/lazyncu` no longer brings `node`/`npm` transitively. Recommend
  adding `depends_on formula: node`. Deferrable — the current `release-pipeline`
  delta only forbids the npm-check-updates dependency and is compatible either
  way; confirm before the release task.
- ~~**Per-project `npm config get` cost**~~ Resolved by D1/D9: one
  `npm config list --json` per process; project `.npmrc` overlaid locally.
