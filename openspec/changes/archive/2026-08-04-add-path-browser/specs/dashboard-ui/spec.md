# dashboard-ui Delta Specification

## MODIFIED Requirements

### Requirement: User can manage paths from the UI
The system SHALL provide keybindings to add a new path and remove the selected path, persisting changes through the configuration store, and SHALL trigger a scan of a newly added path immediately. Removal SHALL require confirmation as defined by the destructive-action requirement.

The add-path modal SHALL be a hybrid picker: an editable text input on top and a directory tree below, rooted at the user's home directory. The input SHALL accept typed or pasted paths; confirming the input uses its text verbatim. Selecting a directory in the tree SHALL rewrite the input text to that directory's path. `Tab` SHALL move focus between input and tree.

The tree SHALL list directories only, reading children lazily when a node expands: `→` expands, `←` collapses, `Enter` confirms the selected directory, `Esc` cancels the modal. Dot-prefixed directories SHALL be hidden by default and toggled with `.`. A directory that cannot be read SHALL degrade gracefully (shown without children), never crashing or aborting the modal.

#### Scenario: Adding a path from the UI
- **WHEN** the user adds a valid path through the add-path modal (typed or picked from the tree)
- **THEN** the path is persisted, appears in the sources panel, and its scan starts immediately

#### Scenario: Picking a directory from the tree
- **WHEN** the user navigates the tree with `→`/`←`, highlights a directory, and presses `Enter`
- **THEN** the modal closes and that directory is registered and scanned

#### Scenario: Pasting a path into the input
- **WHEN** the user pastes a path into the input field and presses `Enter`
- **THEN** the typed text is used verbatim, validated by the config store, and registered on success

#### Scenario: Tree selection updates the input
- **WHEN** the user moves the tree selection to a directory
- **THEN** the input field shows that directory's full path

#### Scenario: Toggling hidden directories
- **WHEN** the user presses `.` while the tree is focused
- **THEN** dot-prefixed directories appear (or disappear) in expanded nodes

#### Scenario: Unreadable directory degrades gracefully
- **WHEN** the user expands a directory the process cannot read
- **THEN** the node shows no children and the modal keeps working

#### Scenario: Removing a path from the UI
- **WHEN** the user removes the selected registered path and confirms the removal
- **THEN** the path is deleted from the configuration and its entry disappears from the panel
