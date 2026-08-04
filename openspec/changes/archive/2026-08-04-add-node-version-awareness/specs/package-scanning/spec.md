# package-scanning Delta Specification

## MODIFIED Requirements

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
The system SHALL scan a path detected as `deep` by executing `ncu --deep --jsonUpgraded --enginesNode` with the path as working directory, and SHALL expand the result into one dashboard entry per discovered `package.json`, labeled with its path relative to the registered path. Engines-aware filtering applies per discovered manifest: each project's declared `engines.node` constrains its own suggestions, and projects without one are unaffected.

#### Scenario: Folder with multiple projects
- **WHEN** a deep scan over a registered folder finds three projects with updates
- **THEN** the dashboard shows three child entries under that source, each with its own package list

#### Scenario: Nested monorepo inside a folder
- **WHEN** a registered folder contains a project that is itself a monorepo with workspaces
- **THEN** each workspace `package.json` found by the deep scan appears as its own entry

## ADDED Requirements

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
