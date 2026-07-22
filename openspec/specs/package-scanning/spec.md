# package-scanning Specification

## Purpose
TBD - created by archiving change add-ncu-tui-dashboard. Update Purpose after archive.
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
The system SHALL scan a path detected as `single` by executing `ncu --jsonUpgraded` against that path's `package.json`, reading current versions from that `package.json`.

#### Scenario: Project with outdated dependencies
- **WHEN** a single-mode scan finds upgradable packages
- **THEN** the project appears as one dashboard entry listing each package with current and new versions

#### Scenario: Project fully up to date
- **WHEN** a single-mode scan finds no upgradable packages
- **THEN** the project appears as up to date with zero pending packages

### Requirement: Deep paths are scanned recursively
The system SHALL scan a path detected as `deep` by executing `ncu --deep --jsonUpgraded` with the path as working directory, and SHALL expand the result into one dashboard entry per discovered `package.json`, labeled with its path relative to the registered path.

#### Scenario: Folder with multiple projects
- **WHEN** a deep scan over a registered folder finds three projects with updates
- **THEN** the dashboard shows three child entries under that source, each with its own package list

#### Scenario: Nested monorepo inside a folder
- **WHEN** a registered folder contains a project that is itself a monorepo with workspaces
- **THEN** each workspace `package.json` found by the deep scan appears as its own entry

### Requirement: Sources are scanned in parallel on launch
The system SHALL start scans for all sources (global plus every registered path) concurrently at application launch, delivering each source's result to the UI as soon as it completes.

#### Scenario: Slow source does not block others
- **WHEN** one source takes 30 seconds and another takes 2 seconds
- **THEN** the fast source's results are displayed as soon as they are ready, while the slow source still shows a loading state

### Requirement: Scan failures are isolated per source
The system SHALL confine any scan failure (non-zero exit, timeout, malformed JSON) to its source, displaying an error state for that source while other sources render normally. The application SHALL NOT crash on scan failures.

#### Scenario: One source fails
- **WHEN** a registered path's scan exits with an error
- **THEN** that source shows an error state with the failure reason and every other source displays its results

#### Scenario: Scan exceeds timeout
- **WHEN** a scan runs longer than the configured timeout
- **THEN** the scan is terminated and its source shows a timeout error state

