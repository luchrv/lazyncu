# Tasks: add-flow-polish

## 1. Pure helpers (TDD)

- [x] 1.1 Write tests: `detailTitle` with the marked counter (packages view only, hidden at zero), `allSourcesMarks` aggregation across sources, `commandBarContent` (single line, two commands, wrap, truncation indicator, line count), `wrapText` (short passthrough, word wrap, long-token split)
- [x] 1.2 Implement the helpers — tests green

## 2. Rescan all

- [x] 2.1 Add the `R` global binding; implement `rescanAll`/`doRescanAll` (skip in-flight, aggregate confirm through `confirm()`, single status line, all-scanning warning) — dispatch and guard tests

## 3. Command bar and errors

- [x] 3.1 Wire `commandBarContent` into `refreshCommandBar` with dynamic height (store the right flex, clamp 3–6 rows, resize on width change); `c` copies the untruncated command — test the copy path with an oversized command
- [x] 3.2 Implement `renderScanError` (header + retry hint + wrapped error rows) in the detail panel — render test

## 4. Esc in the tree

- [x] 4.1 Unify `Esc` into one context-routed binding: table staging unchanged, tree folds the selected source (or the project's parent), info hint when nothing is foldable — staging tests for both panels

## 5. Verification and docs

- [x] 5.1 README (`R` row, `Esc` scope note); UX-REVIEW UX-13/16/18/19/20 shipped (both languages)
- [x] 5.2 `make check` green; VHS smoke: mark counter in the title, `R` aggregate confirm, truncated command bar with full copy, failed-scan panel, Esc folding
- [x] 5.3 `openspec validate add-flow-polish`; PR created and **paused for user review** (branch protection); after merge: verify HEAD, tag `v0.7.0`, release checks
