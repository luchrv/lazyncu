# Tasks: add-readme-demos

## 1. Fixtures

- [x] 1.1 Create `demo/fixtures/webapp/`: package.json (`private: true`) pinning outdated deps across major/minor/patch + `lodash@4.17.15` (or equivalent with live advisories); generate and commit `package-lock.json`
- [x] 1.2 Create `demo/fixtures/tools/`: smaller second project, same treatment
- [x] 1.3 Verify fixtures manually: `ncu` shows mixed severities; `npm audit --json` shows vulns with a chain; adjust pins if not

## 2. Recording infrastructure

- [x] 2.1 Write `assets/tapes/setup-demo-env.sh`: build binary, copy fixtures to /tmp (neutral paths), create temp `XDG_CONFIG_HOME` with seeded config variants (npm install unnecessary — audit works from lockfiles)
- [x] 2.2 Write `assets/tapes/hero.tape`: launch, parallel scan streaming, navigate source/project (pinned size/font/theme, <3 MB target)
- [x] 2.3 Write `assets/tapes/vulns.tape`: `v` toggle, severity counters, dependency chain
- [x] 2.4 Write `assets/tapes/add-path.tape`: `a`, type fixture path, immediate scan
- [x] 2.5 Add `make demos` target (setup + vhs each tape); add VHS install check with friendly error

## 3. Record & tune

- [x] 3.1 Install VHS, run `make demos`, iterate on timing/sizing until the 3 GIFs read well and are <3 MB each (98–166 KB actual)
- [x] 3.2 Verify GIFs show no personal paths and real config untouched

## 4. README & docs

- [x] 4.1 Add badge row (release, Go version, license, downloads, Homebrew tap) at README top; verify all render and link
- [x] 4.2 Embed hero GIF after intro, vulns GIF in audit section, add-path GIF in configuration section
- [x] 4.3 Write `docs/DEMOS.md`: install VHS, tape anatomy (annotated hero.tape), regenerate, add new demo, size guidance, network note
- [x] 4.4 ~~Go Report Card~~ — service retired (badge renders "retired"); replaced with Go version badge (design D5 updated)

## 5. Verification & ship

- [ ] 5.1 `make check` green; `make demos` idempotent (second run works)
- [ ] 5.2 Commit + push; verify README renders correctly on GitHub (badges live, GIFs play)
