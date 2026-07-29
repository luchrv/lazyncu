# dashboard-ui Delta: add-flow-polish

## ADDED Requirements

### Requirement: All sources can be rescanned at once
The system SHALL provide a global keybinding (`R`) that rescans every source whose scan is not already in flight; in-flight sources SHALL be skipped by the existing overlap guard. When any source holds marked packages, the system SHALL ask for confirmation once, stating the aggregate marks and projects that would be discarded across all sources; with no marks the sweep SHALL start immediately. A status message SHALL report how many sources were launched, and pressing `R` while every source is already scanning SHALL produce a warning instead of a no-op.

#### Scenario: Sweep without marks
- **WHEN** the user presses `R` with three idle sources and no marks
- **THEN** all three return to their loading state and a message reports the rescan

#### Scenario: Sweep with marks asks once
- **WHEN** two sources hold 5 marks across 2 projects in total and the user presses `R`
- **THEN** a single confirmation states that 5 marks across 2 projects would be discarded, and the sweep starts only after confirming

#### Scenario: In-flight sources are skipped
- **WHEN** one source is still scanning and the user presses `R`
- **THEN** only the idle sources rescan and the scanning source continues undisturbed

### Requirement: Command bar sizes to its content and truncates loudly
The command bar SHALL grow to fit the visible commands, between one and four content lines; a command needing more than its share SHALL be truncated with an explicit `… (c copies full)` indicator. Copying SHALL always place the complete, untruncated command on the clipboard.

#### Scenario: Short commands get a compact bar
- **WHEN** the selection's update command fits one line and there is no fix command
- **THEN** the command bar shrinks to a single content line

#### Scenario: Overflow is explicit
- **WHEN** a filtered update command exceeds the command bar's budget
- **THEN** the visible text ends with the truncation indicator and pressing `c` copies the full command

## MODIFIED Requirements

### Requirement: Packages can be marked for selective update
While the Packages table is focused on the packages view, `space` SHALL toggle a mark on the row under the cursor and `x` SHALL clear all marks of the current project. Marked rows SHALL render a `✓` prefix with the package name highlighted, and the panel title SHALL show ` · N/M marked` (marked over total packages of the entry) while at least one package is marked. Marks SHALL be kept per project in memory, survive focus and project switches, never persist to configuration, and be cleared when the source is rescanned (after the user confirms the rescan) or its path removed.

#### Scenario: Marking and unmarking
- **WHEN** the user focuses the table, moves to `lodash`, and presses `space`
- **THEN** the row shows `✓ lodash` highlighted, and pressing `space` again removes the mark

#### Scenario: Title counts the marks
- **WHEN** 2 of the project's 6 packages are marked
- **THEN** the Packages title shows "2/6 marked", and the indicator disappears when the last mark is cleared

#### Scenario: Clearing marks
- **WHEN** the current project has marked packages and the user presses `x` with the table focused
- **THEN** all marks of that project are removed and the command bar shows the full update command again

#### Scenario: Confirmed rescan clears the selection
- **WHEN** a source with marked packages is rescanned with `r` and the user confirms the mark-loss warning
- **THEN** its marks are cleared and the rescan starts

### Requirement: Loading and error states are shown per source
The system SHALL show an animated spinner as the loading indicator for each source while its scan is running — in the source's tree row and in the Packages-panel loading message — replace it with results when the scan completes, and show an error state when the scan fails: a compact badge in the tree row, and in the detail panel the full failure reason wrapped over multiple lines together with a "press r to retry" hint. The spinner animation SHALL run only while at least one source is loading, and every frame update SHALL be applied through the serialized UI-update dispatch.

#### Scenario: Progressive loading
- **WHEN** the application launches and scans are in flight
- **THEN** each pending source displays an animated spinner that disappears independently as its scan finishes

#### Scenario: Spinner stops when idle
- **WHEN** every source has finished scanning
- **THEN** no spinner animation remains anywhere in the UI

#### Scenario: Failed source
- **WHEN** a source's scan fails
- **THEN** its tree entry shows an error badge, and the detail panel shows the complete failure reason and the retry hint

### Requirement: Keybindings are scoped to the focused context
The system SHALL dispatch keybindings against the current UI context — `global`, `tree`, `table-packages`, `table-vulns`, or `modal` — from a single declarative keymap that also generates the contextual help bar and the cheat-sheet modal. A key SHALL only trigger its action in the contexts it is bound to: `q`, `c`, `v`, `m`, `h`, `r`, `R`, `?`, `1`, `2`, and `Tab` are global; `a`, `d`, and `Enter` (fold) are bound to the sources tree; `Space` and `x` are bound to the packages table view only (not the vulnerability view); `s` (sort) and `/` (filter) are bound to both table views. `Esc` SHALL peel one layer in every context: in the table it clears an active filter first and returns focus to the tree otherwise; in the tree it collapses the selected source (or the parent of a selected project) and SHALL show an informational message when nothing is foldable. Bindings MAY be hidden from the bottom help bar while still appearing in the cheat sheet. While a modal is open, all keys SHALL be inert except the modal's own close keys and, for the About and cheat-sheet modals only, `q` to quit; `q` SHALL be inert inside confirmation modals. Pressing a key that is bound in a different context SHALL show a transient hint naming where the key works (e.g. "d only works in Sources — Tab to go back") that auto-expires after approximately 5 seconds without affecting the key-help zone; keys bound nowhere SHALL be ignored silently.

#### Scenario: Tree-scoped key pressed in the table
- **WHEN** the Packages table is focused and the user presses `d`
- **THEN** no path is removed and a transient hint explains that `d` works in the Sources panel

#### Scenario: Table-scoped key pressed in the vulnerability view
- **WHEN** the table is focused showing vulnerabilities and the user presses `Space`
- **THEN** no mark is toggled and a transient hint explains that marking works in the packages view

#### Scenario: Esc folds the selected source
- **WHEN** the tree is focused on an expanded source (or one of its projects) and the user presses `Esc`
- **THEN** the source collapses, and pressing `Esc` on something not foldable shows an informational message

#### Scenario: Sort works in both table views
- **WHEN** the table is focused showing vulnerabilities and the user presses `s`
- **THEN** the vulnerability rows reorder by the next sort mode

#### Scenario: Keys are inert while a modal is open
- **WHEN** the About modal is open and the user presses `a` or `d`
- **THEN** no add-path input opens, no path is removed, and the modal stays on top

#### Scenario: Hint expires automatically
- **WHEN** an out-of-context hint has been shown and roughly 5 seconds pass without newer messages
- **THEN** the hint clears itself, and a status message issued after the hint is never cleared by the hint's timer

#### Scenario: Help bar is generated from the keymap
- **WHEN** the dashboard renders the help bar for any context
- **THEN** every hint shown corresponds to a binding active in that context, and no active-context binding advertised in the keymap is missing from the full help variant unless it is marked bar-hidden
