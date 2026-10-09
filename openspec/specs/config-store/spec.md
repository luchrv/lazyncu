# config-store Specification

## Purpose
Persist user settings — registered project paths, the per-command scan timeout and the concurrency bound — as a TOML file under the XDG config directory, created on first launch and editable both from the UI and by hand.

## Requirements

### Requirement: Configuration is persisted as a TOML file
The system SHALL persist user configuration in a TOML file located at `$XDG_CONFIG_HOME/lazyncu/config.toml`, falling back to `~/.config/lazyncu/config.toml` when `XDG_CONFIG_HOME` is unset. A config directory left over from the application's previous name (`ncu-tui`) SHALL NOT be read or migrated.

#### Scenario: First launch with no config file
- **WHEN** the application starts and no config file exists
- **THEN** the application creates the config directory and an empty config file, and starts with zero registered paths

#### Scenario: Config file is loaded on startup
- **WHEN** the application starts and a valid config file exists
- **THEN** all registered paths and settings are loaded from it

#### Scenario: Malformed config file
- **WHEN** the config file exists but contains invalid TOML
- **THEN** the application reports a clear error message identifying the file path and does not overwrite the file

#### Scenario: Old config directory is ignored
- **WHEN** the application starts with a legacy `~/.config/ncu-tui/config.toml` present and no `~/.config/lazyncu/config.toml`
- **THEN** a fresh empty config is created under `lazyncu` and the legacy file is neither read nor modified

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

### Requirement: Registry request bound is configurable
The system SHALL read an optional `max_requests` setting from the config file and use it as the maximum number of registry HTTP requests in flight across all sources, defaulting to 32 when absent. A value below 1 SHALL be treated as absent. Existing config files without it keep working unchanged.

#### Scenario: Custom request bound
- **WHEN** the config file sets `max_requests = 8`
- **THEN** at most 8 registry requests are in flight at the same time

#### Scenario: Missing request bound
- **WHEN** the config file has no `max_requests` entry
- **THEN** the bound is 32 and the file is left unchanged

### Requirement: Cache lifetime is configurable
The system SHALL read an optional `cache_ttl` setting from the config file as a Go duration string (`"1h"`, `"30m"`) and use it as the lifetime of persisted registry metadata and persisted audit results, defaulting to `"1h"` when absent; `"0"` SHALL disable on-disk caching for both. A value that is not a duration SHALL be reported as a configuration error naming the key, and the application SHALL NOT start.

#### Scenario: Custom lifetime
- **WHEN** the config file sets `cache_ttl = "30m"`
- **THEN** persisted entries older than 30 minutes are fetched again

#### Scenario: Cache disabled
- **WHEN** the config file sets `cache_ttl = "0"`
- **THEN** neither `registry-cache.json` nor `audit-cache.json` is written and every launch queries the registry and audits again

#### Scenario: Invalid lifetime
- **WHEN** the config file sets `cache_ttl = "soon"`
- **THEN** the application exits with an error that names `cache_ttl` and the file path
