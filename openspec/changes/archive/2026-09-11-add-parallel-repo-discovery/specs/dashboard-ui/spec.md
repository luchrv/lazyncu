## MODIFIED Requirements

### Requirement: Loading and error states are shown per source
The system SHALL show an animated spinner as the loading indicator for each source while its scan is running — in the source's tree row and in the Packages-panel loading message — replace it with results when the scan completes, and show an error state when the scan fails: a compact badge in the tree row, and in the detail panel the full failure reason wrapped over multiple lines together with a "press r to retry" hint. For a folder source, the same treatment SHALL apply per repository entry: once discovery completes, every discovered repository SHALL appear as a child entry with its own spinner, the folder row SHALL show `done/total` next to its spinner while repositories are still scanning, each entry SHALL switch to its results independently, and a failed repository SHALL show the error badge on its own entry and the full reason plus retry hint in the detail panel while the folder row and its siblings render normally. The folder row SHALL summarize failed repositories with a count next to its aggregate. A folder that completes with zero repositories SHALL say so in its row instead of reporting itself up to date. The spinner animation SHALL run only while at least one source or repository is loading, and every frame update SHALL be applied through the serialized UI-update dispatch.

#### Scenario: Progressive loading
- **WHEN** the application launches and scans are in flight
- **THEN** each pending source displays an animated spinner that disappears independently as its scan finishes

#### Scenario: Spinner stops when idle
- **WHEN** every source has finished scanning
- **THEN** no spinner animation remains anywhere in the UI

#### Scenario: Failed source
- **WHEN** a source's scan fails
- **THEN** its tree entry shows an error badge, and the detail panel shows the complete failure reason and the retry hint

#### Scenario: Folder shows pending repositories
- **WHEN** a folder source's discovery finishes with 50 repositories and 12 have completed
- **THEN** the folder row reads `scanning 12/50` with a spinner, 12 child entries show results and 38 show a spinner

#### Scenario: Failed repository inside a folder
- **WHEN** one repository's scan fails inside a folder of 50
- **THEN** that entry shows the error badge, selecting it shows the full reason and the retry hint, the folder row shows `1 failed` next to its aggregate, and the other 49 entries render normally

#### Scenario: Empty folder
- **WHEN** a folder source completes with no discovered repository
- **THEN** its row reads `no projects found` and it has no child entries

### Requirement: User can rescan the selected source
The system SHALL provide a keybinding that rescans the currently selected source (the global source or a registered path). When the selection is a repository entry inside a folder source, the same keybinding SHALL rescan only that repository, replacing its entry in place while the rest of the folder stays untouched. The rescan SHALL be disabled while the selected source or repository is in flight, informing the user instead of launching an overlapping scan. When the unit being rescanned has marked packages, the rescan SHALL first require confirmation as defined by the destructive-action requirement.

#### Scenario: Rescanning an idle source
- **WHEN** the user presses the rescan keybinding on a source that is not scanning and has no marks
- **THEN** that source returns to its loading state, is rescanned, and its results refresh when the scan completes

#### Scenario: Rescan blocked while scanning
- **WHEN** the user presses the rescan keybinding on a source whose scan is still running
- **THEN** no new scan starts and a message explains the rescan is disabled until the current scan finishes

#### Scenario: Rescanning a single repository
- **WHEN** the user presses the rescan keybinding on a failed repository entry inside a folder
- **THEN** only that entry returns to its loading state and is rescanned, and its siblings keep their results

#### Scenario: Rescanning the folder row
- **WHEN** the user presses the rescan keybinding on the folder source row
- **THEN** the folder is rediscovered and every repository is rescanned
