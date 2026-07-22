# semver-analysis Specification

## Purpose
TBD - created by archiving change add-ncu-tui-dashboard. Update Purpose after archive.
## Requirements
### Requirement: Upgrade severity is classified per package
The system SHALL classify each package's upgrade as `major`, `minor`, or `patch` by comparing the current and new semver versions, and SHALL classify as `other` any pair that cannot be parsed as semver (git URLs, tags, wildcards) or whose current version is unknown.

#### Scenario: Major upgrade
- **WHEN** a package upgrades from 2.4.1 to 3.0.0
- **THEN** its severity is `major`

#### Scenario: Minor upgrade
- **WHEN** a package upgrades from 2.4.1 to 2.5.0
- **THEN** its severity is `minor`

#### Scenario: Patch upgrade
- **WHEN** a package upgrades from 2.4.1 to 2.4.9
- **THEN** its severity is `patch`

#### Scenario: Range prefixes are tolerated
- **WHEN** a package upgrades from ^2.4.1 to ^3.0.0
- **THEN** the prefixes are stripped and severity is `major`

#### Scenario: Non-semver version
- **WHEN** a package's current version is a git URL or dist-tag
- **THEN** its severity is `other` and no error is raised

### Requirement: Per-project severity counters are aggregated
The system SHALL aggregate, for each project entry, the count of pending packages per severity (major, minor, patch), excluding `other` from the counters.

#### Scenario: Mixed severities
- **WHEN** a project has 3 major, 5 minor, and 2 patch upgrades pending
- **THEN** its counters report exactly 3 major, 5 minor, 2 patch

#### Scenario: Up-to-date project
- **WHEN** a project has no pending upgrades
- **THEN** all its counters are zero

