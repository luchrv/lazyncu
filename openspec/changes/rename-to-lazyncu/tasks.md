# Tasks: rename-to-lazyncu

## 1. Module and Imports

- [x] 1.1 Change go.mod module to `github.com/luchrv/lazyncu` and update every internal import path across the Go packages; `go build ./...` clean
- [x] 1.2 Update `.gitignore` binary entry `/ncu-tui` → `/lazyncu`

## 2. App Identity

- [x] 2.1 `config/config.go`: `appDirName = "lazyncu"`, update doc comments; adjust config tests (XDG path expectations)
- [x] 2.2 `main.go`: error prefix `lazyncu:`, doc comment; sweep remaining doc-comment mentions in other packages
- [x] 2.3 Test-first for the no-migration rule: legacy `ncu-tui` config dir present → ignored, fresh `lazyncu` config created (per modified spec scenario)

## 3. Docs and Spec

- [x] 3.1 README: title, install/clone URLs, usage command, note the config-path breaking change
- [x] 3.2 Verification sweep: `grep -r ncu-tui` (excluding `openspec/changes/archive/`) returns nothing beyond deliberate legacy-rule references (README note, no-migration comment/test fixture) and the live config-store spec updated at archive time; `go test -race ./...` and `go vet ./...` clean

## 4. Local Environment

- [x] 4.1 `git remote set-url origin https://github.com/luchrv/lazyncu.git`
- [x] 4.2 Remove any previously installed `ncu-tui` binary from GOBIN/PATH (verified: none present)
