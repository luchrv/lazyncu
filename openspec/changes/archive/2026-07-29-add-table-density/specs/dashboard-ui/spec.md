# dashboard-ui Delta: add-table-density

## ADDED Requirements

### Requirement: Table rows can be sorted
The system SHALL let the user cycle the row order of the detail table with `s` — scan order → severity (most severe first) → name (alphabetical) — in both the packages view and the vulnerability view, keeping ties in scan order. The active order (other than scan order) SHALL be shown in the panel title. Sorting SHALL never mutate scan results and SHALL reset only on app restart.

#### Scenario: Severity sort surfaces majors
- **WHEN** the packages view shows patch, major, and minor rows in scan order and the user presses `s`
- **THEN** the rows reorder to major, minor, patch and the title shows the severity order

#### Scenario: Cycling back to scan order
- **WHEN** the user presses `s` until the cycle wraps
- **THEN** the rows return to scan order and the title shows no order indicator

#### Scenario: Marks follow the rows
- **WHEN** rows are sorted and the user marks the row under the cursor
- **THEN** the mark applies to the package actually displayed on that row

### Requirement: Table rows can be filtered incrementally
The system SHALL let the user filter the detail table with `/`, turning the status-message zone into an inline input: rows filter live by case-insensitive substring on the package name as the user types. `Enter` SHALL keep the filter active and focus the table; `Esc` in the input SHALL clear the filter and close the input. While the input has focus, dashboard keybindings SHALL be inactive. An active filter SHALL be shown in the panel title, and marks and generated commands SHALL be unaffected by row visibility.

#### Scenario: Live filtering
- **WHEN** the user presses `/` and types "axi"
- **THEN** only rows whose package name contains "axi" (case-insensitive) remain visible and the title shows the query

#### Scenario: Typing q while filtering
- **WHEN** the filter input has focus and the user types "q"
- **THEN** the app does not quit; "q" becomes part of the query

#### Scenario: Esc clears filter before leaving the table
- **WHEN** the table is focused with an active filter and the user presses `Esc`
- **THEN** the filter clears and the table stays focused; a second `Esc` returns focus to the tree

#### Scenario: Commands ignore visibility
- **WHEN** packages are marked and a filter hides some of them
- **THEN** the generated update command still reflects every mark

## MODIFIED Requirements

### Requirement: Dashboard layout separates sources and packages
The system SHALL render a terminal UI with a sources/projects panel (global source plus every registered path, with deep sources expanded into child project entries), a package table for the selected entry (columns: package, current version, new version, severity), and a command bar showing the suggested update command for the current selection. In detail tables, only identifying columns (package name, dependency chain) SHALL expand with available width; version, severity, range, and fix columns SHALL keep their natural width. A dependency chain longer than its column budget SHALL be shortened with a middle ellipsis, preserving both ends of the chain.

#### Scenario: Selecting a project
- **WHEN** the user selects a project entry in the sources panel
- **THEN** the package table shows that project's pending packages and the command bar shows its update command

#### Scenario: Selecting the global source
- **WHEN** the user selects the global source
- **THEN** the package table shows global packages and the command bar shows the `npm install -g ...` command

#### Scenario: Long dependency chain keeps its ends
- **WHEN** a vulnerability's dependency chain exceeds the Via column budget
- **THEN** the chain renders with a middle ellipsis keeping the first and last links visible

### Requirement: Severity is color-coded
The system SHALL color package rows by severity — red for major, yellow for minor, green for patch, gray for other — and SHALL display severity counters in the sources panel using shape glyphs (`▲` major, `●` minor, `▪` patch, e.g. "▲3 ●5 ▪2") so that semver counters and audit letter counters (`C`/`H`/`M`/`L`) never share a glyph on the same line. Every source node SHALL display aggregate counters summed across its projects: semver counters over all projects, audit counters over successfully audited projects, with a `✗` marker when any project's audit failed, `0 vulns` when audited and clean, and `audit n/a` when no project produced a usable audit. Aggregates SHALL remain visible while the source is collapsed. Severity cells in detail tables SHALL carry the matching glyph or letter prefix (`▲ major`, `C critical`), so severity is identifiable without color.

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

#### Scenario: Severity cells carry a glyph
- **WHEN** the packages view renders a major row and the vulnerability view renders a critical row
- **THEN** their severity cells read "▲ major" and "C critical" respectively

### Requirement: Keybindings are scoped to the focused context
The system SHALL dispatch keybindings against the current UI context — `global`, `tree`, `table-packages`, `table-vulns`, or `modal` — from a single declarative keymap that also generates the contextual help bar and the cheat-sheet modal. A key SHALL only trigger its action in the contexts it is bound to: `q`, `c`, `v`, `m`, `h`, `r`, `?`, `1`, `2`, and `Tab` are global; `a`, `d`, and `Enter` (fold) are bound to the sources tree; `Space` and `x` are bound to the packages table view only (not the vulnerability view); `s` (sort) and `/` (filter) are bound to both table views. `Esc` in the table SHALL clear an active filter first and return focus to the tree otherwise. Bindings MAY be hidden from the bottom help bar while still appearing in the cheat sheet. While a modal is open, all keys SHALL be inert except the modal's own close keys and, for the About and cheat-sheet modals only, `q` to quit; `q` SHALL be inert inside confirmation modals. Pressing a key that is bound in a different context SHALL show a transient hint naming where the key works (e.g. "d only works in Sources — Tab to go back") that auto-expires after approximately 5 seconds without affecting the key-help zone; keys bound nowhere SHALL be ignored silently.

#### Scenario: Tree-scoped key pressed in the table
- **WHEN** the Packages table is focused and the user presses `d`
- **THEN** no path is removed and a transient hint explains that `d` works in the Sources panel

#### Scenario: Table-scoped key pressed in the vulnerability view
- **WHEN** the table is focused showing vulnerabilities and the user presses `Space`
- **THEN** no mark is toggled and a transient hint explains that marking works in the packages view

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
