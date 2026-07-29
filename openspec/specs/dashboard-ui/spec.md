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
The system SHALL color package rows by severity — red for major, yellow for minor, green for patch, gray for other — and SHALL display severity counters in the sources panel using shape glyphs (`▲` major, `●` minor, `▪` patch, e.g. "▲3 ●5 ▪2") so that semver counters and audit letter counters (`C`/`H`/`M`/`L`) never share a glyph on the same line. Every source node SHALL display aggregate counters summed across its projects: semver counters over all projects, audit counters over successfully audited projects, with a `✗` marker when any project's audit failed, `0 vulns` when audited and clean, and `audit n/a` when no project produced a usable audit. Aggregates SHALL remain visible while the source is collapsed.

#### Scenario: Row colors
- **WHEN** the package table renders a major, a minor, and a patch upgrade
- **THEN** the rows are colored red, yellow, and green respectively

#### Scenario: Panel counters
- **WHEN** a project has 3 major, 5 minor, and 2 patch upgrades
- **THEN** its entry in the sources panel displays the counters "▲3 ●5 ▪2" colored red, yellow, and green respectively

#### Scenario: No glyph collision with audit counters
- **WHEN** a project row shows both semver counters and audit counters
- **THEN** the semver side uses only shapes (`▲ ● ▪`) and the audit side uses only letters (`C H M L`)

#### Scenario: Source node aggregates its projects
- **WHEN** a registered source has two projects with 1 and 2 major updates and audits reporting 1 high and 2 high vulnerabilities
- **THEN** the source node displays "▲3" and "H3" alongside its name

#### Scenario: Aggregate marks failed audits
- **WHEN** a registered source has one project whose audit failed and another with a clean audit
- **THEN** the source node's audit part carries a `✗` marker

#### Scenario: Folding keeps the aggregate visible
- **WHEN** the user collapses a source with pending updates
- **THEN** the collapsed source row still shows its aggregate counters

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

### Requirement: About modal shows version and app info
The system SHALL open a centered About modal when the user presses `h` on the main dashboard, showing app name, version, commit, build date, repository URL, and license. The modal SHALL close on `Esc` or `h`, restoring focus to the dashboard. The global `q` quit binding SHALL keep working while the modal is open. Pressing `h` while an input dialog is active SHALL do nothing. The counter legend lives in the cheat-sheet modal, not in About.

#### Scenario: Opening the About modal
- **WHEN** the user presses `h` on the main dashboard
- **THEN** a centered modal shows name, version, commit, build date, repo URL, and license

#### Scenario: Closing the About modal
- **WHEN** the About modal is open and the user presses `Esc` or `h`
- **THEN** the modal closes and focus returns to the dashboard

#### Scenario: Quit from the modal
- **WHEN** the About modal is open and the user presses `q`
- **THEN** the application quits

### Requirement: Help bar advertises the About modal
The bottom help bar SHALL include an `h about` hint rendered with the existing valid-color-tag convention (no bare bracket literals), and the help zone width SHALL be sized so the full help text is visible.

#### Scenario: Help bar shows the hint
- **WHEN** the dashboard renders
- **THEN** the help bar includes `h about` alongside the existing key hints, fully visible


### Requirement: Packages table is focusable and navigable
The system SHALL toggle keyboard focus between the sources tree and the Packages table with `Tab`. While the table is focused, `↑↓` SHALL move the row cursor and `Esc` or `Tab` SHALL return focus to the tree. Focus toggling SHALL be ignored while the About modal or an input dialog is open.

#### Scenario: Entering and leaving the table
- **WHEN** the user presses `Tab` on the dashboard
- **THEN** the Packages table gains focus with a visible row cursor, and pressing `Esc` (or `Tab` again) returns focus to the sources tree

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

### Requirement: Keybindings are scoped to the focused context
The system SHALL dispatch keybindings against the current UI context — `global`, `tree`, `table-packages`, `table-vulns`, or `modal` — from a single declarative keymap that also generates the contextual help bar and the cheat-sheet modal. A key SHALL only trigger its action in the contexts it is bound to: `q`, `c`, `v`, `m`, `h`, `r`, `?`, `1`, `2`, and `Tab` are global; `a`, `d`, and `Enter` (fold) are bound to the sources tree; `Space` and `x` are bound to the packages table view only (not the vulnerability view). Bindings MAY be hidden from the bottom help bar while still appearing in the cheat sheet. While a modal is open, all keys SHALL be inert except the modal's own close keys and, for the About and cheat-sheet modals only, `q` to quit; `q` SHALL be inert inside confirmation modals. Pressing a key that is bound in a different context SHALL show a transient hint naming where the key works (e.g. "d only works in Sources — Tab to go back") that auto-expires after approximately 5 seconds without affecting the key-help zone; keys bound nowhere SHALL be ignored silently.

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
- **THEN** every hint shown corresponds to a binding active in that context, and no active-context binding advertised in the keymap is missing from the full help variant unless it is marked bar-hidden

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

### Requirement: Keymap cheat sheet is available in-app
The system SHALL open a centered cheat-sheet modal when the user presses `?` on the main dashboard, generated from the declarative keymap and grouped by context (Global, Sources panel, Packages panel), including help-only bindings and bindings hidden from the bottom help bar, plus the counter legend explaining the severity glyphs (`▲` major, `●` minor, `▪` patch) and the audit letters (`C` critical, `H` high, `M` moderate, `L` low). The modal SHALL close on `Esc` or `?`, restoring focus to the dashboard; `q` SHALL quit while it is open; all other keys SHALL be inert. Pressing `?` while another modal or an input dialog is active SHALL do nothing.

#### Scenario: Opening the cheat sheet
- **WHEN** the user presses `?` on the main dashboard
- **THEN** a centered modal lists every advertised binding grouped by Global, Sources, and Packages, followed by the counter legend

#### Scenario: Cheat sheet cannot drift from behavior
- **WHEN** the cheat sheet renders
- **THEN** every entry corresponds to a row of the declarative keymap, including help-only rows and bar-hidden rows

#### Scenario: Closing the cheat sheet
- **WHEN** the cheat sheet is open and the user presses `Esc` or `?`
- **THEN** the modal closes and focus returns to the dashboard

#### Scenario: Quit from the cheat sheet
- **WHEN** the cheat sheet is open and the user presses `q`
- **THEN** the application quits

### Requirement: First run shows an empty-state call to action
When no paths are registered, the system SHALL show a hint in the sources tree ("No paths registered." / "Press a to add one.") as non-selectable entries under the global source, and the Packages panel SHALL show onboarding text explaining that `a` accepts a single project, a monorepo, or a folder of projects with automatic detection, plus a pointer to `?`. The onboarding text SHALL only replace empty package views: actual scan results and scan errors SHALL always take priority over it. The hints SHALL disappear once a path is registered.

#### Scenario: Empty first launch
- **WHEN** the app launches with no registered paths and the global scan finds nothing to update
- **THEN** the tree shows the add-a-path hint and the Packages panel shows the onboarding text instead of the up-to-date message

#### Scenario: Real data wins over onboarding
- **WHEN** no paths are registered but the global source has pending updates
- **THEN** the Packages panel shows the global package rows, not the onboarding text

#### Scenario: Hint disappears after adding a path
- **WHEN** the user registers a valid path with `a`
- **THEN** the tree hint entries are removed and the new source appears scanning

### Requirement: Focused panel is visually highlighted
The system SHALL render the keyboard-focused panel (sources tree or Packages table) with a highlighted border and title color, keeping the other panel on the default color. Panel titles SHALL be numbered (`1 Sources`, `2 Packages`), and the keys `1` and `2` SHALL move focus directly to the corresponding panel from any non-modal context. The `1`/`2` bindings SHALL be advertised in the cheat sheet but not in the bottom help bar.

#### Scenario: Focus follows Tab
- **WHEN** the user presses `Tab` to focus the Packages table
- **THEN** the table's border and title switch to the highlight color and the tree's revert to the default

#### Scenario: Direct panel addressing
- **WHEN** the user presses `2` while the tree is focused
- **THEN** the Packages table gains focus, and pressing `1` returns focus to the tree
