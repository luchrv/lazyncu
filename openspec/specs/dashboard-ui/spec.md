# dashboard-ui Specification

## Purpose
TBD - created by archiving change add-ncu-tui-dashboard. Update Purpose after archive.
## Requirements
### Requirement: Dashboard layout separates sources and packages
The system SHALL render a terminal UI with a sources/projects panel (global source plus every registered path, with deep sources expanded into child project entries), a package table for the selected entry (columns: package, current version, new version, severity), and a command bar showing the suggested update command for the current selection.

#### Scenario: Selecting a project
- **WHEN** the user selects a project entry in the sources panel
- **THEN** the package table shows that project's pending packages and the command bar shows its update command

#### Scenario: Selecting the global source
- **WHEN** the user selects the global source
- **THEN** the package table shows global packages and the command bar shows the `npm install -g ...` command

### Requirement: Severity is color-coded
The system SHALL color package rows by severity — red for major, yellow for minor, green for patch, gray for other — and SHALL display per-project severity counters in the sources panel.

#### Scenario: Row colors
- **WHEN** the package table renders a major, a minor, and a patch upgrade
- **THEN** the rows are colored red, yellow, and green respectively

#### Scenario: Panel counters
- **WHEN** a project has 3 major, 5 minor, and 2 patch upgrades
- **THEN** its entry in the sources panel displays the counters "3 major, 5 minor, 2 patch"

### Requirement: Loading and error states are shown per source
The system SHALL show a loading indicator for each source while its scan is running, replace it with results when the scan completes, and show an error state with the failure reason when the scan fails.

#### Scenario: Progressive loading
- **WHEN** the application launches and scans are in flight
- **THEN** each pending source displays a loading indicator that disappears independently as its scan finishes

#### Scenario: Failed source
- **WHEN** a source's scan fails
- **THEN** its entry shows an error indicator and the failure reason is visible to the user

### Requirement: Update command can be copied to the clipboard
The system SHALL provide a keybinding that copies the currently displayed update command to the system clipboard and confirms the copy with a status message.

#### Scenario: Successful copy
- **WHEN** the user presses the copy keybinding with a command displayed
- **THEN** the command text is placed on the system clipboard and a confirmation message is shown

#### Scenario: Clipboard unavailable
- **WHEN** the clipboard is not available (e.g., headless session)
- **THEN** a non-fatal error message is shown and the command remains visible for manual copying

### Requirement: Widget updates from goroutines are serialized
The system SHALL funnel every UI mutation originating from a scan goroutine through a single dispatch function that wraps tview's `QueueUpdateDraw`, and no goroutine SHALL access widgets directly.

#### Scenario: Concurrent scan completions
- **WHEN** multiple scans complete at nearly the same time
- **THEN** all UI updates are applied without data races and the interface remains consistent

### Requirement: Source project lists can be collapsed and expanded
The system SHALL let the user collapse and expand the project list under a source node in the panel, showing a fold indicator on sources that have projects. The fold state SHALL survive UI refreshes (including scan results arriving), and collapsing a source whose project was selected SHALL move the selection to the source itself.

#### Scenario: Collapsing a source
- **WHEN** the user activates the fold keybinding on an expanded source with projects
- **THEN** its project entries are hidden and the indicator shows the collapsed state

#### Scenario: Fold state survives incoming results
- **WHEN** a source is collapsed and another source's scan result arrives
- **THEN** the collapsed source remains collapsed after the panel refreshes

#### Scenario: Collapsing moves a hidden selection
- **WHEN** the user collapses a source while one of its projects is selected
- **THEN** the selection moves to the source node itself

### Requirement: Status messages can be hidden and shown
The system SHALL provide a keybinding that toggles the visibility of the transient status-message zone. Hiding SHALL never affect the key-help zone, and showing again SHALL restore the most recent message.

#### Scenario: Hiding messages
- **WHEN** the user presses the messages keybinding while a message is visible
- **THEN** the message zone clears and subsequent messages stay hidden, while the key help remains visible

#### Scenario: Showing messages restores the last one
- **WHEN** the user presses the messages keybinding again
- **THEN** the most recent message reappears in the message zone

### Requirement: User can rescan the selected source
The system SHALL provide a keybinding that rescans the currently selected source (the global source or a registered path). The rescan SHALL be disabled while that source's scan is in flight, informing the user instead of launching an overlapping scan.

#### Scenario: Rescanning an idle source
- **WHEN** the user presses the rescan keybinding on a source that is not scanning
- **THEN** that source returns to its loading state, is rescanned, and its results refresh when the scan completes

#### Scenario: Rescan blocked while scanning
- **WHEN** the user presses the rescan keybinding on a source whose scan is still running
- **THEN** no new scan starts and a message explains the rescan is disabled until the current scan finishes

### Requirement: User can manage paths from the UI
The system SHALL provide keybindings to add a new path (via a text input) and remove the selected path, persisting changes through the configuration store, and SHALL trigger a scan of a newly added path immediately.

#### Scenario: Adding a path from the UI
- **WHEN** the user adds a valid path through the add-path input
- **THEN** the path is persisted, appears in the sources panel, and its scan starts immediately

#### Scenario: Removing a path from the UI
- **WHEN** the user removes the selected registered path
- **THEN** the path is deleted from the configuration and its entry disappears from the panel

