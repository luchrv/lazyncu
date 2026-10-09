# Proposal

## Why

`lazyncu` is a Go binary that shells out to the `ncu` CLI (npm-check-updates) at
runtime, and its suggested update commands still say `ncu -u`. This forces every
user — including Homebrew installs, prebuilt binaries, and `go install` — to
have npm-check-updates installed, even though `lazyncu` already requires `npm`
for `npm ls -g` and `npm audit`. Removing the dependency makes `lazyncu` a
self-contained tool that never asks the user to install a separate Node CLI,
while keeping the scan results faithful to what `ncu` would report.

## What Changes

- **BREAKING**: `lazyncu` no longer requires or invokes the `ncu` executable at
  runtime. The version scan queries the npm registry directly (packuments over
  HTTP using npm's configuration, `npm view` as fallback) and caches results.
- **BREAKING**: project update commands change from `cd <dir> && ncu -u && <install>`
  to `cd <dir> && <pm> add <pkg>@<version> …` (npm/pnpm/yarn), and marked-package
  commands narrow accordingly.
- The startup preflight no longer checks for `ncu`/its minimum version.
- Deep scans replicate `ncu --deep` semantics as a recursive `**/package.json`
  walk (skipping `node_modules` and `.pnpm-store`), instead of delegating to
  `ncu --deep`.
- Scans honor a supported subset of each project's `.ncurc.*` configuration
  (data formats) and warn when a project carries options outside that subset or
  an executable `.ncurc.js/.cjs/.mjs`.
- The `packageManager` field of a manifest participates as a dependency, matching
  `ncu`'s default dependency sections.
- The Homebrew cask drops its `depends_on` on the `npm-check-updates` formula, and
  the README install-channel requirements are updated.

Non-goals: reimplementing `ncu` options beyond the behavior `lazyncu` actually
uses (no `--peer`, `--minimal`, `-t minor/patch`, `--doctor`, interactive mode,
writing `package.json`), or changing the read-only guarantee — `lazyncu` still
only displays and copies commands.

## Capabilities

### New Capabilities

- `registry-query`: how `lazyncu` resolves upgrade targets from the npm registry
  (direct packument requests with `npm view` as fallback, an on-disk TTL cache,
  target selection from `dist-tags.latest` with the `greatest` fallback,
  prerelease/deprecated filtering, `engines.node` compatibility, and
  registry/auth resolution — including private registries — from npm's own
  configuration).
- `update-config`: how `lazyncu` reads a project's `.ncurc.*` files, which
  options it honors, and how it surfaces out-of-scope options or executable
  configs.

### Modified Capabilities

- `package-scanning`: the ncu-availability preflight requirement is removed;
  single, deep, and global scans source upgrades from the registry client;
  the `packageManager` field is included; deep scans use the recursive manifest
  walk.
- `scan-detection`: `deep` mode no longer delegates to `ncu --deep`; it is a
  recursive `**/package.json` discovery whose results label each manifest.
- `update-commands`: the project update command no longer contains `ncu -u`;
  it installs the resolved versions directly through the detected package
  manager, and selection filtering applies to that form.
- `release-pipeline`: the Homebrew cask no longer declares a `npm-check-updates`
  dependency, and the README requirement about install channels is updated.

## Impact

- `scanner/`: `scanner.go`, `types.go`, `execrunner.go` — replace ncu invocation
  and preflight with the registry client (HTTP transport, `npm view` fallback,
  cache); keep `npm ls -g`.
- `command/`: `command.go`, `filtered.go` — build `npm install`/`pnpm add`/
  `yarn add` commands instead of `ncu -u`.
- `detect/`: deep discovery (recursive `**/package.json`) and `packageManager`
  field parsing.
- New packages: registry client and `.ncurc` reader (names decided in design).
- `main.go`: remove the ncu preflight call.
- `.goreleaser.yaml`: drop the cask `depends_on` on `npm-check-updates`.
- `README.md`, `docs/`: requirements, install channels, and command table.
- Specs: `package-scanning`, `scan-detection`, `update-commands`,
  `release-pipeline` deltas; new `registry-query` and `update-config` specs.
- Runtime dependencies: `npm` (already required) becomes the sole external
  scanning tool; `ncu` is removed.
