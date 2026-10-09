# Spec Delta

## Purpose

Resolve, for every dependency of a scanned manifest, the registry version that
`lazyncu` should suggest as an upgrade — using the same registry, credentials,
and version-selection rules npm projects already rely on.

## ADDED Requirements

### Requirement: Registry metadata is queried per dependency
The system SHALL obtain, for each dependency considered for upgrade, the set of
published versions with each version's `engines.node` and deprecation state
and the package's dist-tags, from the package's registry, using a single
request per dependency: the abbreviated packument (`Accept:
application/vnd.npm.install-v1+json`) over HTTP, bounded by the configured
`max_requests`. Publish times SHALL be requested (full packument) only when a
release cooldown applies. When the HTTP request fails for any reason
(configuration, network, non-2xx status, unparseable body) the system SHALL
fall back to `npm view` for that dependency.

#### Scenario: Metadata retrieved for an upgradable package
- **WHEN** a dependency has published versions and a `latest` dist-tag
- **THEN** the system has the full version list, each version's `engines.node`, and the dist-tags available to select a target

#### Scenario: Dependency with no published versions
- **WHEN** the registry returns no versions for a dependency
- **THEN** the dependency is not suggested for upgrade and the scan continues

#### Scenario: HTTP request fails
- **WHEN** the registry answers a dependency's request with 404 or the connection fails
- **THEN** the same dependency is resolved through `npm view` and its siblings are unaffected

### Requirement: Registry and authentication are resolved from npm's configuration
The system SHALL resolve the registry location, scoped registries, TLS
(`strict-ssl`, `cafile`), proxy (`proxy`, `https-proxy`) and authentication from
npm's own configuration layers — global, user, environment (one `npm config
list --json` per process) and the project's `.npmrc` — so that dependencies
served by private or scoped registries are queried with the same credentials
npm uses. Credentials SHALL be read from the npmrc files npm reports, the
project `.npmrc` and `npm_config_//…` variables, matched by registry prefix the
way npm does, and SHALL never be logged, displayed or written to the cache.

#### Scenario: Dependency from a scoped private registry
- **WHEN** a dependency's scope is mapped to a private registry in npm configuration
- **THEN** its metadata is requested from that registry with the configured credentials

#### Scenario: Authenticated registry
- **WHEN** an npmrc file carries `//host/:_authToken` (or `_auth`, or `username` with `_password`) for the target registry
- **THEN** the request carries the matching `Authorization` header and no credential appears in any output or file lazyncu writes

#### Scenario: Project npmrc overrides
- **WHEN** a project's `.npmrc` maps a scope to a different registry than the user's configuration
- **THEN** that project's dependencies in the scope are requested from the project's registry while other projects keep the user's

### Requirement: Registry metadata is cached with a TTL
The system SHALL reuse a package's metadata within one process, SHALL persist
it to `registry-cache.json` in the config directory, and SHALL reuse persisted
entries across launches while younger than the configured `cache_ttl`
(default 1 hour; `"0"` disables persistence). An explicit rescan SHALL bypass
cached entries and replace them. Failed lookups SHALL NOT be cached. A missing
or malformed cache file SHALL be ignored.

#### Scenario: Relaunch within the TTL
- **WHEN** the application is launched again 10 minutes after a scan with `cache_ttl = "1h"`
- **THEN** no registry request is made for the packages scanned before and the results match the previous scan

#### Scenario: Entry expired
- **WHEN** a persisted entry is older than `cache_ttl`
- **THEN** it is fetched again and the new result replaces it

#### Scenario: Rescan bypasses the cache
- **WHEN** the user presses `r` on a source whose packages are cached
- **THEN** every package of that source is fetched again and the cache is updated

### Requirement: Target version is the latest dist-tag with a stable fallback
The system SHALL select, per dependency, the version published to the `latest`
dist-tag; when that version is excluded by a filter, the system SHALL fall back
to the highest published version that passes all filters.

#### Scenario: Latest tag used
- **WHEN** a dependency's `latest` dist-tag points to a version that passes all filters
- **THEN** that version is the suggested target

#### Scenario: Latest excluded falls back
- **WHEN** a dependency's `latest` dist-tag points to a prerelease version and the current version is stable
- **THEN** the highest stable published version that passes all filters is suggested instead

### Requirement: Prerelease versions are excluded unless the current version is a prerelease
The system SHALL exclude prerelease versions from suggestions unless the
dependency's declared current version is itself a prerelease.

#### Scenario: Stable current version
- **WHEN** a dependency is declared as `^1.2.3` and only `2.0.0-beta.1` is newer
- **THEN** no prerelease is suggested

#### Scenario: Prerelease current version
- **WHEN** a dependency is declared as `2.0.0-beta.1` and `2.0.0-beta.5` exists
- **THEN** the newer prerelease may be suggested

### Requirement: Suggested versions are compatible with the project's engines.node
The system SHALL, when a project declares `engines.node`, limit that project's
suggestions to versions whose declared `engines.node` is satisfied by the
project's minimum node version. A candidate that declares no `engines.node` is
always allowed.

#### Scenario: Newest version incompatible with declared node
- **WHEN** a project declares `engines.node: ">=16 <17"` and a dependency's latest version requires node >= 20
- **THEN** the newest version satisfying the constraint is suggested, or the dependency is omitted if none does

#### Scenario: Candidate without engines
- **WHEN** a project declares `engines.node` and a candidate version declares no `engines.node`
- **THEN** the candidate is allowed

#### Scenario: Project without engines declaration
- **WHEN** a project declares no `engines.node`
- **THEN** suggestions are not constrained by engines regardless of what candidates declare

### Requirement: Deprecated versions are included by default
The system SHALL allow deprecated published versions to be suggested by default,
matching the reference behavior, unless a project's configuration excludes them.

#### Scenario: Deprecated latest
- **WHEN** a dependency's newest version passing the other filters is marked deprecated
- **THEN** it may be suggested by default

### Requirement: Non-registry dependency specs are ignored
The system SHALL ignore dependency specifications that do not reference a
registry version — `file:`, `link:`, `workspace:`, `catalog:`, `portal:`, and
short GitHub `owner/repo` references — without raising an error.

#### Scenario: Protocol spec is skipped
- **WHEN** a dependency's declared spec is `workspace:*` or `file:../local`
- **THEN** the dependency is not queried and no error is raised

#### Scenario: Mixed specs in one manifest
- **WHEN** a manifest mixes registry specs with non-registry specs
- **THEN** only the registry specs are queried and the non-registry ones appear with unknown severity

### Requirement: A failed registry query does not fail the scan
The system SHALL treat a dependency whose registry metadata cannot be retrieved
(error, timeout, or unparseable response) as not upgradable, and SHALL NOT fail
the surrounding project scan because of it.

#### Scenario: One dependency fails to fetch
- **WHEN** one dependency's registry query errors while its siblings succeed
- **THEN** the siblings' upgrades are reported and the failed dependency is omitted
