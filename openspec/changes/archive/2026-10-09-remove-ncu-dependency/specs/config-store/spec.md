# Spec Delta

## ADDED Requirements

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
