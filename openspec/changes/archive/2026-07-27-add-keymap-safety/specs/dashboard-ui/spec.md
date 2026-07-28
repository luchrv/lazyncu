# dashboard-ui Delta: add-keymap-safety

## ADDED Requirements

### Requirement: Keybindings are scoped to the focused context
The system SHALL dispatch keybindings against the current UI context — `global`, `tree`, `table-packages`, `table-vulns`, or `modal` — from a single declarative keymap that also generates the contextual help bar. A key SHALL only trigger its action in the contexts it is bound to: `q`, `c`, `v`, `m`, `h`, `r`, and `Tab` are global; `a`, `d`, and `Enter` (fold) are bound to the sources tree; `Space` and `x` are bound to the packages table view only (not the vulnerability view). While a modal is open, all keys SHALL be inert except the modal's own close keys and, for the About modal only, `q` to quit; `q` SHALL be inert inside confirmation modals. Pressing a key that is bound in a different context SHALL show a transient hint naming where the key works (e.g. "d only works in Sources — Tab to go back") that auto-expires after approximately 5 seconds without affecting the key-help zone; keys bound nowhere SHALL be ignored silently.

#### Scenario: Tree-scoped key pressed in the table
- **WHEN** the Packages table is focused and the user presses `d`
- **THEN** no path is removed and a transient hint explains that `d` works in the Sources panel

#### Scenario: Table-scoped key pressed in the vulnerability view
- **WHEN** the table is focused showing vulnerabilities and the user presses `Space`
- **THEN** no mark is toggled and a transient hint explains that marking works in the packages view

#### Scenario: Keys are inert while a modal is open
- **WHEN** the About modal is open and the user presses `a` or `d`
- **THEN** no add-path input opens, no path is removed, and the modal stays on top

#### Scenario: Hint expires automatically
- **WHEN** an out-of-context hint has been shown and roughly 5 seconds pass without newer messages
- **THEN** the hint clears itself, and a status message issued after the hint is never cleared by the hint's timer

#### Scenario: Help bar is generated from the keymap
- **WHEN** the dashboard renders the help bar for any context
- **THEN** every hint shown corresponds to a binding active in that context, and no active-context binding advertised in the keymap is missing from the full help variant

### Requirement: Destructive actions require confirmation
The system SHALL show a centered confirmation modal before executing a destructive action, with Cancel as the default-focused button, `Esc` cancelling, and mouse-clickable buttons. Removing a registered path SHALL always ask, stating that the folder on disk is not touched. Rescanning a source SHALL ask only when the source has marked packages in any of its projects, stating the aggregate number of marks and projects that would be discarded; with no marks the rescan SHALL start immediately.

#### Scenario: Removing a path asks first
- **WHEN** the user presses `d` on a registered path in the sources tree
- **THEN** a confirmation modal appears with Cancel focused, and the path is removed only after the user confirms

#### Scenario: Cancelling keeps the path
- **WHEN** the removal confirmation is open and the user presses `Esc` or activates Cancel
- **THEN** the modal closes and the path remains registered and visible

#### Scenario: Rescan with marks warns about the loss
- **WHEN** the user presses `r` on a source whose projects hold 8 marks across 3 projects
- **THEN** a confirmation modal states that rescanning discards 8 marks across 3 projects, and the rescan starts only after confirmation

#### Scenario: Rescan without marks is immediate
- **WHEN** the user presses `r` on a source with no marked packages
- **THEN** the rescan starts immediately without any confirmation modal

## MODIFIED Requirements

### Requirement: Severity is color-coded
The system SHALL color package rows by severity — red for major, yellow for minor, green for patch, gray for other — and SHALL display per-project severity counters in the sources panel using shape glyphs (`▲` major, `●` minor, `▪` patch, e.g. "▲3 ●5 ▪2") so that semver counters and audit letter counters (`C`/`H`/`M`/`L`) never share a glyph on the same line.

#### Scenario: Row colors
- **WHEN** the package table renders a major, a minor, and a patch upgrade
- **THEN** the rows are colored red, yellow, and green respectively

#### Scenario: Panel counters
- **WHEN** a project has 3 major, 5 minor, and 2 patch upgrades
- **THEN** its entry in the sources panel displays the counters "▲3 ●5 ▪2" colored red, yellow, and green respectively

#### Scenario: No glyph collision with audit counters
- **WHEN** a project row shows both semver counters and audit counters
- **THEN** the semver side uses only shapes (`▲ ● ▪`) and the audit side uses only letters (`C H M L`)

### Requirement: User can rescan the selected source
The system SHALL provide a keybinding that rescans the currently selected source (the global source or a registered path). The rescan SHALL be disabled while that source's scan is in flight, informing the user instead of launching an overlapping scan. When the source has marked packages in any of its projects, the rescan SHALL first require confirmation as defined by the destructive-action requirement.

#### Scenario: Rescanning an idle source
- **WHEN** the user presses the rescan keybinding on a source that is not scanning and has no marks
- **THEN** that source returns to its loading state, is rescanned, and its results refresh when the scan completes

#### Scenario: Rescan blocked while scanning
- **WHEN** the user presses the rescan keybinding on a source whose scan is still running
- **THEN** no new scan starts and a message explains the rescan is disabled until the current scan finishes

### Requirement: User can manage paths from the UI
The system SHALL provide keybindings to add a new path (via a text input) and remove the selected path, persisting changes through the configuration store, and SHALL trigger a scan of a newly added path immediately. Removal SHALL require confirmation as defined by the destructive-action requirement.

#### Scenario: Adding a path from the UI
- **WHEN** the user adds a valid path through the add-path input
- **THEN** the path is persisted, appears in the sources panel, and its scan starts immediately

#### Scenario: Removing a path from the UI
- **WHEN** the user removes the selected registered path and confirms the removal
- **THEN** the path is deleted from the configuration and its entry disappears from the panel

### Requirement: Packages can be marked for selective update
While the Packages table is focused on the packages view, `space` SHALL toggle a mark on the row under the cursor and `x` SHALL clear all marks of the current project. Marked rows SHALL render a `✓` prefix with the package name highlighted. Marks SHALL be kept per project in memory, survive focus and project switches, never persist to configuration, and be cleared when the source is rescanned (after the user confirms the rescan) or its path removed.

#### Scenario: Marking and unmarking
- **WHEN** the user focuses the table, moves to `lodash`, and presses `space`
- **THEN** the row shows `✓ lodash` highlighted, and pressing `space` again removes the mark

#### Scenario: Clearing marks
- **WHEN** the current project has marked packages and the user presses `x` with the table focused
- **THEN** all marks of that project are removed and the command bar shows the full update command again

#### Scenario: Confirmed rescan clears the selection
- **WHEN** a source with marked packages is rescanned with `r` and the user confirms the mark-loss warning
- **THEN** its marks are cleared and the rescan starts

### Requirement: Help bar reflects the focused panel
The help bar SHALL be generated from the declarative keymap for the current context — a tree variant when the sources tree is focused and a table variant when the Packages table is focused — using the existing valid-color-tag convention. The system SHALL choose between a full and a compact help variant based on the terminal width so that the help text is never clipped and the status-message zone always retains space.

#### Scenario: Help follows focus
- **WHEN** the user presses `Tab` to focus the Packages table
- **THEN** the help bar switches to the table-context bindings generated from the keymap, and switches back when focus returns to the tree

#### Scenario: Narrow terminal gets compact help
- **WHEN** the terminal is too narrow for the full help variant
- **THEN** the help bar shows the compact variant and the status-message zone keeps a non-zero width

### Requirement: About modal shows version and app info
The system SHALL open a centered About modal when the user presses `h` on the main dashboard, showing app name, version, commit, build date, repository URL, license, and a counter legend explaining the severity glyphs (`▲` major, `●` minor, `▪` patch) and the audit letters (`C` critical, `H` high, `M` moderate, `L` low). The modal SHALL close on `Esc` or `h`, restoring focus to the dashboard. The global `q` quit binding SHALL keep working while the modal is open. Pressing `h` while an input dialog is active SHALL do nothing.

#### Scenario: Opening the About modal
- **WHEN** the user presses `h` on the main dashboard
- **THEN** a centered modal shows name, version, commit, build date, repo URL, license, and the counter legend

#### Scenario: Closing the About modal
- **WHEN** the About modal is open and the user presses `Esc` or `h`
- **THEN** the modal closes and focus returns to the dashboard

#### Scenario: Quit from the modal
- **WHEN** the About modal is open and the user presses `q`
- **THEN** the application quits
