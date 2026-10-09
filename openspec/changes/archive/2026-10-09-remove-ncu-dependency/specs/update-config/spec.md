# Spec Delta

## Purpose

Read and apply a project's `.ncurc` configuration so that `lazyncu`'s scan
options match what npm-check-updates would use for that project, within a
supported subset, while making any gap in that parity visible.

## ADDED Requirements

### Requirement: Supported `.ncurc` data files are loaded per project
The system SHALL load a project's configuration from the nearest of
`.ncurc`, `.ncurc.json`, `.ncurc.yaml`, or `.ncurc.yml`, searched in the
project directory (and, in deep scans, relative to each discovered manifest).
A missing configuration file is not an error.

#### Scenario: JSON configuration present
- **WHEN** a project directory contains `.ncurc.json` with supported options
- **THEN** those options apply to that project's scan

#### Scenario: YAML configuration present
- **WHEN** a project directory contains `.ncurc.yml` with supported options
- **THEN** those options apply to that project's scan

#### Scenario: No configuration file
- **WHEN** a project directory contains no supported configuration file
- **THEN** the scan uses default options and succeeds

### Requirement: Only supported options are applied
The system SHALL apply only these configuration keys: `filter`, `reject`,
`filterVersion`, `rejectVersion`, `pre`, `deprecated`, and `target` when it
names a dist-tag. It SHALL NOT apply any other key.

#### Scenario: Reject list applied
- **WHEN** `.ncurc` sets `reject: ["@types/*"]`
- **THEN** matching dependencies are excluded from the suggestions

#### Scenario: Dist-tag target applied
- **WHEN** `.ncurc` sets `target: "@next"`
- **THEN** suggestions use the `next` dist-tag for that project

#### Scenario: Unsupported target ignored
- **WHEN** `.ncurc` sets `target: "minor"` (a level, not a dist-tag)
- **THEN** the level is not applied and the default target is used

### Requirement: Out-of-scope options and executable configs are surfaced
The system SHALL warn, visibly, when a project's configuration contains keys
outside the supported set (for example `peer`, `minimal`, `interactive`,
`doctor`) or when only an executable `.ncurc.js`, `.ncurc.cjs`, or `.ncurc.mjs`
is present, so the user knows that project's results are not guaranteed to match
npm-check-updates.

#### Scenario: Out-of-scope key present
- **WHEN** a project's `.ncurc` contains `peer: true`
- **THEN** the supported options still apply and a warning states that the project's results may differ from npm-check-updates

#### Scenario: Executable config present
- **WHEN** a project has `.ncurc.js` and no supported data-format file
- **THEN** the default options apply and a warning states that the executable configuration is not supported

### Requirement: Deep scans apply configuration per manifest
In a deep scan, the system SHALL apply each discovered manifest's own directory
configuration to that manifest, independently of the others.

#### Scenario: Differing per-package configuration
- **WHEN** a monorepo has two manifests with different `.ncurc` files
- **THEN** each manifest is scanned with its own options
