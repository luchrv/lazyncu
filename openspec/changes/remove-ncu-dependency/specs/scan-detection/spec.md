# Spec Delta

## MODIFIED Requirements

### Requirement: Repositories inside a folder are discovered by lazyncu
For a path in `folder` mode, the system SHALL discover repositories by walking the folder and treating the first directory on each branch that contains a `package.json` as one repository, without descending further into it. The walk SHALL follow symbolic links to directories, SHALL never enter the same real directory twice (deduplicating by resolved real path, which also terminates link cycles), SHALL skip `node_modules` and dot-prefixed directories, SHALL descend without a depth limit, and SHALL tolerate unreadable entries and dangling links by skipping them. Discovered repositories SHALL be reported in a deterministic order, labeled by their path relative to the folder as reached, and the label of a repository reached through a symbolic link SHALL be the link's name, not the target's.

#### Scenario: Flat folder of repositories
- **WHEN** a folder contains `api/package.json`, `web/package.json` and a `notes/` directory without manifest
- **THEN** discovery yields exactly the repositories `api` and `web`

#### Scenario: Discovery stops at the repository root
- **WHEN** a repository `api` contains `api/package.json` and also `api/packages/core/package.json`
- **THEN** discovery yields `api` once and does not yield `api/packages/core`

#### Scenario: Symbolic link to an external repository
- **WHEN** the folder contains `shared -> /elsewhere/shared-lib` and `/elsewhere/shared-lib/package.json` exists
- **THEN** discovery yields a repository labeled `shared`

#### Scenario: Link cycle does not loop
- **WHEN** the folder contains a symbolic link `loop` pointing to the folder itself or to one of its ancestors
- **THEN** discovery completes, yields each real repository exactly once, and never yields a label under `loop/`

#### Scenario: Same repository reachable twice
- **WHEN** the folder contains `api/package.json` and a link `api-link -> api`
- **THEN** discovery yields that repository exactly once

#### Scenario: Dangling link and unreadable directory
- **WHEN** the folder contains a symbolic link whose target does not exist and a directory the process cannot read
- **THEN** discovery skips both and yields the remaining repositories

#### Scenario: Nested groups
- **WHEN** the folder contains `group/api/package.json`
- **THEN** discovery yields a repository labeled `group/api`

#### Scenario: Deeply nested repository
- **WHEN** the only `package.json` in the folder sits at `a/b/c/d/e/repo/package.json`
- **THEN** discovery yields a repository labeled `a/b/c/d/e/repo`

#### Scenario: Ignored directories
- **WHEN** the folder contains `node_modules/pkg/package.json` and `.cache/tool/package.json`
- **THEN** discovery yields neither of them

## ADDED Requirements

### Requirement: Deep manifests are discovered by a recursive walk
For a path in `deep` mode, the system SHALL discover every `package.json` under the path recursively, including manifests nested inside another manifest's directory, skipping `node_modules` and `.pnpm-store`. Discovered manifests SHALL be reported in a deterministic order and labeled by their path relative to the scanned path.

#### Scenario: Nested manifests both discovered
- **WHEN** a monorepo has a root `package.json` and `packages/core/package.json`
- **THEN** discovery yields both manifests

#### Scenario: Dependency and store directories skipped
- **WHEN** the tree contains `node_modules/pkg/package.json` and `.pnpm-store/x/package.json`
- **THEN** neither is discovered

#### Scenario: Root manifest discovered
- **WHEN** the scanned path itself contains a `package.json`
- **THEN** the root manifest is discovered alongside any nested ones
