# Proposal: add-package-selection

## Why

Today the suggested update command is all-or-nothing: `ncu -u` upgrades every dependency of a project. Users often want to upgrade only specific packages (skip risky majors, take a security patch). ncu supports positional filters (`ncu -u lodash debug`), so lazyncu can offer per-package selection while staying strictly read-only — the selection only shapes the copyable command.

## What Changes

- Make the Packages table focusable and navigable: `Tab` toggles focus between the sources tree and the table, `↑↓` moves across rows.
- Mark/unmark packages with `space` (visual: `✓` prefix + yellow package name); `x` clears all marks in the current project.
- With ≥1 mark, the command bar's `update:` line becomes the filtered command — `cd <dir> && ncu -u <pkg…> && <install>` for projects, subset of `npm install -g <pkg>@<ver> …` for the global source. No marks → full command as today. The `fix:` line is untouched.
- Help bar becomes focus-contextual: tree-focus variant (current keys + `Tab pkgs`), table-focus variant (`q · ↑↓ move · ␣ mark · x clear · c copy cmd · Tab/Esc back`).
- Selection lives per project in memory: preserved when switching focus or projects, cleared on rescan or path removal. Never persisted to config.
- README keybindings table updated. **Out of scope:** vulnerability view selection (`npm audit fix` cannot filter per package), new demo GIF (tapes exist; deferred), any command execution.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dashboard-ui`: table focus + navigation, package marking with visual indicator, focus-contextual help bar, selection lifecycle (per-project, cleared on rescan).
- `update-commands`: filtered command variants when a selection exists (project positional-filter form and global subset form).

## Impact

- **Code**: `ui/` (focus handling in `input.go`, table selectability + mark rendering in `detail.go`/`layout.go`, help bar switching, selection state in `app.go` per-project state), `command/` (subset variants of `ProjectUpdate`/`GlobalUpdate`).
- **Specs**: delta specs for `dashboard-ui` and `update-commands`; read-only guarantee unchanged and re-asserted.
- **Docs**: README keybindings table.
- **No new dependencies; no config format changes.**
