# cli-path-argument Specification

## Purpose
Let users launch lazyncu with an optional positional path (`lazyncu .`, `lazyncu <path>`) that registers the target directory — or, when it is already covered by a registered path, positions the cursor on the matching source or project entry — with validation before the TUI opens.
## Requirements
### Requirement: CLI accepts zero or one positional path argument

The system SHALL accept at most one positional path argument on launch. With no argument, behavior SHALL be identical to today: the TUI opens with the configured sources and the current working directory is never registered implicitly. With more than one positional argument, the system SHALL print a usage error to stderr and exit non-zero without opening the TUI. The `--version` flag SHALL keep precedence as the first argument.

#### Scenario: No argument keeps current behavior
- **WHEN** the user runs `lazyncu` with no arguments from any directory
- **THEN** the TUI opens with the configured sources and no path is added to the configuration

#### Scenario: Too many arguments
- **WHEN** the user runs `lazyncu path1 path2`
- **THEN** a usage error is printed to stderr and the process exits with a non-zero code without opening the TUI

#### Scenario: Version flag still wins
- **WHEN** the user runs `lazyncu --version`
- **THEN** the version line is printed and the process exits 0, exactly as before

### Requirement: Positional path is resolved against the working directory

The system SHALL resolve the positional argument to an absolute, cleaned path: `.` and relative paths resolve against the current working directory, a leading `~` expands to the user's home directory, and absolute paths are used as-is. Comparisons against registered paths SHALL be performed on symlink-resolved forms, without rewriting the paths persisted in the configuration file.

#### Scenario: Dot resolves to the working directory
- **WHEN** the user runs `lazyncu .` from `/projects/api`
- **THEN** the target path is `/projects/api`

#### Scenario: Relative and tilde forms resolve
- **WHEN** the user runs `lazyncu ../web` from `/projects/api` or `lazyncu ~/projects/web`
- **THEN** the target path is `/projects/web` or `<home>/projects/web` respectively

#### Scenario: Symlinked target matches its real registered path
- **WHEN** the user runs `lazyncu /link/api` where `/link/api` is a symlink to the registered path `/projects/api`
- **THEN** the target is treated as equal to the registered path and no duplicate is added

### Requirement: Target is validated as a Node directory before the TUI opens

The system SHALL reject the target before opening the TUI when it does not exist, is not a directory, or is not a Node target — defined as containing a `package.json` at its root or in at least one subdirectory within a bounded search depth, excluding `node_modules` directories. Rejection SHALL print a clear error to stderr identifying the path and the reason, and exit with a non-zero code.

#### Scenario: Nonexistent path
- **WHEN** the user runs `lazyncu ./does-not-exist`
- **THEN** an error naming the resolved path is printed to stderr and the process exits non-zero without opening the TUI

#### Scenario: Directory without any Node project
- **WHEN** the user runs `lazyncu .` in a directory with no `package.json` at its root nor in any subdirectory within the search depth
- **THEN** an error explaining that no Node project was found is printed to stderr and the process exits non-zero without opening the TUI

#### Scenario: Folder of projects is accepted
- **WHEN** the user runs `lazyncu .` in a directory without a root `package.json` but containing subdirectories with `package.json` files
- **THEN** the target is accepted and processed as a scan source

### Requirement: Duplicate and contained targets are not re-registered

When the target equals a registered path, the system SHALL NOT modify the configuration and SHALL select that source in the sources panel on startup. When the target is a strict subpath of a registered path, the system SHALL NOT modify the configuration and SHALL select the covering source immediately; once that source's scan completes, the selection SHALL move to the project entry whose directory matches the target, if one exists. Path containment SHALL be decided per path segment, never by raw string prefix.

#### Scenario: Target equals a registered path
- **WHEN** the user runs `lazyncu .` from a directory already registered
- **THEN** the configuration is unchanged and the cursor is on that source when the TUI opens

#### Scenario: Target is inside a registered path
- **WHEN** the user runs `lazyncu .` from `/projects/api` and `/projects` is registered
- **THEN** the configuration is unchanged, the cursor starts on the `/projects` source, and after its scan completes the cursor selects the project entry for `/projects/api` if the scan discovered it

#### Scenario: Scan does not surface the target
- **WHEN** the target is inside a registered path but the completed scan yields no project entry matching it
- **THEN** the cursor remains on the covering source and no error is raised

#### Scenario: Sibling name prefix is not containment
- **WHEN** the user runs `lazyncu /projects-old/api` and `/projects` is registered
- **THEN** the target is treated as a new path, not as contained in `/projects`

### Requirement: Registering a parent offers to consolidate covered children

When the target is a strict parent of one or more registered paths, the system SHALL add and persist the target, then present a confirmation dialog listing the covered child paths and offering to remove them. Accepting SHALL remove the listed children from the configuration and persist the result; declining SHALL keep them. In both cases the cursor SHALL be on the newly added parent source.

#### Scenario: User consolidates children
- **WHEN** the user runs `lazyncu .` from `/projects` with `/projects/api` and `/projects/web` registered, and accepts the consolidation dialog
- **THEN** `/projects` is registered, `/projects/api` and `/projects/web` are removed, and the change is persisted

#### Scenario: User declines consolidation
- **WHEN** the same dialog is declined
- **THEN** `/projects` is registered and both child paths remain registered

### Requirement: New targets are registered and selected

When the target is valid and neither equal to, contained in, nor a parent of any registered path, the system SHALL add it to the configuration, persist immediately, start its scan, and select the new source in the sources panel on startup.

#### Scenario: Registering the current project
- **WHEN** the user runs `lazyncu .` from an unregistered Node project directory
- **THEN** the directory is appended to the configuration file, its scan starts, and the cursor is on the new source when the TUI opens

