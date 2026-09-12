## ADDED Requirements

### Requirement: Concurrency bound is configurable
The system SHALL read an optional `max_parallel` setting from the config file and use it as the maximum number of external commands running at the same time across all sources, defaulting to 4 when absent. A value below 1 SHALL be treated as absent. The setting SHALL be optional in the file: existing config files without it keep working unchanged.

#### Scenario: Custom bound configured
- **WHEN** the config file sets `max_parallel = 8`
- **THEN** at most 8 external commands run at the same time

#### Scenario: No bound configured
- **WHEN** the config file has no `max_parallel` entry
- **THEN** at most 4 external commands run at the same time

#### Scenario: Invalid bound falls back to default
- **WHEN** the config file sets `max_parallel = 0` or a negative value
- **THEN** the default of 4 is used

#### Scenario: First launch writes the default bound
- **WHEN** the config file is created on first launch
- **THEN** it contains `max_parallel = 4` next to `timeout_ms`, so the setting is discoverable by hand-editing
