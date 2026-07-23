# dashboard-ui Specification (delta)

## ADDED Requirements

### Requirement: About modal shows version and app info
The system SHALL open a centered About modal when the user presses `h` on the main dashboard, showing app name, version, commit, build date, repository URL, and license. The modal SHALL close on `Esc` or `h`, restoring focus to the dashboard. The global `q` quit binding SHALL keep working while the modal is open. Pressing `h` while an input dialog is active SHALL do nothing.

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
