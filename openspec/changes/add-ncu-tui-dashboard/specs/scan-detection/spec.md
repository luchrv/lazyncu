# scan-detection

## ADDED Requirements

### Requirement: Scan mode is auto-detected per path
The system SHALL determine the scan mode for each registered path at scan time, without persisting the result, using the following decision tree: if the path contains no `package.json`, the mode is `deep` (folder of projects); if the path contains a `package.json` with a `workspaces` field or a `pnpm-workspace.yaml` file is present, the mode is `deep` (monorepo); otherwise the mode is `single` (plain project).

#### Scenario: Folder containing multiple projects
- **WHEN** a registered path has no `package.json` in its root
- **THEN** the detected mode is `deep`

#### Scenario: Monorepo with npm/yarn workspaces
- **WHEN** a registered path has a `package.json` whose content includes a `workspaces` field
- **THEN** the detected mode is `deep`

#### Scenario: Monorepo with pnpm workspaces
- **WHEN** a registered path has a `package.json` and a `pnpm-workspace.yaml` file
- **THEN** the detected mode is `deep`

#### Scenario: Single project
- **WHEN** a registered path has a `package.json` with no `workspaces` field and no `pnpm-workspace.yaml`
- **THEN** the detected mode is `single`

#### Scenario: Detection is re-evaluated every scan
- **WHEN** a path previously detected as `single` gains a `pnpm-workspace.yaml` before the next launch
- **THEN** the next scan detects it as `deep` without any user action

### Requirement: Package manager is detected from lockfiles
The system SHALL determine the package manager of a project directory from its lockfile: `package-lock.json` → npm, `pnpm-lock.yaml` → pnpm, `yarn.lock` → yarn. When no lockfile is present, the system SHALL default to npm.

#### Scenario: pnpm project
- **WHEN** a project directory contains `pnpm-lock.yaml`
- **THEN** the detected package manager is pnpm

#### Scenario: yarn project
- **WHEN** a project directory contains `yarn.lock`
- **THEN** the detected package manager is yarn

#### Scenario: npm project
- **WHEN** a project directory contains `package-lock.json`
- **THEN** the detected package manager is npm

#### Scenario: No lockfile
- **WHEN** a project directory contains no recognized lockfile
- **THEN** the detected package manager defaults to npm

#### Scenario: Multiple lockfiles present
- **WHEN** a project directory contains more than one recognized lockfile
- **THEN** detection resolves deterministically with precedence pnpm-lock.yaml, then yarn.lock, then package-lock.json
