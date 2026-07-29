# dashboard-ui Delta: add-discoverability

## ADDED Requirements

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

## MODIFIED Requirements

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
