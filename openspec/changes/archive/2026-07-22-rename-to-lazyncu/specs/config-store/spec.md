# config-store

## MODIFIED Requirements

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
