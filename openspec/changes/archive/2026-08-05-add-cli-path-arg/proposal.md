## Why

Running `lazyncu` from inside a project currently requires opening the TUI and registering the current directory by hand through the add-path browser. Users expect the common CLI idiom `lazyncu .` (or `lazyncu <path>`) to register the directory they are standing in — or, when it is already covered by a registered path, to simply land the cursor on it.

## What Changes

- `lazyncu` accepts an optional positional path argument (0 or 1): `lazyncu .`, `lazyncu ..`, `lazyncu ~/projects/api`, `lazyncu /abs/path`. More than one positional argument is a clear error.
- The argument is resolved to an absolute path against the current working directory (with `~` expansion and symlink resolution for comparisons).
- Before entering the TUI, the resolved path is validated: it must exist, be a directory, and be a Node target (a `package.json` at its root or within a bounded depth, skipping `node_modules`). Invalid targets print an error to stderr and exit non-zero without opening the TUI.
- Dedupe and containment against registered paths:
  - Exactly equal to a registered path → nothing is added; the cursor selects that source.
  - Subpath of a registered path → nothing is added; after that source's scan completes, the cursor selects the exact matching project node (falling back to the covering source while scanning or when no node matches).
  - Parent of one or more registered paths → the parent is added and persisted, then a confirmation modal offers to remove the now-covered child paths.
  - Otherwise → the path is added, persisted, and the cursor selects the new source.
- Running `lazyncu` with no argument keeps today's behavior exactly; the current directory is never registered implicitly.

## Capabilities

### New Capabilities

- `cli-path-argument`: Optional positional path argument on launch — resolution, pre-TUI validation, dedupe/containment against registered paths, parent consolidation offer, and startup cursor targeting of the matching source or project node.

### Modified Capabilities

<!-- No existing capability's requirements change. config-store's AddPath/RemovePath semantics stay as specified; app-version's argument handling is untouched (--version still wins as first argument). -->

## Impact

- `main.go`: positional argument parsing alongside `--version`; pre-TUI validation and error path.
- `config`: containment/normalization helper (symlink-resolved, segment-wise comparison) used by the launch flow; no change to persisted format or `AddPath`/`RemovePath` behavior.
- `detect` (or a new helper): bounded-depth Node-target check (`package.json` lookup skipping `node_modules`) — note `detect.ScanMode` cannot be reused as a validator since missing `package.json` maps to deep mode.
- `ui`: `New`/`Run` accept an optional launch target; pending-selection state resolved in `applyEvent` when the covering source's scan event arrives; consolidation confirm modal reusing the existing confirm component.
- No new dependencies. No breaking changes: bare `lazyncu` and `lazyncu --version` behave exactly as before.
