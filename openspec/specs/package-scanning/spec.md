# package-scanning Specification

## Purpose
Run ncu and npm through an injected runner to discover pending upgrades — globally installed packages plus every registered path (single project, monorepo, or folder scanned per repository) — with engines-aware suggestions, per-project node context, bounded parallel scans, and per-source and per-repository failure isolation.

## Requirements

### Requirement: ncu availability is verified at startup
The system SHALL verify at startup that the `ncu` executable is available on PATH with major version >= 18, and SHALL show an actionable error screen (including the install command `npm install -g npm-check-updates`) when the check fails.

#### Scenario: ncu not installed
- **WHEN** the application starts and `ncu` is not found on PATH
- **THEN** an error screen explains the missing dependency and how to install it, and no scans are attempted

#### Scenario: ncu too old
- **WHEN** the application starts and `ncu --version` reports a major version below 18
- **THEN** an error screen explains the minimum version requirement

### Requirement: Global packages are scanned
The system SHALL scan globally installed packages by executing `ncu -g --jsonUpgraded` and SHALL obtain currently installed versions via `npm ls -g --depth=0 --json`, producing a package list with name, current version, and new version.

#### Scenario: Global packages with updates
- **WHEN** the global scan finds upgradable packages
- **THEN** each package is reported with its name, installed version, and latest version

#### Scenario: Installed-version lookup fails
- **WHEN** `ncu -g` succeeds but `npm ls -g` fails
- **THEN** packages are reported with their new version and an unknown severity, and the source is not marked as failed

### Requirement: Single projects are scanned
The system SHALL scan a path detected as `single` by executing `ncu --jsonUpgraded --enginesNode` against that path's `package.json`, reading current versions from that `package.json`. When the manifest declares `engines.node`, suggested upgrades SHALL be limited to versions satisfying that constraint; without `engines.node`, results are unchanged.

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
The system SHALL scan a path detected as `deep` (a workspaces monorepo) by executing `ncu --deep --jsonUpgraded --enginesNode` with that path as working directory, and SHALL expand the result into one dashboard entry per discovered `package.json`, labeled with its path relative to the scanned path. Engines-aware filtering applies per discovered manifest: each project's declared `engines.node` constrains its own suggestions, and projects without one are unaffected. A path detected as `folder` SHALL NOT be scanned with `ncu --deep`; it follows the folder scanning requirement.

#### Scenario: Monorepo with workspaces
- **WHEN** a deep scan over a registered monorepo finds three workspace packages with updates
- **THEN** the dashboard shows three child entries under that source, each with its own package list

#### Scenario: Folder with multiple projects
- **WHEN** a registered folder has no root `package.json` and contains three repositories
- **THEN** no `ncu --deep` runs over the folder; the three repositories are scanned per repository as defined by the folder scanning requirement

#### Scenario: Nested monorepo inside a folder
- **WHEN** a folder source contains a repository that is itself a monorepo with workspaces
- **THEN** that repository is scanned with `ncu --deep` scoped to the repository, and each workspace `package.json` appears as its own entry under the folder source

### Requirement: Sources are scanned in parallel on launch
The system SHALL start scans for all sources (global plus every registered path) concurrently at application launch, delivering each source's results to the UI as soon as they are available — per repository for folder sources, per source otherwise. The number of external commands (`ncu`, `npm ls`, `npm audit`) running at the same time across all sources SHALL never exceed the configured `max_parallel` bound; commands waiting for a free slot SHALL NOT consume their own timeout budget while waiting.

#### Scenario: Slow source does not block others
- **WHEN** one source takes 30 seconds and another takes 2 seconds
- **THEN** the fast source's results are displayed as soon as they are ready, while the slow source still shows a loading state

#### Scenario: Concurrency bound is respected
- **WHEN** `max_parallel` is 4 and a folder with 50 repositories is scanned alongside the global source
- **THEN** at most 4 external commands run at any instant and the remaining ones start as slots free up

#### Scenario: Waiting for a slot does not eat the timeout
- **WHEN** a repository's `ncu` run waits 40 seconds for a slot with a 30-second timeout
- **THEN** its command still gets the full 30 seconds once it starts

### Requirement: Scan failures are isolated per source
The system SHALL confine any scan failure (non-zero exit, timeout, malformed JSON) to the unit that failed: a source for global, single and deep sources; a single repository for folder sources, whose siblings continue and render normally. A failing unit SHALL display an error state with the failure reason. The application SHALL NOT crash on scan failures.

#### Scenario: One source fails
- **WHEN** a registered path's scan exits with an error
- **THEN** that source shows an error state with the failure reason and every other source displays its results

#### Scenario: Scan exceeds timeout
- **WHEN** a scan command runs longer than the configured timeout
- **THEN** that command is terminated and its unit (source or repository) shows a timeout error state

#### Scenario: One repository fails inside a folder
- **WHEN** a folder with 50 repositories has one repository whose `ncu` run fails
- **THEN** the folder source stays usable, 49 repositories show their results, and only the failed repository shows an error state with its reason

#### Scenario: Folder cannot be read
- **WHEN** the registered folder itself cannot be read during discovery
- **THEN** the folder source shows an error state with the reason

### Requirement: Project node context is captured in scan results
Each scanned project SHALL carry its declared node context: the trimmed content of a `.nvmrc` file in the project directory and the `engines.node` constraint from its `package.json`, each empty when absent. A missing or unreadable `.nvmrc` or manifest SHALL degrade to an empty value, never fail the scan.

#### Scenario: Project with .nvmrc
- **WHEN** a scanned project directory contains `.nvmrc` with `18.19.0`
- **THEN** the project's scan result carries nvmrc context `18.19.0`

#### Scenario: Project with engines.node only
- **WHEN** a scanned project's `package.json` declares `engines.node: ">=18"` and no `.nvmrc` exists
- **THEN** the project's scan result carries engines context `>=18` and empty nvmrc context

#### Scenario: Project without node declarations
- **WHEN** a scanned project has neither `.nvmrc` nor `engines.node`
- **THEN** both context values are empty and the scan succeeds normally

### Requirement: Folder sources are scanned per repository
The system SHALL scan a path detected as `folder` by first discovering its repositories, then scanning each discovered repository as an independent unit through the single/deep detection applied to that repository, with its own command timeout. Each repository's results SHALL be delivered as soon as that repository completes, and the set of discovered repositories SHALL be announced before any of them completes so the UI can show what is pending.

#### Scenario: Folder with many repositories
- **WHEN** a folder source contains 50 repositories
- **THEN** 50 independent scans run under the concurrency bound, each repository appears with its own results when it finishes, and no single command's timeout covers more than one repository

#### Scenario: Repository list is announced first
- **WHEN** discovery of a folder finishes and its repository scans have not yet completed
- **THEN** the UI already knows the labels of every discovered repository

#### Scenario: Symlinked repository scans in place
- **WHEN** a discovered repository was reached through a symbolic link
- **THEN** its scan runs against the linked directory and its entry keeps the link's label

#### Scenario: Empty folder
- **WHEN** discovery of a folder yields no repository
- **THEN** the folder source completes with zero entries and no error
