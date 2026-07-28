# Tasks: add-keymap-safety

## 1. Declarative keymap (TDD — pure, no tview)

- [x] 1.1 Write tests for `ui/keymap.go`: context resolution table, `lookup(key, ctx)` returning bound action / other-context hint / nothing, and per-context help generation (full + compact variants, valid-color-tag convention, ordering)
- [x] 1.2 Implement `ui/keymap.go`: `context` enum (`global`, `tree`, `tablePackages`, `tableVulns`, `modal`), `binding` struct (key/rune, contexts, action id, help label, priority), the keymap table, lookup, and help generators — tests green
- [x] 1.3 Implement `currentContext()` on `App` deriving the context from `pages` state, `tableFocused`, and `showVulns`

## 2. Context-scoped dispatch and teaching hints

- [x] 2.1 Rewrite `handleKey` to resolve `currentContext()` and dispatch through the keymap; delete the flat unconditional switch; modal context inert except About `q`/`Esc`/`h` and confirmation `Esc`/buttons
- [x] 2.2 Implement teaching hints: out-of-context bound key sets a status hint ("d only works in Sources — Tab to go back") with ~5 s TTL via `time.AfterFunc` + `QueueUpdateDraw`, guarded by a generation counter (write the counter guard test first); unbound keys silent
- [x] 2.3 Verify the About leak is closed: with About open, `a`/`d`/`v`/`r`/`m` do nothing, `q` quits, `Esc`/`h` close

## 3. Confirmation modals

- [x] 3.1 Write tests for the pure guards: `markCount(sourceState)` aggregating marks across all projects, and the confirm-message formatter ("discards N marks across M projects")
- [x] 3.2 Implement `ui/confirm.go`: reusable `confirm(title, text, onAccept)` wrapping `tview.NewModal` — buttons [Cancel, Confirm], default focus Cancel, `Esc` cancels, mouse-clickable, `q` inert
- [x] 3.3 Wire `d`: always confirm ("Stop tracking <path>? The folder on disk is not touched."); removal only on confirm
- [x] 3.4 Wire `r`: confirm only when `markCount > 0` with the aggregate message; immediate rescan otherwise; marks cleared only after confirmation

## 4. Counter encoding and legend

- [x] 4.1 Write tests for `updateSummary` (`▲3 ●5 ▪2`, colors, up-to-date case) and `auditSummary` (letter+digit order `C1 H2 M3 L4`, n/a and failed states unchanged)
- [x] 4.2 Update `ui/panel.go` summaries to the shape/letter encoding — tests green
- [x] 4.3 Add the legend to the About modal (`▲ major ● minor ▪ patch` / `C critical H high M moderate L low`), adjusting modal height

## 5. Width-aware generated help bar

- [x] 5.1 Delete `statusHelpTree`/`statusHelpTable`/`helpWidth`; render help from the keymap generators; pick full vs compact variant from terminal width (pure selector function, tested) keeping a minimum message-zone width
- [x] 5.2 Hook variant selection on resize/draw so the bar adapts live

## 6. Verification and docs

- [x] 6.1 Update `refreshHelp` call sites, `README.md` keybindings table (confirmations, counter legend, context scoping) and check `docs/UX-REVIEW.md` items UX-01/02/07 as addressed
- [x] 6.2 `make check` green (gofmt, vet, race tests); manual smoke pass over the five contexts, both confirmations, hint expiry, and narrow-terminal help
- [x] 6.3 `openspec validate --change add-keymap-safety` passes
