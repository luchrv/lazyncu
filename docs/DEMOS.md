# Recording demos

The README GIFs are not screen recordings — they are rendered from
[VHS](https://github.com/charmbracelet/vhs) tape files, so anyone can
regenerate them identically after a UI change.

```
demo/fixtures/           assets/tapes/*.tape
  (synthetic npm             (scripted keystrokes,
   projects, old deps)        pinned size/theme)
        │                         │
        └──────► make demos ◄─────┘
                     │
                     ▼
            assets/demo/*.gif  ──► embedded in README.md
```

## Prerequisites

- `brew install vhs` (renders tapes headlessly; bundles its own font)
- `ncu` and `npm` on PATH (same requirements as lazyncu itself)
- **Network access** — recordings run real `ncu` and `npm audit` against the
  npm registry.

## Regenerating all demos

```sh
make demos
```

This runs `assets/tapes/setup-demo-env.sh`, which:

1. Copies `demo/fixtures/` to `/tmp/lazyncu-demo/fixtures/` — GIFs must show
   neutral `/tmp` paths, never personal ones.
2. Seeds throwaway configs under `/tmp/lazyncu-demo/config/` and the tapes
   launch lazyncu with `XDG_CONFIG_HOME` pointing there — your real
   `~/.config/lazyncu` is never read or written.
3. Builds the binary.

Then it renders each tape in `assets/tapes/` to its GIF in `assets/demo/`.

## Tape anatomy

Annotated excerpt from `hero.tape`:

```tape
Output assets/demo/hero.gif   # where the GIF lands

Set FontSize 15               # pin rendering so output is identical
Set Width 1200                # on every machine
Set Height 700
Set Theme "Dracula"
Set TypingSpeed 60ms

Hide                          # setup happens off-camera:
Type "cp ... && export XDG_CONFIG_HOME=... && export PATH=$PWD:$PATH && clear"
Enter
Show                          # camera on

Type "lazyncu"                # what the viewer sees typed
Enter
Sleep 12s                     # wait for parallel scans to finish

Down                          # drive the TUI: arrow keys, single-letter
Sleep 2.5s                    # keybindings, generous pauses to read
Type "q"                      # always quit at the end
```

Gotcha: the VHS parser does not support escaped quotes (`\"`) inside `Type`
strings — write shell setup lines without them.

## The fixtures

`demo/fixtures/webapp` and `demo/fixtures/tools` pin intentionally outdated
dependencies (majors, minors, and a patch) plus packages with published
advisories (`lodash@4.17.15`, `axios@0.21.1`, `semver@7.3.5`) so the
vulnerability view has real content. Lockfiles are committed — `npm audit`
works from them directly; fixtures never need `npm install`.

If an advisory gets withdrawn and the vulns demo goes empty, run `npm audit`
inside the fixture, pick another old pin with live advisories, regenerate the
lockfile with `npm install --package-lock-only`, and re-record.

## Adding a new demo

1. Copy an existing tape in `assets/tapes/` and rename it (`<feature>.tape`).
2. Keep the `Set` block and the hidden setup line as-is (pick the config
   variant: `config-full.toml` registers both fixtures, `config-addpath.toml`
   only webapp).
3. Script the keystrokes; leave 2–3 s of `Sleep` after every visible action
   so viewers can read.
4. Add the tape to the `demos` target in the Makefile.
5. Run `make demos`, inspect the GIF, tune timing.
6. Reference it from the README: `![<alt>](assets/demo/<feature>.gif)`.

Keep GIFs under ~3 MB (current ones are well below); if one balloons, reduce
duration before resolution.

## Content ages — that's fine

The registry moves on: fixture scans will show more pending updates and
different audit counts over time. The demo *structure* is stable (the pins
guarantee majors/minors/patch and vulnerabilities). Regenerate whenever the
UI changes or the GIFs look stale.
