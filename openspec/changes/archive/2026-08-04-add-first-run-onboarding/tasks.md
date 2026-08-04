# Tasks: add-first-run-onboarding

## 1. config: created flag

- [x] 1.1 Extend config tests: `Load` reports created=true when the file did not exist, created=false for an existing (valid) file; error paths unaffected
- [x] 1.2 Change `Load` to return `(Config, bool, error)` and update `main.go`

## 2. ui: first-run hook

- [x] 2.1 Write UI tests: first-run app opens the browser with the welcome title before the loop; non-first-run app opens nothing; Esc on the welcome browser emits the a/? hint; Esc on a normal browser emits none; adding from the welcome browser registers and scans as usual
- [x] 2.2 Thread the flag `main` → `ui.New` → `App` field; open the welcome browser at the top of `Run` (extract the pre-loop setup so tests can exercise it without entering `tv.Run`)
- [x] 2.3 Add the welcome title variant and the `welcome` flag on `pathBrowser`; emit the dismiss hint on the Esc routes (tree and input focus)

## 3. Verification

- [x] 3.1 Run the gate: `go vet ./...`, `go build ./...`, `go test ./...`
- [x] 3.2 Manual smoke: move `~/.config/lazyncu/config.toml` aside, launch — welcome browser appears; Esc shows the hint; relaunch — nothing auto-opens; restore config
- [x] 3.3 README: note the first-run behavior if the install/usage section warrants it
