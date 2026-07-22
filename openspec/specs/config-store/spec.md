# config-store Specification

## Purpose
TBD - created by archiving change add-ncu-tui-dashboard. Update Purpose after archive.
## Requirements
### Requirement: Configuration is persisted as a TOML file
The system SHALL persist user configuration in a TOML file located at `$XDG_CONFIG_HOME/ncu-tui/config.toml`, falling back to `~/.config/ncu-tui/config.toml` when `XDG_CONFIG_HOME` is unset.

#### Scenario: First launch with no config file
- **WHEN** the application starts and no config file exists
- **THEN** the application creates the config directory and an empty config file, and starts with zero registered paths

#### Scenario: Config file is loaded on startup
- **WHEN** the application starts and a valid config file exists
- **THEN** all registered paths and settings are loaded from it

#### Scenario: Malformed config file
- **WHEN** the config file exists but contains invalid TOML
- **THEN** the application reports a clear error message identifying the file path and does not overwrite the file

### Requirement: User can register project paths
The system SHALL allow the user to add a filesystem path to the registered paths list, expanding a leading `~` to the user's home directory, and persist the change immediately.

#### Scenario: Adding a valid path
- **WHEN** the user adds a path that exists on the filesystem
- **THEN** the path is appended to the config file and appears as a source in the dashboard

#### Scenario: Adding a non-existent path
- **WHEN** the user adds a path that does not exist
- **THEN** the system rejects it with an error message and the config file is not modified

#### Scenario: Adding a duplicate path
- **WHEN** the user adds a path that is already registered (after tilde expansion and path cleaning)
- **THEN** the system rejects it as a duplicate and the config file is not modified

### Requirement: User can remove registered paths
The system SHALL allow the user to remove a previously registered path and persist the change immediately.

#### Scenario: Removing a registered path
- **WHEN** the user removes a registered path
- **THEN** the path is deleted from the config file and its source disappears from the dashboard

### Requirement: Scan timeout is configurable
The system SHALL read an optional `timeout_ms` setting from the config file and use it as the ncu scan timeout, defaulting to 30000 ms when absent.

#### Scenario: Custom timeout configured
- **WHEN** the config file sets `timeout_ms = 60000`
- **THEN** scans are executed with a 60-second timeout

#### Scenario: No timeout configured
- **WHEN** the config file has no `timeout_ms` entry
- **THEN** scans are executed with the 30000 ms default

