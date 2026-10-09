# Spec Delta

## REMOVED Requirements

### Requirement: ncu availability is verified at startup
**Reason**: The runtime dependency on the `ncu` executable is removed; the scan no longer invokes npm-check-updates.
**Migration**: No user action. `lazyncu` no longer requires `npm install -g npm-check-updates`; it relies on `npm`, which it already required.

## MODIFIED Requirements

### Requirement: Global packages are scanned
The system SHALL scan globally installed packages by reading the installed set and current versions from `npm ls -g --depth=0 --json` and resolving upgrade targets from the registry, producing a package list with name, current version, and new version. A failed installed-version lookup SHALL degrade to no upgradable packages rather than failing the source.

#### Scenario: Global packages with updates
- **WHEN** the global scan finds upgradable packages
- **THEN** each package is reported with its name, installed version, and latest version

#### Scenario: Installed-version lookup fails
- **WHEN** `npm ls -g` fails
- **THEN** the global source reports no upgradable packages and is not marked as failed

### Requirement: Single projects are scanned
The system SHALL scan a path detected as `single` by reading the dependency versions declared in that path's `package.json` — including the `packageManager` field — and resolving their upgrade targets from the registry. When the manifest declares `engines.node`, suggested upgrades SHALL be limited to versions satisfying that constraint; without `engines.node`, results are unchanged.

#### Scenario: Project with outdated dependencies
- **WHEN** a single-mode scan finds upgradable packages
- **THEN** the project appears as one dashboard entry listing each package with current and new versions

#### Scenario: Project fully up to date
- **WHEN** a single-mode scan finds no upgradable packages
- **THEN** the project appears as up to date with zero pending packages

#### Scenario: Upgrade incompatible with declared node version
- **WHEN** a project declares `engines.node: ">=16 <17"` and a dependency's latest version requires node >= 20
- **THEN** the scan suggests the newest version satisfying the constraint (or omits the package), never the incompatible latest

### Requirement: Deep paths are scanned recursively
The system SHALL scan a path detected as `deep` (a workspaces monorepo) by discovering its manifests recursively and resolving each manifest's upgrade targets from the registry, producing one dashboard entry per discovered `package.json`, labeled with its path relative to the scanned path. Engines-aware filtering applies per discovered manifest: each project's declared `engines.node` constrains its own suggestions, and projects without one are unaffected. A path detected as `folder` SHALL NOT be scanned as a monorepo; it follows the folder scanning requirement.

#### Scenario: Monorepo with workspaces
- **WHEN** a deep scan over a registered monorepo finds three workspace packages with updates
- **THEN** the dashboard shows three child entries under that source, each with its own package list

#### Scenario: Folder with multiple projects
- **WHEN** a registered folder has no root `package.json` and contains three repositories
- **THEN** the folder is not scanned as a monorepo; the three repositories are scanned per repository as defined by the folder scanning requirement

#### Scenario: Nested monorepo inside a folder
- **WHEN** a folder source contains a repository that is itself a monorepo with workspaces
- **THEN** that repository is scanned as a monorepo scoped to the repository, and each workspace `package.json` appears as its own entry under the folder source

### Requirement: Sources are scanned in parallel on launch
The system SHALL start scans for all sources (global plus every registered path) concurrently at application launch, delivering each source's results to the UI as soon as they are available — per repository for folder sources, per source otherwise. The number of external commands (`npm config`, `npm view`, `npm ls`, `npm audit`) running at the same time across all sources SHALL never exceed the configured `max_parallel` bound, and the number of registry HTTP requests in flight SHALL never exceed the configured `max_requests` bound; commands and requests waiting for a free slot SHALL NOT consume their own timeout budget while waiting.

#### Scenario: Slow source does not block others
- **WHEN** one source takes 30 seconds and another takes 2 seconds
- **THEN** the fast source's results are displayed as soon as they are ready, while the slow source still shows a loading state

#### Scenario: Concurrency bound is respected
- **WHEN** `max_parallel` is 4 and a folder with 50 repositories is scanned alongside the global source
- **THEN** at most 4 external commands run at any instant and the remaining ones start as slots free up

#### Scenario: Waiting for a slot does not eat the timeout
- **WHEN** a repository's scan waits 40 seconds for a slot with a 30-second timeout
- **THEN** its command still gets the full 30 seconds once it starts

### Requirement: Scan failures are isolated per source
The system SHALL confine any scan-level failure to the unit that failed: a source whose discovery or manifest read fails shows an error state with the reason, and a single repository inside a folder source fails independently while its siblings continue and render normally. A dependency whose registry query fails (error, timeout, or unparseable response) SHALL be omitted without failing its project. The application SHALL NOT crash on scan failures.

#### Scenario: One source fails
- **WHEN** a registered path's discovery or scan fails
- **THEN** that source shows an error state with the failure reason and every other source displays its results

#### Scenario: Scan exceeds timeout
- **WHEN** a scan command runs longer than the configured timeout
- **THEN** that command is terminated and its dependency is omitted without failing the source

#### Scenario: One repository fails inside a folder
- **WHEN** a folder with 50 repositories has one repository whose scan fails
- **THEN** the folder source stays usable, 49 repositories show their results, and only the failed repository shows an error state with its reason

#### Scenario: Folder cannot be read
- **WHEN** the registered folder itself cannot be read during discovery
- **THEN** the folder source shows an error state with the reason
