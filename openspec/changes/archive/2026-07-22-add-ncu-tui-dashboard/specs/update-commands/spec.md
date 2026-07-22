# update-commands

## ADDED Requirements

### Requirement: Global update command is constructed from scan results
The system SHALL build, for the global source, the command `npm install -g <pkg>@<newVersion> ...` including every upgradable global package with its target version.

#### Scenario: Multiple global packages
- **WHEN** the global scan reports `typescript` → 5.6.2 and `npm-check-updates` → 18.1.0
- **THEN** the suggested command is `npm install -g typescript@5.6.2 npm-check-updates@18.1.0`

#### Scenario: No global updates
- **WHEN** the global scan reports no upgradable packages
- **THEN** no update command is shown for the global source

### Requirement: Project update command matches the detected package manager
The system SHALL build, for each project entry, the command `cd <projectDir> && ncu -u && <install>` where `<install>` is `npm install`, `pnpm install`, or `yarn` according to the detected package manager.

#### Scenario: pnpm project command
- **WHEN** a project's package manager is detected as pnpm
- **THEN** the suggested command is `cd <projectDir> && ncu -u && pnpm install`

#### Scenario: yarn project command
- **WHEN** a project's package manager is detected as yarn
- **THEN** the suggested command is `cd <projectDir> && ncu -u && yarn`

#### Scenario: Default npm command
- **WHEN** a project has no recognized lockfile
- **THEN** the suggested command is `cd <projectDir> && ncu -u && npm install`

### Requirement: The dashboard never executes update commands
The system SHALL display suggested update commands as text only and SHALL NOT execute any command that modifies packages, package files, or global installations.

#### Scenario: Read-only guarantee
- **WHEN** the user interacts with any dashboard element, including the command bar
- **THEN** no process is spawned other than the read-only scan commands (`ncu` without `-u`, `npm ls`)
