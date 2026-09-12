# scan-detection Specification

## Purpose
Infer per registered path, statelessly on every scan, how each path is scanned (single project, workspaces monorepo, or folder of repositories discovered one by one) and which package manager owns each project based on its lockfile.

## Requirements

### Requirement: Scan mode is auto-detected per path
The system SHALL determine the scan mode for each registered path at scan time, without persisting the result, using the following decision tree: if the path contains no `package.json`, the mode is `folder` (folder of repositories, each discovered and scanned on its own); if the path contains a `package.json` with a `workspaces` field or a `pnpm-workspace.yaml` file is present, the mode is `deep` (monorepo); otherwise the mode is `single` (plain project).

#### Scenario: Folder containing multiple projects
- **WHEN** a registered path has no `package.json` in its root
- **THEN** the detected mode is `folder`

#### Scenario: Monorepo with npm/yarn workspaces
- **WHEN** a registered path has a `package.json` whose content includes a `workspaces` field
- **THEN** the detected mode is `deep`

#### Scenario: Monorepo with pnpm workspaces
- **WHEN** a registered path has a `package.json` and a `pnpm-workspace.yaml` file
- **THEN** the detected mode is `deep`

#### Scenario: Single project
- **WHEN** a registered path has a `package.json` with no `workspaces` field and no `pnpm-workspace.yaml`
- **THEN** the detected mode is `single`

#### Scenario: Detection is re-evaluated every scan
- **WHEN** a path previously detected as `single` gains a `pnpm-workspace.yaml` before the next launch
- **THEN** the next scan detects it as `deep` without any user action

#### Scenario: Repository inside a folder is detected on its own
- **WHEN** a folder-mode path contains a repository whose `package.json` declares `workspaces`
- **THEN** that repository is detected as `deep` and its siblings are detected independently

### Requirement: Package manager is detected from lockfiles
The system SHALL determine the package manager of a project directory from its lockfile: `package-lock.json` → npm, `pnpm-lock.yaml` → pnpm, `yarn.lock` → yarn. When no lockfile is present, the system SHALL default to npm.

#### Scenario: pnpm project
- **WHEN** a project directory contains `pnpm-lock.yaml`
- **THEN** the detected package manager is pnpm

#### Scenario: yarn project
- **WHEN** a project directory contains `yarn.lock`
- **THEN** the detected package manager is yarn

#### Scenario: npm project
- **WHEN** a project directory contains `package-lock.json`
- **THEN** the detected package manager is npm

#### Scenario: No lockfile
- **WHEN** a project directory contains no recognized lockfile
- **THEN** the detected package manager defaults to npm

#### Scenario: Multiple lockfiles present
- **WHEN** a project directory contains more than one recognized lockfile
- **THEN** detection resolves deterministically with precedence pnpm-lock.yaml, then yarn.lock, then package-lock.json

### Requirement: Repositories inside a folder are discovered by lazyncu
For a path in `folder` mode, the system SHALL discover repositories by walking the folder and treating the first directory on each branch that contains a `package.json` as one repository, without descending further into it. The walk SHALL follow symbolic links to directories, SHALL never enter the same real directory twice (deduplicating by resolved real path, which also terminates link cycles), SHALL skip `node_modules` and dot-prefixed directories, SHALL descend without a depth limit (matching `ncu --deep`), and SHALL tolerate unreadable entries and dangling links by skipping them. Discovered repositories SHALL be reported in a deterministic order, labeled by their path relative to the folder as reached, and the label of a repository reached through a symbolic link SHALL be the link's name, not the target's.

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
