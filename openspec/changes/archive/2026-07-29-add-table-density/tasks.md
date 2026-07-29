# Tasks: add-table-density

## 1. Pure view transforms (TDD)

- [x] 1.1 Write tests: `sortPackages`/`sortVulns` (severity rank, name, scan passthrough, stability, no input mutation), `filterPackages`/`filterVulns` (case-insensitive substring, empty query passthrough), `middleEllipsis` (short passthrough, ends preserved), severity/vuln cell labels (`▲ major`, `C critical`), title composition (base, +sort, +filter, both)
- [x] 1.2 Implement the transforms, labels, and `detailTitle()` — tests green

## 2. Sort and glyph cells

- [x] 2.1 Add `sortMode` state and the `s` binding (both table views); cycle + title indicator + refresh
- [x] 2.2 Apply sort in `renderPackages`/`renderVulns` before row rendering (rowPkgs mapping follows); severity cells use the glyph labels — dispatch and rendering tests

## 3. Inline filter

- [x] 3.1 Add the collapsed `InputField` to the bottom flex, the `/` binding (both table views) expanding/focusing it, live `SetChangedFunc` filtering, `Enter` keep+focus table, `Esc` clear+close
- [x] 3.2 Two-stage `Esc` in the table (clear filter first, tree second) — tests for staging and command-ignores-visibility

## 4. Column sizing

- [x] 4.1 Per-column expansion: only `Package` and `Via` expand; `Via` capped with `middleEllipsis` — vulns render test

## 5. Verification and docs

- [x] 5.1 README (`s`, `/` rows in the keybindings table); UX-REVIEW: UX-14/15/17 shipped, UX-21 partial-shipped note (both languages)
- [x] 5.2 `make check` green; VHS smoke: severity sort, live filter narrowing rows with `q` in the query, Esc staging, Via ellipsis
- [x] 5.3 `openspec validate add-table-density` passes
