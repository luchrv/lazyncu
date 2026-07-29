# dashboard-ui Delta: add-scan-feedback

## ADDED Requirements

### Requirement: Aggregate scan progress is shown while scanning
The system SHALL display an aggregate progress indicator (`scanning N/M`, where N is the number of finished sources and M the total) in a dedicated bottom-bar segment between the status-message zone and the key help while at least one source is scanning, and SHALL hide it when no scan is in flight. Transient status messages SHALL never cover the progress indicator.

#### Scenario: Progress during the launch scan
- **WHEN** the app launches with 4 registered paths and 2 sources have finished scanning
- **THEN** the bottom bar shows "scanning 2/5" (the global source counts toward the total)

#### Scenario: Progress disappears on completion
- **WHEN** the last in-flight scan finishes
- **THEN** the progress segment is removed from the bottom bar

#### Scenario: Messages do not cover progress
- **WHEN** the user copies a command while scans are in flight
- **THEN** both the copy confirmation and the progress indicator are visible simultaneously

### Requirement: Status messages carry a severity level and expire
The system SHALL render every status message with a severity level — info (`·`, gray), ok (`✓`, green), warn (`!`, yellow), error (`✗`, red) — deriving the icon and color from the level in a single place, so severity survives monochrome terminals. Messages of level info, ok, and warn SHALL auto-expire after approximately 5 seconds unless a newer message replaced them; error messages SHALL persist until replaced. Out-of-context key hints SHALL use this same mechanism at the warn level.

#### Scenario: Confirmation expires
- **WHEN** the user copies a command and roughly 5 seconds pass without newer messages
- **THEN** the copy confirmation clears itself

#### Scenario: Errors persist
- **WHEN** a clipboard error is shown and roughly 5 seconds pass
- **THEN** the error message remains visible until a newer message replaces it

#### Scenario: Expiry never clears a newer message
- **WHEN** a message is shown and a newer message replaces it before the first one's expiry fires
- **THEN** the newer message stays visible when the first message's timer fires

#### Scenario: Severity is visible without color
- **WHEN** any status message renders
- **THEN** its level is identifiable by the icon prefix alone (`·`, `✓`, `!`, `✗`)

## MODIFIED Requirements

### Requirement: Loading and error states are shown per source
The system SHALL show an animated spinner as the loading indicator for each source while its scan is running — in the source's tree row and in the Packages-panel loading message — replace it with results when the scan completes, and show an error state with the failure reason when the scan fails. The spinner animation SHALL run only while at least one source is loading, and every frame update SHALL be applied through the serialized UI-update dispatch.

#### Scenario: Progressive loading
- **WHEN** the application launches and scans are in flight
- **THEN** each pending source displays an animated spinner that disappears independently as its scan finishes

#### Scenario: Spinner stops when idle
- **WHEN** every source has finished scanning
- **THEN** no spinner animation remains anywhere in the UI

#### Scenario: Failed source
- **WHEN** a source's scan fails
- **THEN** its entry shows an error indicator and the failure reason is visible to the user

### Requirement: Status messages can be hidden and shown
The system SHALL provide a keybinding that toggles the visibility of the transient status-message zone. Hiding SHALL never affect the key-help zone or the scan-progress segment, and showing again SHALL restore the most recent message only if it has not expired — an expired message SHALL NOT resurrect.

#### Scenario: Hiding messages
- **WHEN** the user presses the messages keybinding while a message is visible
- **THEN** the message zone clears and subsequent messages stay hidden, while the key help remains visible

#### Scenario: Showing messages restores the last live one
- **WHEN** the user presses the messages keybinding again before the last message expired
- **THEN** the most recent message reappears in the message zone

#### Scenario: Expired messages do not resurrect
- **WHEN** the user hides messages, the last message expires, and the user shows messages again
- **THEN** the message zone stays empty
