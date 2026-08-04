# Tasks: add-ncu-brew-dependency

## 1. GoReleaser configuration

- [x] 1.1 Add `dependencies: [{formula: npm-check-updates}]` to the `homebrew_casks` entry in `.goreleaser.yaml`
- [x] 1.2 Run `goreleaser check` to validate the configuration
- [x] 1.3 Run `goreleaser release --snapshot --clean --skip=publish` and verify the generated cask under `dist/` contains `depends_on formula: "npm-check-updates"`

## 2. README

- [x] 2.1 Update the Requirements section: scope the `npm install -g npm-check-updates` prerequisite to release-binary / `go install` / source installs
- [x] 2.2 Update the Install section: note that `brew install luchrv/tap/lazyncu` installs npm-check-updates automatically

## 3. Verification

- [x] 3.1 Run the standard gate: `go vet ./...`, `go build ./...`, `go test ./...` (no Go changes expected — confirms nothing regressed)
- [x] 3.2 Review the README diff for accuracy against the delta spec scenarios
