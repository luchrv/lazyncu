# update-commands Specification

## Purpose
Construct the exact update command for the current selection — global, per project by package manager, or narrowed to marked packages — for the user to copy; the dashboard never executes it.

## Requirements

### Requirement: Global update command is constructed from scan results
The system SHALL build, for the global source, the command `npm install -g <pkg>@<newVersion> ...` including every upgradable global package with its target version.

#### Scenario: Multiple global packages
- **WHEN** the global scan reports `typescript` → 5.6.2 and `npm-check-updates` → 18.1.0
- **THEN** the suggested command is `npm install -g typescript@5.6.2 npm-check-updates@18.1.0`

#### Scenario: No global updates
- **WHEN** the global scan reports no upgradable packages
- **THEN** no update command is shown for the global source

### Requirement: Project update command matches the detected package manager
The system SHALL build, for each project entry, the command `cd <projectDir> && <add> <pkg>@<newVersion> …` listing every upgradable package of that project, where `<add>` is `npm install`, `pnpm add`, or `yarn add` according to the detected package manager.

#### Scenario: pnpm project command
- **WHEN** a pnpm project has `lodash` upgradable to `4.17.21`
- **THEN** the suggested command is `cd <projectDir> && pnpm add lodash@4.17.21`

#### Scenario: yarn project command
- **WHEN** a yarn project has `lodash` upgradable to `4.17.21`
- **THEN** the suggested command is `cd <projectDir> && yarn add lodash@4.17.21`

#### Scenario: Default npm command
- **WHEN** a project has no recognized lockfile and has `lodash` upgradable to `4.17.21`
- **THEN** the suggested command is `cd <projectDir> && npm install lodash@4.17.21`

### Requirement: The dashboard never executes update commands
The system SHALL display suggested update commands as text only and SHALL NOT execute any command that modifies packages, package files, or global installations.

#### Scenario: Read-only guarantee
- **WHEN** the user interacts with any dashboard element, including the command bar
- **THEN** no process is spawned other than the read-only scan commands (`npm config list`, `npm view`, `npm ls`, `npm audit`) and no network request other than registry `GET`s is made

### Requirement: Selection filters the suggested update command
When one or more packages of the current entry are marked, the system SHALL show a filtered update command instead of the full one: for a project, `cd <projectDir> && <add> <pkg>@<newVersion> …` restricted to the marked packages, with `<add>` per the detected package manager; for the global source, `npm install -g <pkg>@<newVersion> …` restricted to the marked packages. With no marks the full command SHALL be shown unchanged, and the vulnerability `fix:` command SHALL never be affected by marks.

#### Scenario: Filtered project command
- **WHEN** `lodash` and `debug` are marked in an npm project with targets `4.17.21` and `4.3.4`
- **THEN** the suggested command is `cd <projectDir> && npm install lodash@4.17.21 debug@4.3.4`

#### Scenario: Filtered global command
- **WHEN** only `typescript` → 5.6.2 is marked in the global source that also has other upgradable packages
- **THEN** the suggested command is `npm install -g typescript@5.6.2`

#### Scenario: Clearing the selection restores the full command
- **WHEN** the last mark of the current project is removed
- **THEN** the suggested command returns to the unfiltered form

#### Scenario: Read-only guarantee still holds with selection
- **WHEN** the user marks packages and copies the filtered command
- **THEN** lazyncu spawns no process that modifies packages — the command is text to copy, nothing more
