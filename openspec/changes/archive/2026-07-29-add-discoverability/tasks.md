# Tasks: add-discoverability

## 1. Keymap extensions (TDD — pure)

- [x] 1.1 Write tests: `barHidden` rows excluded from `helpText` but present in the cheat-sheet text; `?`/`1`/`2` resolve as global dispatch rows; cheat-sheet groups (Global / Sources / Packages) contain the expected entries and the legend lines
- [x] 1.2 Add `barHidden` to `binding`, skip it in `helpText`, and add bindings: `?` (help, compact, opens cheat sheet), `1` (focus sources, barHidden), `2` (focus packages, barHidden) — tests green
- [x] 1.3 Implement `keysText()` generating the grouped cheat-sheet body from the keymap plus a shared `legendLines()` helper

## 2. `?` cheat-sheet modal

- [x] 2.1 Implement `ui/keys.go`: `pageKeys` centered modal sized from the generated content; `toggleKeys()` open/close
- [x] 2.2 Extend `currentContext()` and `handleModalKey`: `Esc`/`?` close the cheat sheet, `q` quits, everything else inert; write the modal-inertia test (mirror of the About test)
- [x] 2.3 Move the legend out of About (revert height), keep About content version-only

## 3. Source aggregates

- [x] 3.1 Write tests for `aggregateSource(projects)`: semver sums, audit sums over StatusOK only, failed count, all-n/a case, empty projects
- [x] 3.2 Implement aggregation and render it in `sourceText` for registered sources (loading/failed states unchanged); audit part: sums, `✗` marker on failures, `0 vulns` clean, `audit n/a` when nothing usable — tests green
- [x] 3.3 Verify fold keeps the aggregate visible (existing fold indicator prepends to the same text)

## 4. Focus affordance

- [x] 4.1 Number the titles (` 1 Sources `, ` 2 Packages …` incl. the vulns variant) and implement `applyFocusStyle()` (yellow border+title on the focused panel), called from `setTableFocus` and initial build
- [x] 4.2 Wire `1`/`2` dispatch to `setTableFocus(false/true)`; write dispatch tests (jump from either panel, inert in modals)

## 5. First-run empty state

- [x] 5.1 Write tests: predicate (no paths registered), tree hint nodes present and non-selectable, hint absent once a path exists, onboarding text shown only when the global view would be empty (packages present → rows win; scan error → error wins)
- [x] 5.2 Implement tree hint nodes and the Packages-panel onboarding text — tests green

## 6. Verification and docs

- [x] 6.1 README: add `?`, `1`, `2` to the keybindings table; note the legend now lives in `?`; mark UX-03/05/06/09 shipped in `docs/UX-REVIEW.md` (both languages)
- [x] 6.2 `make check` green; VHS smoke: cheat sheet content, focus highlight following Tab/1/2, source aggregates folded and unfolded, empty-state launch
- [x] 6.3 `openspec validate add-discoverability` passes
