# Tasks: add-path-browser

## 1. Lazy directory tree core

- [x] 1.1 Write tests for directory listing: dirs only, sorted, dot-prefixed filtered by hidden flag, unreadable dir yields empty list (via `t.TempDir` fixtures)
- [x] 1.2 Implement the listing helper and lazy node-expansion wiring in `ui/browser.go` (node holds absolute path; expand runs the listing; hidden toggle re-reads children)

## 2. Modal composition and key routing

- [x] 2.1 Write tests for browser modal behavior: tree selection rewrites input text; Enter on tree confirms selected dir; Enter on input confirms typed text; Esc closes without adding; `.` toggles hidden children
- [x] 2.2 Build the hybrid modal (InputField + TreeView in a bordered Flex, teaching title, `centeredPct` sizing, `$HOME` root with `/` fallback) and swap `openAddPath` to it
- [x] 2.3 Add `pageAddPath` to the `ctxModal` branch of `currentContext()` and a browser branch in `handleModalKey` (Tab focus swap, explicit key allowlist to the TreeView, input passthrough)

## 3. Discoverability

- [x] 3.1 Add the browser keys to the `?` cheat-sheet content (with test if the sheet's content is asserted)

## 4. Verification

- [x] 4.1 Run the gate: `go vet ./...`, `go build ./...`, `go test ./...`
- [x] 4.2 Manual smoke in a real terminal: `a` opens browser at `$HOME`; navigate, toggle hidden, pick a project dir, confirm scan starts; paste path into input still works; Esc cancels; dashboard keys dead under the modal
- [x] 4.3 Update README keybindings section if it lists the add-path flow
