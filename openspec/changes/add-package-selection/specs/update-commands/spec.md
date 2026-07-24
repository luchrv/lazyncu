# update-commands Specification (delta)

## ADDED Requirements

### Requirement: Selection filters the suggested update command
When one or more packages of the current entry are marked, the system SHALL show a filtered update command instead of the full one: for a project, `cd <projectDir> && ncu -u <pkg…> && <install>` with the marked package names as positional ncu filters and `<install>` per the detected package manager; for the global source, `npm install -g <pkg>@<newVersion> …` restricted to the marked packages. With no marks the full command SHALL be shown unchanged, and the vulnerability `fix:` command SHALL never be affected by marks.

#### Scenario: Filtered project command
- **WHEN** `lodash` and `debug` are marked in an npm project
- **THEN** the suggested command is `cd <projectDir> && ncu -u lodash debug && npm install`

#### Scenario: Filtered global command
- **WHEN** only `typescript` → 5.6.2 is marked in the global source that also has other upgradable packages
- **THEN** the suggested command is `npm install -g typescript@5.6.2`

#### Scenario: Clearing the selection restores the full command
- **WHEN** the last mark of the current project is removed
- **THEN** the suggested command returns to the unfiltered form

#### Scenario: Read-only guarantee still holds with selection
- **WHEN** the user marks packages and copies the filtered command
- **THEN** lazyncu spawns no process that modifies packages — the command is text to copy, nothing more
