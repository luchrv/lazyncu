# dashboard-ui Delta Specification

## MODIFIED Requirements

### Requirement: First run shows an empty-state call to action
When no paths are registered, the system SHALL show a hint in the sources tree ("No paths registered." / "Press a to add one.") as non-selectable entries under the global source, and the Packages panel SHALL show onboarding text explaining that `a` accepts a single project, a monorepo, or a folder of projects with automatic detection, plus a pointer to `?`. The onboarding text SHALL only replace empty package views: actual scan results and scan errors SHALL always take priority over it. The hints SHALL disappear once a path is registered.

On a first launch — identified by the configuration file having just been created — the system SHALL additionally open the add-path folder browser automatically, once, titled as a welcome ("Welcome — pick a folder to scan…"). Launches with an existing configuration file SHALL never auto-open it, regardless of how many paths are registered. Dismissing the auto-opened browser without adding a path SHALL show a status hint pointing at `a` and `?`; adding a path from it behaves exactly like the normal add-path flow.

#### Scenario: Empty first launch
- **WHEN** the app launches with no registered paths and the global scan finds nothing to update
- **THEN** the tree shows the add-a-path hint and the Packages panel shows the onboarding text instead of the up-to-date message

#### Scenario: Real data wins over onboarding
- **WHEN** no paths are registered but the global source has pending updates
- **THEN** the Packages panel shows the global package rows, not the onboarding text

#### Scenario: Hint disappears after adding a path
- **WHEN** the user registers a valid path with `a`
- **THEN** the tree hint entries are removed and the new source appears scanning

#### Scenario: Very first launch opens the browser
- **WHEN** the app starts and the configuration file did not exist before this launch
- **THEN** the add-path browser opens automatically with the welcome title

#### Scenario: Subsequent empty launches stay passive
- **WHEN** the app starts with an existing configuration file that registers no paths
- **THEN** no modal opens automatically and the passive onboarding is shown

#### Scenario: Dismissing the welcome browser
- **WHEN** the auto-opened browser is closed with `Esc` without adding a path
- **THEN** a status hint points at `a` to add a path and `?` for the keybindings

#### Scenario: Adding a path from the welcome browser
- **WHEN** the user picks a folder in the auto-opened browser and confirms
- **THEN** the path is registered and scanned exactly as in the normal add-path flow
