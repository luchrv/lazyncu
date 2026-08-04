# Proposal: add-path-browser

## Why

Adding a path today means typing or pasting it blind into a bare text input — a typo earns an error in the status bar and a full retry, and there is no way to discover what is actually on disk from inside the app. A folder browser turns the highest-friction interaction in the TUI into navigation, while keeping the fast paste-a-path flow for users who already have the path at hand.

## What Changes

- The `a` add-path modal becomes a hybrid picker: an editable input field on top (accepts typing/pasting, always shows the current selection) and a lazy directory tree below, starting at `$HOME`.
- Tree navigation: `→` expands a directory (children read on demand, directories only), `←` collapses, `Enter` confirms the selected directory, `Esc` cancels. `Tab` moves focus between input and tree. Navigating the tree rewrites the input text; editing the input by hand and pressing `Enter` uses the typed text verbatim.
- Dot-prefixed directories are hidden by default; `.` toggles their visibility.
- Unreadable directories degrade gracefully (node shows as empty/inaccessible; no crash, no scan abort).
- Confirmation feeds the existing `addPath` flow unchanged: validation via the config store, persistence, immediate scan of the new source.

Out of scope:
- File selection (directories only — files are meaningless as scan sources).
- Recursive pre-loading or filesystem watching; children are read only when a node expands.
- Any change to path removal or to the config store.

## Capabilities

### New Capabilities

None — this reshapes an existing dashboard interaction.

### Modified Capabilities

- `dashboard-ui`: the "User can manage paths from the UI" requirement changes — the add-path input becomes the hybrid browser modal described above, with its navigation, hidden-toggle, and graceful-degradation behavior.

## Impact

- `ui`: new browser modal file (input + TreeView composition, lazy loading, key handling); `openAddPath` swaps to it; `handleModalKey`/keymap gain the modal's context; cheat sheet (`?`) gains the browser keys.
- No changes to `config`, `scanner`, `detect`, or `orchestrator` — `addPath` remains the single entry point.
- Read-only invariant intact: browsing only reads directory listings.
