# lazyncu

A read-only terminal dashboard for [npm-check-updates](https://github.com/raineorshine/npm-check-updates). It answers one question at a glance: **which of my projects need updates, and how urgent are they?**

- Scans **global packages** (`ncu -g`) and every **registered path** in parallel on launch.
- Auto-detects what each path is — single project, monorepo, or folder of projects — and picks `ncu` or `ncu --deep` accordingly. Zero per-path configuration.
- Classifies every upgrade as **major / minor / patch** with color coding and per-project counters.
- Runs **`npm audit` / `pnpm audit`** per project alongside the version scan: severity counters (critical/high/moderate/low), vulnerable-package detail, and the dependency chain that drags each vulnerability in (`lodash ← express`).
- **Never modifies anything.** It shows the exact update/fix command for the current selection and copies it to your clipboard.

## Requirements

- [npm-check-updates](https://github.com/raineorshine/npm-check-updates) >= 18 on PATH: `npm install -g npm-check-updates`
- `npm` (and `pnpm` if you want pnpm projects audited)
- Network access (ncu queries the npm registry; audit queries the advisory endpoint)

## Install

```sh
go install github.com/luchrv/lazyncu@latest
```

Or from source:

```sh
git clone https://github.com/luchrv/lazyncu && cd lazyncu && go build
```

## Usage

```sh
lazyncu
```

All sources scan in parallel; results stream in as each finishes. Select a source or project in the left panel to see its packages, toggle the vulnerability view, and copy the suggested command.

### Keybindings

| Key | Action |
|-----|--------|
| `q` | Quit |
| `c` | Copy the visible command (update command; fix command in the vulnerability view) |
| `v` | Toggle vulnerability detail view |
| `r` | Rescan the selected source (disabled while it is already scanning) |
| `a` | Add a path (validated, persisted, scanned immediately) |
| `d` | Remove the selected path |
| `Enter` | Collapse/expand the selected source's project list |
| `m` | Hide/show status messages (bottom left) |
| `↑↓` | Navigate sources and projects |

### Suggested commands (shown, never executed)

| Context | Command |
|---------|---------|
| Global packages | `npm install -g pkg@x.y.z ...` |
| npm project | `cd <dir> && ncu -u && npm install` |
| pnpm project | `cd <dir> && ncu -u && pnpm install` |
| yarn project | `cd <dir> && ncu -u && yarn` |
| Vulnerabilities (npm) | `cd <dir> && npm audit fix` |
| Vulnerabilities (pnpm) | `cd <dir> && pnpm audit --fix` |

## Configuration

`$XDG_CONFIG_HOME/lazyncu/config.toml` (default `~/.config/lazyncu/config.toml`), created on first launch. Manage paths from the UI or edit by hand:

> **Renamed from ncu-tui:** an existing `~/.config/ncu-tui/` is ignored — no migration. Re-add your paths with `a` (or copy the old `config.toml` into the new directory yourself).

```toml
timeout_ms = 30000   # per-command timeout (default 30000)

[[paths]]
path = "/Users/me/projects"        # folder of projects → ncu --deep

[[paths]]
path = "/Users/me/projects/my-app" # single project → ncu
```

How a path is scanned is re-detected on every launch:

| Path contents | Mode |
|---------------|------|
| No `package.json` | `ncu --deep` (folder of projects) |
| `package.json` with `workspaces`, or `pnpm-workspace.yaml` | `ncu --deep` (monorepo) |
| Plain `package.json` | `ncu` |

## Audit coverage notes

- **Global packages are not audited** — `npm audit` requires a lockfile and does not support global installs. The UI shows "audit n/a", which is distinct from "0 vulns".
- **Yarn projects are not audited** in v1 (yarn classic emits a different audit format). Version scanning works normally.
- `npm audit` exiting non-zero *with* valid JSON means vulnerabilities exist — that is a successful audit, and the dashboard treats it as such.

## Development

```sh
make check   # gofmt + go vet + race tests + coverage
```

Business logic lives in pure, exec-injected packages (`config`, `detect`, `scanner`, `semver`, `command`, `audit`, `orchestrator`); the `ui` package is a thin tview layer where every async widget update passes through a single `QueueUpdateDraw` choke point.

## License

See [LICENSE](LICENSE).
