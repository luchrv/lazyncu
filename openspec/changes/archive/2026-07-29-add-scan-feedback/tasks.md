# Tasks: add-scan-feedback

## 1. Leveled, expiring messages (TDD — pure first)

- [x] 1.1 Write tests: level decoration (icon+color per level, printable icon survives tag stripping), expiry scheduled for info/ok/warn and not for error, generation guard reuse, `m` not resurrecting an expired message
- [x] 1.2 Implement `msgLevel` + decoration, change `setStatus(level, format, ...)`, generalize the hint TTL into it (`clearIfCurrent`), absorb `showHint`; error persists — tests green
- [x] 1.3 Update every call site with its level (copy ok/error, rescan info/warn, add/remove ok/error/warn, hints warn) and fix affected tests

## 2. Spinner

- [x] 2.1 Write tests: `spinnerGlyph(frame)` cycles the braille frames; `sourceText` loading row and the detail loading message include the glyph; `anyLoading` predicate
- [x] 2.2 Implement the self-stopping ticker goroutine (`ensureSpinner`, ~120 ms, guarded by a `spinning` flag, frames applied via `QueueUpdateDraw`), hooked from `refreshAll` — tests green

## 3. Aggregate progress

- [x] 3.1 Write tests: `progressText(done, total)` formatting, hidden when nothing loading, done/total counting from source states
- [x] 3.2 Add the progress segment to the bottom flex (messages | progress | help) and `refreshProgress()` sizing/hiding it — tests green

## 4. Verification and docs

- [x] 4.1 README: message-levels note; mark UX-10/11/12 shipped in `docs/UX-REVIEW.md` (both languages)
- [x] 4.2 `make check` green; VHS smoke: spinner animating during launch, `scanning N/M` visible alongside a copy toast, toast expiring, error persisting
- [x] 4.3 `openspec validate add-scan-feedback` passes
