# dashboard-ui Specification (delta)

## ADDED Requirements

### Requirement: Packages table is focusable and navigable
The system SHALL toggle keyboard focus between the sources tree and the Packages table with `Tab`. While the table is focused, `↑↓` SHALL move the row cursor and `Esc` or `Tab` SHALL return focus to the tree. Focus toggling SHALL be ignored while the About modal or an input dialog is open.

#### Scenario: Entering and leaving the table
- **WHEN** the user presses `Tab` on the dashboard
- **THEN** the Packages table gains focus with a visible row cursor, and pressing `Esc` (or `Tab` again) returns focus to the sources tree

### Requirement: Packages can be marked for selective update
While the Packages table is focused, `space` SHALL toggle a mark on the row under the cursor and `x` SHALL clear all marks of the current project. Marked rows SHALL render a `✓` prefix with the package name highlighted. Marks SHALL be kept per project in memory, survive focus and project switches, never persist to configuration, and be cleared when the source is rescanned or its path removed.

#### Scenario: Marking and unmarking
- **WHEN** the user focuses the table, moves to `lodash`, and presses `space`
- **THEN** the row shows `✓ lodash` highlighted, and pressing `space` again removes the mark

#### Scenario: Clearing marks
- **WHEN** the current project has marked packages and the user presses `x` with the table focused
- **THEN** all marks of that project are removed and the command bar shows the full update command again

#### Scenario: Rescan clears the selection
- **WHEN** a source with marked packages is rescanned with `r`
- **THEN** its marks are cleared

### Requirement: Help bar reflects the focused panel
The help bar SHALL show a tree-focus variant (existing keys plus a `Tab pkgs` hint) when the sources tree is focused, and a table-focus variant (`q quit · ↑↓ move · ␣ mark · x clear · c copy cmd · Tab/Esc back`) when the Packages table is focused, using the existing valid-color-tag convention.

#### Scenario: Help follows focus
- **WHEN** the user presses `Tab` to focus the Packages table
- **THEN** the help bar switches to the table-focus keys, and switches back when focus returns to the tree
