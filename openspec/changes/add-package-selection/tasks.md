# Tasks: add-package-selection

## 1. Filtered commands (TDD, pure package)

- [x] 1.1 Write tests for `command.ProjectUpdateFiltered` (npm/pnpm/yarn, single + multiple pkgs, scoped names) and `command.GlobalUpdateFiltered` (subset, empty → empty string)
- [x] 1.2 Implement both functions; existing `ProjectUpdate`/`GlobalUpdate` untouched (install-step map extracted, DRY)

## 2. Selection state

- [x] 2.1 Add per-project mark storage to `sourceState` (immutable-style toggle/clear helpers); clear on rescan (`scanOne`) and path removal
- [x] 2.2 Wire mark lookup for the current `selection{source, projectIdx}` entry

## 3. Table focus & marking UI

- [x] 3.1 Add `Tab` focus toggle tree↔table in `handleKey` (guarded: no-op while About modal or input dialog open); table `SetSelectable(true, false)` only while focused; `Esc` returns to tree
- [x] 3.2 Render marks in `refreshDetail`: `✓` prefix + yellow package name; keep row→package mapping for the cursor
- [x] 3.3 Handle `space` (toggle mark under cursor) and `x` (clear project marks) while table focused
- [x] 3.4 `refreshCommandBar`: use filtered command variants when current entry has ≥1 mark; `fix:` line untouched

## 4. Contextual help bar

- [x] 4.1 Split `statusHelp` into tree/table variants (`Tab pkgs` hint added to tree variant); `refreshHelp()` on focus change; resize `helpWidth` to the longer variant

## 5. Verification & docs

- [ ] 5.1 TUI smoke test (tmux): Tab focus both ways, mark/unmark, `x` clear, filtered command in bar, `c` copies filtered, rescan clears marks, `q` quits from table focus, help bar switches
- [ ] 5.2 Update README keybindings table (`Tab`, `␣`, `x`, `Esc`)
- [ ] 5.3 `make check` green (coverage ≥80% pure packages)
- [ ] 5.4 Commit + push
