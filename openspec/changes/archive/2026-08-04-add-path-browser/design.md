# Design: add-path-browser

## Context

See proposal.md — Why. Current mechanics that shape the approach:

- `openAddPath` (`ui/input.go:179`) shows a bare `InputField` on page `pageAddPath`; `addPath` validates via `cfg.AddPath`, saves, and scans the new source. That downstream flow stays untouched.
- Key routing today has two escape hatches: `handleKey` steps aside whenever the focused widget is an `InputField` (`ui/input.go:16`), and `currentContext()` (`ui/app.go:320`) returns `ctxModal` only for the About/Confirm/Keys pages — **`pageAddPath` is not a modal context today**. A `TreeView` inside the add-path modal would receive global dashboard keybindings unless routing changes.
- Modal furniture exists: `pages.AddPage` + `centered()` (`ui/layout.go`), per-page branches in `handleModalKey`, cheat-sheet content generated from the keymap plus static entries (`ui/keys.go`).
- tview's `TreeView` supports per-node expansion callbacks; the wiki's directory-browser pattern (lazy `os.ReadDir` on expand) is the reference implementation.

## Goals / Non-Goals

**Goals:**
- Pick-a-folder flow with zero typing, without losing paste-a-path.
- Follow the app's TUI conventions: Esc closes, titles teach keys, modal keys inert to the dashboard beneath, no color-only signals.

**Non-Goals:**
- Generic file-picker component (directories only, single selection).
- Persisting browser state (last directory, hidden toggle) across opens — every open starts fresh at `$HOME`.
- Async/streamed directory reading — `os.ReadDir` per expanded node is synchronous and fast enough for interactive use.

## Decisions

**D1 — One new file `ui/browser.go` composing InputField + TreeView in a Flex.**
Modal layout: bordered Flex, input row (height 1) on top, tree below, sized ~60% × ~70% of the screen via a new `centeredPct` helper (the existing `centered` takes absolute cells; the tree needs proportional space). Title teaches: `" Add path (Tab input/tree · → expand · Enter select · . hidden · Esc cancel) "` — trimmed to fit if needed.

**D2 — Key routing: `pageAddPath` joins the modal context.**
`currentContext()` adds `pageAddPath` to the `ctxModal` branch, and `handleModalKey` gains a browser branch that owns the modal's keys: `Esc` closes; `Tab` swaps focus input↔tree; with tree focus, `→`/`←`/`↑`/`↓`/`Enter` go to the TreeView and `.` toggles hidden; with input focus, everything passes through to the InputField (the existing InputField step-aside already covers typing). This kills the current latent hole where a non-InputField widget inside `pageAddPath` would leak keys to the dashboard.

**D3 — Lazy tree: read children on expand, directories only, sorted by name.**
Each node holds its absolute path; expanding runs `os.ReadDir`, filters to directories (dot-prefixed excluded unless hidden-visible), and adds children with a placeholder-free contract: a node that fails `os.ReadDir` or yields zero child directories simply shows no children. Symlinked directories are listed and expandable (user-driven depth; no cycle chasing on our side). The hidden toggle re-reads the expanded node's children rather than tracking two child sets.

**D4 — Selection sync is one-way tree→input; Enter semantics by focus.**
Moving the tree selection rewrites the input text to the node's absolute path. `Enter` with tree focus confirms the highlighted node's path; `Enter` with input focus confirms the input text verbatim (typed, pasted, or tree-seeded then hand-edited). Both routes call the existing `addPath`, whose config-store validation remains the single gatekeeper — the browser never pre-validates, so error behavior (status bar message) is unchanged.

**D5 — Cheat sheet gains a Modal/browser group entry.**
The `?` modal's generated content gets the browser keys as static entries alongside the existing modal keys, matching how other non-keymap keys are documented.

## Risks / Trade-offs

- [Huge directories (node_modules, ~/Library) make expansion slow] → Directories-only filtering keeps lists small; expansion is user-initiated and synchronous `os.ReadDir` on one level is milliseconds even for thousands of entries. No pagination for now.
- [Symlink loops if a user keeps expanding a cyclic link] → Depth is always user-driven; no automatic traversal exists to loop. Accepted.
- [`$HOME` unresolvable (empty env in odd environments)] → Fall back to `/`. Cheap guard in the open path.
- [Focus-dependent Enter may surprise] → Input always mirrors the tree selection, so both Enters agree on the same path except after deliberate hand-editing; the title documents the keys.
- [tview TreeView key defaults may overlap the modal's bindings] → Browser branch in `handleModalKey` filters explicitly what reaches the TreeView; anything unlisted dies at the modal boundary (same inert-modal principle as confirm).

## Migration Plan

Pure UI change behind the same `a` keybinding; no config or data migration. Rollback = revert commit.

## Open Questions

None.
