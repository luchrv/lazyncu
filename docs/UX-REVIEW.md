# lazyncu — TUI UX/UI Review

> Bilingual document · Documento bilingüe
> [English](#english) · [Español](#español)

Reviewed version: `main` @ commit `82e1f0b` (2026-07-27)
Scope: audit of the **existing** interface only — no new product features proposed.
Constraint honoured: **lazyncu stays read-only**. Nothing below suggests executing `ncu -u`, `npm install`, or `npm audit fix` from inside the app.
Benchmarks used: `lazygit`, `k9s`, `lazydocker`, `gh dash`, `btop`.

---

# English

## 1. Executive summary

lazyncu already gets the hardest part right: a single screen answers "which projects need updates and how urgent". The architecture (thin `ui/` over pure core packages) makes every improvement below cheap to implement.

The weaknesses are not in what the app shows, but in **what it doesn't tell the user**:

| Theme | Diagnosis |
|-------|-----------|
| Discoverability | Full keymap lives in the README, not in the app. No `?` cheat sheet. No first-run guidance. |
| Focus model | Two focusable panels, but focus is only signalled by a help-bar swap — no visual anchor. |
| Key safety | Every global key fires regardless of focus; `d` deletes a registered path with no confirmation and no undo. |
| Information scent | Collapsed/registered source nodes show a bare folder name — zero signal. Only the global node summarises. |
| Encoding | Severity is carried by colour + one ambiguous letter. `M` means *major* on the left of the row and *moderate* on the right of the same row. |
| Responsiveness | The help bar is a hardcoded 106 columns. Below ~110 cols the status zone collapses and help clips. |

23 improvements are listed in §4, grouped P0/P1/P2.

## 2. Observations by area

### 2.1 Layout and spatial model

`ui/layout.go:58-78` builds: tree (flex 1) | [packages table (flex 1) + command bar (4 fixed rows)] (flex 2), plus a 1-row bottom bar.

- **O-01 — Fixed-width help bar.** `helpWidth = 106` (`ui/layout.go:15`) is a hard allocation. `statusHelpTree` is itself ~100 printable columns. On an 80-column terminal — still the default in many contexts — the message zone gets 0 columns and the help itself is clipped. There is no minimum-size guard and no compact variant.
- **O-02 — Command bar is fixed at 4 rows** (`cmdBarRows`, `ui/layout.go:16`) but renders at most 2 lines. Two rows are permanently wasted vertical space in the densest part of the screen; conversely, a filtered update command over many marked packages wraps and can exceed 4 rows with no overflow indicator.
- **O-03 — No focus affordance.** Nothing sets a border colour or title marker when focus moves (`setTableFocus`, `ui/input.go:79-91`). The only signals are the help-bar text swap and row selectability. In a two-panel TUI the focused panel should be unmistakable at a glance — lazygit and k9s both re-colour the focused border.
- **O-04 — Panels are not addressable.** No `1`/`2` (lazygit) or number-jump convention; `Tab` is the only way to move focus, and it is a toggle, so it does not scale if a third panel is ever added.
- **O-05 — The Sources panel title carries no hint** while the other two do (`" Packages (v for vulnerabilities) "`, `" Command (copy with c) "`). The pattern is good and inconsistently applied.

### 2.2 Keymap and interaction

- **O-06 — Global keys ignore focus.** `handleKey` (`ui/input.go:35-73`) dispatches `q c v r a d m h` unconditionally. With the packages table focused, `d` still removes the registered path — even though `statusHelpTable` (`ui/layout.go:14`) does not list `d` at all. A key that is invisible in the current context but still destructive is the single worst interaction defect in the app.
- **O-07 — Destructive action without confirmation.** `removeSelectedPath` (`ui/input.go:211-228`) unregisters the path and persists the config immediately. No confirmation, no undo, no "press d again to confirm". The blast radius is small (a config entry), the surprise is not.
- **O-08 — No in-app cheat sheet.** There is no `?`. `h` opens About (`ui/input.go:52`), which is doubly unconventional: `?` is the near-universal TUI help key, and `h` is vim's "left". The About modal (12 rows, `ui/about.go:14`) is mostly empty and would host the keymap comfortably.
- **O-09 — `Esc` is inconsistent.** It closes the About modal and leaves the table (`ui/input.go:19-27`), but does nothing in the tree. Users learn "Esc = go back one level"; here the last level is silent.
- **O-10 — Marking is hard to discover.** `Space` only works with the table focused (`ui/input.go:61-65`), which requires already knowing `Tab`. Nothing in the tree context suggests packages are selectable.
- **O-11 — No rescan-all.** `r` rescans the selected source only (`rescanSelected`, `ui/input.go:142`). After a `git pull` across a workspace the user must visit each source.
- **O-12 — No search/filter and no sort** anywhere. Package tables are rendered in scan order (`renderPackages`, `ui/detail.go:57`). For a project with 40 outdated packages there is no way to see majors first or jump to a name.

### 2.3 Feedback, state and time

- **O-13 — Loading is a static string.** `"scanning…"` in the table, `"[gray]scanning…[-]"` in the tree (`ui/panel.go:64-65`). No spinner, so a slow registry call is indistinguishable from a hung app. `ncu` over many projects routinely takes tens of seconds, with a 30 s per-command timeout (`config.DefaultTimeoutMS`).
- **O-14 — No aggregate progress.** Sources scan in parallel and stream in, but nothing shows "4/9 sources done". The tree must be read row by row to know whether the launch scan finished.
- **O-15 — Status messages never expire.** `setStatus` (`ui/app.go:183`) writes and the text stays until the next message. Ten minutes later the user still sees `copied: cd /x && ncu -u && npm install` and cannot tell whether it refers to the current selection. `m` hides the zone entirely, which is a blunt fix for a staleness problem.
- **O-16 — Message severity is ad-hoc.** Colour tags are inlined at each call site (`[red]`, `[green]`, `[yellow]`) rather than derived from a level. There is no icon prefix, so on a monochrome terminal an error reads exactly like a confirmation.
- **O-17 — Scan errors are terminal.** A failed source shows `✗ scan failed` in the tree and `scan failed: <err>` in the table (`ui/detail.go:41-43`). The message is not scrollable or copyable, and nothing hints that `r` retries.

### 2.4 Information encoding and density

- **O-18 — Letter collision on the same line.** A project row reads `3M 5m 2p │ 1C 2H 3M` (`updateSummary` / `auditSummary`, `ui/panel.go:82-126`). The left `M` is *major*, the right `M` is *moderate*. Two different taxonomies, one glyph, one line, no legend anywhere in the UI.
- **O-19 — Registered sources carry no summary.** `sourceText` returns the bare `filepath.Base` for any non-global source (`ui/panel.go:71-72`). The global node shows counters; a registered path does not. Collapse it (`Enter`) and the row becomes pure noise — the fold gesture destroys all information instead of aggregating it.
- **O-20 — Same-basename sources are indistinguishable.** `~/work/api` and `~/personal/api` both render as `api`.
- **O-21 — Marking overwrites severity colour.** `renderPackages` (`ui/detail.go:62-65`) replaces cell 0 with a yellow `✓ name`, so a marked package loses its major/minor/patch colour on the name while the other cells keep it — a half-coloured row. The `✓ ` prefix also shifts the text, breaking the left alignment of the column.
- **O-22 — Mark count is invisible.** The only evidence of how many packages are selected is reading the generated command. No `3 selected` anywhere, and no way to see marks without focusing the table.
- **O-23 — All columns expand equally.** `SetExpansion(1)` on every cell (`ui/detail.go:203-212`) gives `Severity` (≤5 chars) the same growth as `Package` (up to 40+). In the vulnerability view the `Via` chain (`lodash ← express ← …`) is the longest field and the first to be clipped, with no ellipsis marker.
- **O-24 — Colour is the primary channel.** Severity has no shape/symbol backup (`severityColor`, `vulnColor`). Under `NO_COLOR`, a monochrome profile, or common red/green colour-blindness, major and patch rows become identical.

### 2.5 Onboarding and first run

- **O-25 — Empty first run is a dead end.** `config.Load` creates an empty config on first launch (`config/config.go:57-63`). The user sees a tree with one `Global (npm -g)` node and an empty right panel. Nothing says "no paths registered yet — press `a` to add one". The bottom help bar does list `a add path`, but it is one item among ten at the far right of the screen.
- **O-26 — Preflight failure is pre-TUI.** `ncu` missing or too old prints to stderr and exits (`scanner.Preflight`, `main.go:54`). This is the correct behaviour and the message already includes the install command — no change needed, listed here for completeness.
- **O-27 — Mouse is enabled but undocumented.** `EnableMouse(true)` (`ui/app.go:136`). Clicking navigates, but click-to-mark and click-to-sort do not exist and no affordance suggests the mouse works at all.

### 2.6 What already works well (do not regress)

- Read-only invariant, made visible: the command bar shows exactly what will run, and `c` copies it. This is the product's differentiator.
- Titles that teach (`" Packages (v for vulnerabilities) "`) — cheap, contextual, effective.
- Message zone and help zone are separate, so a message can never hide the help (`ui/layout.go:66-70`). Many TUIs get this wrong.
- The three audit states (`n/a` / `✗` / `0 vulns`) are kept visually distinct — "not audited" is not misreported as "clean".
- Rescan is guarded against overlapping scans, with an explicit message instead of a silent no-op (`ui/input.go:148-151`).
- `m` to reclaim the message zone is a nice power-user touch.

## 3. Proposed target screens

### 3.1 Main screen

```
┌─ 1 Sources ────────── a add · d del · r rescan ─┐┌─ 2 Packages ─ / filter · s sort · v vulns ────┐
│ ▾ Global (npm -g)      maj 2 min 1 · audit n/a  ││    Package          Current   New      Sev   │
│ ▾ work/api             maj 3 min 5 pat 2 · C1 H2││ [✓] axios            0.21.1   1.7.7    major │
│   ├ api-gateway        maj 1 min 2 · 0 vulns    ││ [ ] chalk             4.1.0    5.3.0    major │
│   └ api-workers        maj 2 min 3 pat 2 · C1 H2││ [✓] express          4.16.0   4.19.2    minor │
│ ▸ personal/api    ⠹ scanning…                   ││ [ ] debug             4.4.1    4.4.3    patch │
│ ▸ sandbox              up to date · 0 vulns     ││                                              │
│                                                 ││ 2 of 4 marked · x clear                      │
│                                                 │└──────────────────────────────────────────────┘
│                                                 │┌─ Command · c copy ───────────────────────────┐
│                                                 ││ update  cd ~/work/api && ncu -u axios \      │
│                                                 ││         express && npm install               │
└─────────────────────────────────────────────────┘└──────────────────────────────────────────────┘
 ✓ copied to clipboard                    scanning 3/5   q quit · Tab pkgs · ? help
```

Changes visible above: focused panel border highlighted and numbered; per-source aggregate counters that survive folding; disambiguated counter labels (`maj/min/pat` vs `C/H/M/L`); spinner on in-flight sources; dedicated mark column that preserves severity colour; mark counter; compact three-item help bar with `?` for the rest; aggregate scan progress.

### 3.2 `?` keymap modal

```
┌─ Keybindings ────────────────────────────────────────────┐
│  Global                                                  │
│    q          quit             ?  keybindings            │
│    Tab        switch panel     H  about                  │
│    m          toggle messages  c  copy visible command   │
│                                                          │
│  Sources panel                                           │
│    ↑ ↓ / j k  move            Enter  fold / unfold       │
│    a          add path        d      remove path         │
│    r          rescan source   R      rescan all          │
│                                                          │
│  Packages panel                                          │
│    ↑ ↓ / j k  move            Space  mark / unmark       │
│    x          clear marks     /      filter              │
│    s          cycle sort      v      vulnerabilities     │
│                                                          │
│  Legend   maj / min / pat = semver bump                  │
│           C H M L          = critical high moderate low  │
│                                        Esc / ? to close  │
└──────────────────────────────────────────────────────────┘
```

### 3.3 Destructive-action confirmation

```
┌─ Remove path ────────────────────────────────────┐
│  Stop tracking /Users/me/work/api ?              │
│  The folder on disk is not touched.              │
│                                                  │
│              [ Enter remove ]   [ Esc cancel ]   │
└──────────────────────────────────────────────────┘
```

### 3.4 Empty first run

```
┌─ Sources ───────────────────────┐┌─ Packages ───────────────────────────────────┐
│ ▾ Global (npm -g)  maj 2 min 1  ││                                              │
│                                 ││   No project paths registered yet.           │
│   No paths registered.          ││                                              │
│   Press  a  to add one.         ││   Press  a  and enter a folder — a single    │
│                                 ││   project, a monorepo, or a folder of        │
│                                 ││   projects. Detection is automatic.          │
│                                 ││                                              │
│                                 ││   Press  ?  for all keybindings.             │
└─────────────────────────────────┘└──────────────────────────────────────────────┘
```

## 4. Improvement backlog

Impact = user-visible value. Effort: **S** ≤ half a day, **M** ≈ 1–2 days, **L** > 2 days (all within `ui/` unless noted).

### P0 — safety, discoverability, correctness of encoding

| ID | Improvement | Area | Impact | Effort | Notes |
|----|-------------|------|--------|--------|-------|
| UX-01 | Scope keys to the focused panel: `a`/`d`/`r`/`Enter` only in the tree, `Space`/`x`/`/`/`s` only in the table; `q`/`c`/`v`/`m`/`?`/`Tab` stay global | Keymap | High | S | Fixes O-06; the help bar then matches reality · **Shipped** (`add-keymap-safety`) |
| UX-02 | Confirmation modal before removing a registered path, stating that the folder on disk is untouched | Safety | High | S | Fixes O-07 · §3.3 · **Shipped** (`add-keymap-safety`) |
| UX-03 | `?` keymap modal, grouped by panel, including the counter legend; move About to `H` | Discoverability | High | M | Fixes O-08, O-18 · §3.2 · **Shipped** (`add-discoverability`) |
| UX-04 | Responsive bottom bar: compact help variants by terminal width, drop the hardcoded 106 columns, minimum-size notice below ~60 cols | Layout | High | M | Fixes O-01 |
| UX-05 | Aggregate counters on every source node, not just global; keep them visible when folded | Info scent | High | S | Fixes O-19 — counters already exist in `semver.Counters` · **Shipped** (`add-discoverability`) |
| UX-06 | First-run empty state in both panels with the `a` call to action | Onboarding | High | S | Fixes O-25 · §3.4 · **Shipped** (`add-discoverability`) |
| UX-07 | Disambiguate counters: `maj/min/pat` for semver, `C/H/M/L` reserved for audit, plus the legend in `?` | Encoding | High | S | Fixes O-18 · **Shipped** (`add-keymap-safety`) |
| UX-08 | Dedicated fixed-width mark column `[✓]`; the package name keeps its severity colour | Encoding | Med | S | Fixes O-21 |

### P1 — feedback, orientation, density

| ID | Improvement | Area | Impact | Effort | Notes |
|----|-------------|------|--------|--------|-------|
| UX-09 | Highlight the focused panel: border colour plus a number in the title (`1 Sources`, `2 Packages`) | Focus | High | S | Fixes O-03, O-05 · **Shipped** (`add-discoverability`) |
| UX-10 | Braille spinner on in-flight sources, driven by a ticker through the existing `QueueUpdateDraw` choke point | Feedback | High | M | Fixes O-13 |
| UX-11 | Aggregate scan progress (`scanning 3/5`) in the bottom bar, cleared on completion | Feedback | Med | S | Fixes O-14 |
| UX-12 | Auto-expiring status messages (~5 s) with a level (`info`/`ok`/`warn`/`error`) and an icon prefix | Feedback | Med | M | Fixes O-15, O-16 |
| UX-13 | Mark counter in the table footer or title (`2 of 4 marked`) | Density | Med | S | Fixes O-22 |
| UX-14 | Sort in the packages table (`s` cycles severity → name → current version) | Density | Med | M | Fixes O-12 |
| UX-15 | Incremental filter (`/`) over the visible table, `Esc` clears | Density | Med | M | Fixes O-12 |
| UX-16 | `R` rescans every source, respecting the in-flight guard | Keymap | Med | S | Fixes O-11 |
| UX-17 | Column sizing: expansion only on `Package` and `Via`, fixed widths for versions and severity, middle-ellipsis on overflow | Density | Med | M | Fixes O-23 |
| UX-18 | Command bar: shrink to 3 rows, show `… +N more` when the command exceeds the box; copying always yields the full command | Layout | Med | M | Fixes O-02 |
| UX-19 | Failed scans: `press r to retry` hint plus the full error reachable in a modal | Errors | Med | M | Fixes O-17 |
| UX-20 | `Esc` in the tree collapses the current source, then falls back to a no-op message instead of silence | Keymap | Low | S | Fixes O-09 |

### P2 — accessibility, polish, persistence

| ID | Improvement | Area | Impact | Effort | Notes |
|----|-------------|------|--------|--------|-------|
| UX-21 | Symbol backup for severity (`▲ major`, `● minor`, `· patch`) and `NO_COLOR` / monochrome mode | A11y | High | M | Fixes O-24 |
| UX-22 | Configurable theme in `config.toml` (`[theme]` with named colours), defaulting to the current palette | A11y | Low | L | Complements UX-21 |
| UX-23 | Disambiguate same-basename sources by showing the shortened parent (`work/api`, `personal/api`) | Info scent | Med | S | Fixes O-20 |
| UX-24 | Mouse: click a row to mark, click a header to sort; mention the mouse in `?` | Input | Low | M | Fixes O-27 |
| UX-25 | Persist UI preferences (fold state per source, `msgsHidden`) in `config.toml` | Persistence | Low | M | Requires a `config` change; keep it backward compatible |
| UX-26 | Flash the command-bar border on copy, in addition to the status message | Polish | Low | S | Reinforces the one truly rewarding action |

### Suggested sequencing

1. **Safety and truthfulness first** — UX-01, UX-02, UX-07 (the app currently allows an invisible destructive key and shows two meanings for `M`).
2. **Make the app self-explanatory** — UX-03, UX-06, UX-05, UX-09.
3. **Make waiting legible** — UX-10, UX-11, UX-12.
4. **Then density and accessibility** — UX-14/15/17, UX-21.

UX-01 through UX-09 together are roughly a two-to-three day change and cover every P0 finding.

---

# Español

## 1. Resumen ejecutivo

lazyncu ya resuelve lo difícil: una sola pantalla responde "qué proyectos necesitan updates y cuán urgente es". La arquitectura (`ui/` delgada sobre paquetes core puros) hace que todas las mejoras de abajo sean baratas de implementar.

Las debilidades no están en lo que la app muestra, sino en **lo que no le dice al usuario**:

| Tema | Diagnóstico |
|------|-------------|
| Descubribilidad | El keymap completo vive en el README, no en la app. No hay `?`. No hay guía de primer arranque. |
| Modelo de foco | Dos paneles enfocables, pero el foco sólo se señala cambiando la barra de ayuda — sin ancla visual. |
| Seguridad de teclas | Toda tecla global dispara sin importar el foco; `d` borra un path registrado sin confirmación ni undo. |
| Rastro de información | Los nodos de source registrados muestran sólo el nombre de la carpeta — señal cero. Sólo el nodo global resume. |
| Codificación | La severidad viaja en color + una letra ambigua. `M` es *major* a la izquierda de la fila y *moderate* a la derecha de la misma fila. |
| Responsividad | La barra de ayuda tiene 106 columnas hardcodeadas. Bajo ~110 columnas la zona de mensajes colapsa y la ayuda se corta. |

23 mejoras listadas en §4, agrupadas P0/P1/P2.

## 2. Observaciones por área

### 2.1 Layout y modelo espacial

`ui/layout.go:58-78` arma: árbol (flex 1) | [tabla de paquetes (flex 1) + barra de comando (4 filas fijas)] (flex 2), más una barra inferior de 1 fila.

- **O-01 — Barra de ayuda de ancho fijo.** `helpWidth = 106` (`ui/layout.go:15`) es una asignación rígida. `statusHelpTree` mide ~100 columnas imprimibles. En una terminal de 80 columnas la zona de mensajes queda en 0 y la propia ayuda se corta. No hay guarda de tamaño mínimo ni variante compacta.
- **O-02 — La barra de comando es fija en 4 filas** (`cmdBarRows`, `ui/layout.go:16`) pero renderiza como máximo 2 líneas. Dos filas de espacio vertical desperdiciadas en la zona más densa; a la inversa, un comando filtrado con muchos paquetes marcados hace wrap y puede exceder 4 filas sin indicador de overflow.
- **O-03 — No hay affordance de foco.** Nada cambia color de borde ni título al mover el foco (`setTableFocus`, `ui/input.go:79-91`). Las únicas señales son el texto de la ayuda y la selectabilidad de filas. En una TUI de dos paneles el panel enfocado debe ser inconfundible de un vistazo — lazygit y k9s recolorean el borde.
- **O-04 — Los paneles no son direccionables.** No hay convención `1`/`2` (lazygit); `Tab` es la única forma de mover foco y además es un toggle, así que no escala si aparece un tercer panel.
- **O-05 — El título del panel Sources no lleva pista** mientras los otros dos sí (`" Packages (v for vulnerabilities) "`, `" Command (copy with c) "`). El patrón es bueno y está aplicado de forma inconsistente.

### 2.2 Keymap e interacción

- **O-06 — Las teclas globales ignoran el foco.** `handleKey` (`ui/input.go:35-73`) despacha `q c v r a d m h` incondicionalmente. Con la tabla de paquetes enfocada, `d` igual elimina el path registrado — aunque `statusHelpTable` (`ui/layout.go:14`) ni siquiera lista `d`. Una tecla invisible en el contexto actual pero igualmente destructiva es el peor defecto de interacción de la app.
- **O-07 — Acción destructiva sin confirmación.** `removeSelectedPath` (`ui/input.go:211-228`) desregistra el path y persiste el config de inmediato. Sin confirmación, sin undo, sin "presioná d de nuevo para confirmar". El radio de daño es chico (una entrada de config), la sorpresa no.
- **O-08 — No hay cheat sheet en la app.** No existe `?`. `h` abre About (`ui/input.go:52`), doblemente poco convencional: `?` es la tecla de ayuda casi universal en TUIs, y `h` es "izquierda" en vim. El modal About (12 filas, `ui/about.go:14`) está casi vacío y alojaría el keymap sin problema.
- **O-09 — `Esc` es inconsistente.** Cierra el modal About y sale de la tabla (`ui/input.go:19-27`), pero no hace nada en el árbol. El usuario aprende "Esc = volver un nivel"; acá el último nivel queda mudo.
- **O-10 — Marcar es difícil de descubrir.** `Space` sólo funciona con la tabla enfocada (`ui/input.go:61-65`), lo que exige ya conocer `Tab`. Nada en el contexto del árbol sugiere que los paquetes son seleccionables.
- **O-11 — No hay rescan-all.** `r` reescanea sólo el source seleccionado (`rescanSelected`, `ui/input.go:142`). Después de un `git pull` sobre un workspace hay que visitar source por source.
- **O-12 — No hay búsqueda/filtro ni ordenamiento** en ningún lado. Las tablas se renderizan en orden de scan (`renderPackages`, `ui/detail.go:57`). Con 40 paquetes desactualizados no hay forma de ver primero los majors ni saltar a un nombre.

### 2.3 Feedback, estado y tiempo

- **O-13 — El loading es un string estático.** `"scanning…"` en la tabla, `"[gray]scanning…[-]"` en el árbol (`ui/panel.go:64-65`). Sin spinner, una llamada lenta al registry es indistinguible de una app colgada. `ncu` sobre varios proyectos tarda decenas de segundos, con timeout de 30 s por comando (`config.DefaultTimeoutMS`).
- **O-14 — No hay progreso agregado.** Los sources escanean en paralelo y llegan en streaming, pero nada muestra "4/9 sources listos". Hay que leer el árbol fila por fila para saber si el scan inicial terminó.
- **O-15 — Los mensajes de estado nunca expiran.** `setStatus` (`ui/app.go:183`) escribe y el texto queda hasta el próximo mensaje. Diez minutos después el usuario sigue viendo `copied: cd /x && ncu -u && npm install` sin saber si corresponde a la selección actual. `m` esconde toda la zona, que es un remedio grueso para un problema de obsolescencia.
- **O-16 — La severidad del mensaje es ad-hoc.** Los tags de color se escriben inline en cada call site (`[red]`, `[green]`, `[yellow]`) en vez de derivarse de un nivel. No hay prefijo de ícono: en una terminal monocroma un error se lee igual que una confirmación.
- **O-17 — Los errores de scan son terminales.** Un source fallido muestra `✗ scan failed` en el árbol y `scan failed: <err>` en la tabla (`ui/detail.go:41-43`). El mensaje no es scrolleable ni copiable, y nada sugiere que `r` reintenta.

### 2.4 Codificación de información y densidad

- **O-18 — Colisión de letras en la misma línea.** Una fila de proyecto se lee `3M 5m 2p │ 1C 2H 3M` (`updateSummary` / `auditSummary`, `ui/panel.go:82-126`). La `M` izquierda es *major*, la `M` derecha es *moderate*. Dos taxonomías, un glifo, una línea, y ninguna leyenda en la UI.
- **O-19 — Los sources registrados no llevan resumen.** `sourceText` devuelve el `filepath.Base` pelado para cualquier source no global (`ui/panel.go:71-72`). El nodo global muestra contadores; un path registrado no. Al plegarlo (`Enter`) la fila queda como ruido puro — el gesto de fold destruye toda la información en vez de agregarla.
- **O-20 — Sources con el mismo basename son indistinguibles.** `~/work/api` y `~/personal/api` se ven ambos como `api`.
- **O-21 — Marcar pisa el color de severidad.** `renderPackages` (`ui/detail.go:62-65`) reemplaza la celda 0 por un `✓ nombre` amarillo, así que un paquete marcado pierde su color major/minor/patch en el nombre mientras el resto de las celdas lo conserva — fila medio coloreada. El prefijo `✓ ` además corre el texto y rompe la alineación izquierda de la columna.
- **O-22 — La cantidad de marcas es invisible.** La única evidencia de cuántos paquetes están seleccionados es leer el comando generado. No hay `3 seleccionados` en ningún lado, ni forma de ver marcas sin enfocar la tabla.
- **O-23 — Todas las columnas expanden igual.** `SetExpansion(1)` en cada celda (`ui/detail.go:203-212`) le da a `Severity` (≤5 chars) el mismo crecimiento que a `Package` (40+). En la vista de vulnerabilidades la cadena `Via` (`lodash ← express ← …`) es el campo más largo y el primero en cortarse, sin marcador de elipsis.
- **O-24 — El color es el canal primario.** La severidad no tiene respaldo de forma/símbolo (`severityColor`, `vulnColor`). Con `NO_COLOR`, perfil monocromo o daltonismo rojo/verde, las filas major y patch quedan idénticas.

### 2.5 Onboarding y primer arranque

- **O-25 — El primer arranque vacío es un callejón sin salida.** `config.Load` crea un config vacío en el primer launch (`config/config.go:57-63`). El usuario ve un árbol con un solo nodo `Global (npm -g)` y el panel derecho vacío. Nada dice "todavía no hay paths registrados — presioná `a`". La barra inferior sí lista `a add path`, pero es un ítem entre diez, en el extremo derecho de la pantalla.
- **O-26 — El fallo de preflight es pre-TUI.** Si falta `ncu` o es muy viejo, imprime a stderr y sale (`scanner.Preflight`, `main.go:54`). Es el comportamiento correcto y el mensaje ya incluye el comando de instalación — sin cambios, se lista por completitud.
- **O-27 — El mouse está habilitado pero indocumentado.** `EnableMouse(true)` (`ui/app.go:136`). Clickear navega, pero click-para-marcar y click-para-ordenar no existen, y ninguna affordance sugiere que el mouse funcione.

### 2.6 Lo que ya funciona bien (no romper)

- Invariante read-only, hecho visible: la barra de comando muestra exactamente lo que se va a correr y `c` lo copia. Es el diferenciador del producto.
- Títulos que enseñan (`" Packages (v for vulnerabilities) "`) — barato, contextual, efectivo.
- Zona de mensajes y zona de ayuda separadas, así un mensaje nunca tapa la ayuda (`ui/layout.go:66-70`). Muchas TUIs fallan en esto.
- Los tres estados de audit (`n/a` / `✗` / `0 vulns`) se mantienen visualmente distintos — "no auditado" no se reporta como "limpio".
- El rescan está protegido contra scans solapados, con mensaje explícito en vez de un no-op silencioso (`ui/input.go:148-151`).
- `m` para recuperar la zona de mensajes es un buen detalle de power user.

## 3. Pantallas objetivo propuestas

### 3.1 Pantalla principal

```
┌─ 1 Sources ────────── a add · d del · r rescan ─┐┌─ 2 Packages ─ / filtro · s orden · v vulns ───┐
│ ▾ Global (npm -g)      maj 2 min 1 · audit n/a  ││    Package          Current   New      Sev   │
│ ▾ work/api             maj 3 min 5 pat 2 · C1 H2││ [✓] axios            0.21.1   1.7.7    major │
│   ├ api-gateway        maj 1 min 2 · 0 vulns    ││ [ ] chalk             4.1.0    5.3.0    major │
│   └ api-workers        maj 2 min 3 pat 2 · C1 H2││ [✓] express          4.16.0   4.19.2    minor │
│ ▸ personal/api    ⠹ escaneando…                 ││ [ ] debug             4.4.1    4.4.3    patch │
│ ▸ sandbox              al día · 0 vulns         ││                                              │
│                                                 ││ 2 de 4 marcados · x limpiar                  │
│                                                 │└──────────────────────────────────────────────┘
│                                                 │┌─ Command · c copiar ─────────────────────────┐
│                                                 ││ update  cd ~/work/api && ncu -u axios \      │
│                                                 ││         express && npm install               │
└─────────────────────────────────────────────────┘└──────────────────────────────────────────────┘
 ✓ copiado al portapapeles              escaneando 3/5   q salir · Tab pkgs · ? ayuda
```

Cambios visibles arriba: borde del panel enfocado resaltado y numerado; contadores agregados por source que sobreviven al plegado; etiquetas de contador desambiguadas (`maj/min/pat` vs `C/H/M/L`); spinner en sources en vuelo; columna de marca dedicada que preserva el color de severidad; contador de marcas; barra de ayuda compacta de tres ítems con `?` para el resto; progreso agregado del scan.

### 3.2 Modal de keymap (`?`)

```
┌─ Keybindings ────────────────────────────────────────────┐
│  Global                                                  │
│    q          salir            ?  keybindings            │
│    Tab        cambiar panel    H  about                  │
│    m          toggle mensajes  c  copiar comando visible │
│                                                          │
│  Panel Sources                                           │
│    ↑ ↓ / j k  mover           Enter  plegar / desplegar  │
│    a          agregar path    d      quitar path         │
│    r          rescan source   R      rescan todo         │
│                                                          │
│  Panel Packages                                          │
│    ↑ ↓ / j k  mover           Space  marcar / desmarcar  │
│    x          limpiar marcas  /      filtrar             │
│    s          ciclar orden    v      vulnerabilidades    │
│                                                          │
│  Leyenda  maj / min / pat = salto semver                 │
│           C H M L          = critical high moderate low  │
│                                       Esc / ? para cerrar│
└──────────────────────────────────────────────────────────┘
```

### 3.3 Confirmación de acción destructiva

```
┌─ Quitar path ────────────────────────────────────┐
│  ¿Dejar de trackear /Users/me/work/api ?         │
│  La carpeta en disco no se toca.                 │
│                                                  │
│            [ Enter quitar ]   [ Esc cancelar ]   │
└──────────────────────────────────────────────────┘
```

### 3.4 Primer arranque vacío

```
┌─ Sources ───────────────────────┐┌─ Packages ───────────────────────────────────┐
│ ▾ Global (npm -g)  maj 2 min 1  ││                                              │
│                                 ││   Todavía no hay paths de proyecto.          │
│   Sin paths registrados.        ││                                              │
│   Presioná  a  para agregar.    ││   Presioná  a  e ingresá una carpeta: un     │
│                                 ││   proyecto, un monorepo o una carpeta de     │
│                                 ││   proyectos. La detección es automática.     │
│                                 ││                                              │
│                                 ││   Presioná  ?  para ver todas las teclas.    │
└─────────────────────────────────┘└──────────────────────────────────────────────┘
```

## 4. Backlog de mejoras

Impacto = valor visible para el usuario. Esfuerzo: **S** ≤ medio día, **M** ≈ 1–2 días, **L** > 2 días (todo dentro de `ui/` salvo indicación).

### P0 — seguridad, descubribilidad, corrección de la codificación

| ID | Mejora | Área | Impacto | Esfuerzo | Notas |
|----|--------|------|---------|----------|-------|
| UX-01 | Acotar teclas al panel enfocado: `a`/`d`/`r`/`Enter` sólo en el árbol, `Space`/`x`/`/`/`s` sólo en la tabla; `q`/`c`/`v`/`m`/`?`/`Tab` quedan globales | Keymap | Alto | S | Corrige O-06; la barra de ayuda pasa a decir la verdad · **Entregado** (`add-keymap-safety`) |
| UX-02 | Modal de confirmación antes de quitar un path registrado, aclarando que la carpeta en disco no se toca | Seguridad | Alto | S | Corrige O-07 · §3.3 · **Entregado** (`add-keymap-safety`) |
| UX-03 | Modal de keymap con `?`, agrupado por panel, incluyendo la leyenda de contadores; mover About a `H` | Descubribilidad | Alto | M | Corrige O-08, O-18 · §3.2 · **Entregado** (`add-discoverability`) |
| UX-04 | Barra inferior responsiva: variantes compactas de ayuda según ancho, eliminar las 106 columnas hardcodeadas, aviso de tamaño mínimo bajo ~60 cols | Layout | Alto | M | Corrige O-01 |
| UX-05 | Contadores agregados en todos los nodos source, no sólo el global; visibles también plegados | Rastro info | Alto | S | Corrige O-19 — los contadores ya existen en `semver.Counters` · **Entregado** (`add-discoverability`) |
| UX-06 | Estado vacío de primer arranque en ambos paneles con el call to action de `a` | Onboarding | Alto | S | Corrige O-25 · §3.4 · **Entregado** (`add-discoverability`) |
| UX-07 | Desambiguar contadores: `maj/min/pat` para semver, `C/H/M/L` reservado a audit, más la leyenda en `?` | Codificación | Alto | S | Corrige O-18 · **Entregado** (`add-keymap-safety`) |
| UX-08 | Columna de marca dedicada de ancho fijo `[✓]`; el nombre del paquete conserva su color de severidad | Codificación | Medio | S | Corrige O-21 |

### P1 — feedback, orientación, densidad

| ID | Mejora | Área | Impacto | Esfuerzo | Notas |
|----|--------|------|---------|----------|-------|
| UX-09 | Resaltar el panel enfocado: color de borde más número en el título (`1 Sources`, `2 Packages`) | Foco | Alto | S | Corrige O-03, O-05 · **Entregado** (`add-discoverability`) |
| UX-10 | Spinner braille en sources en vuelo, movido por un ticker a través del choke point `QueueUpdateDraw` existente | Feedback | Alto | M | Corrige O-13 |
| UX-11 | Progreso agregado del scan (`escaneando 3/5`) en la barra inferior, limpiado al terminar | Feedback | Medio | S | Corrige O-14 |
| UX-12 | Mensajes de estado con auto-expiración (~5 s), nivel (`info`/`ok`/`warn`/`error`) y prefijo de ícono | Feedback | Medio | M | Corrige O-15, O-16 |
| UX-13 | Contador de marcas en el pie o título de la tabla (`2 de 4 marcados`) | Densidad | Medio | S | Corrige O-22 |
| UX-14 | Ordenamiento en la tabla de paquetes (`s` cicla severidad → nombre → versión actual) | Densidad | Medio | M | Corrige O-12 |
| UX-15 | Filtro incremental (`/`) sobre la tabla visible, `Esc` limpia | Densidad | Medio | M | Corrige O-12 |
| UX-16 | `R` reescanea todos los sources, respetando la guarda de scan en vuelo | Keymap | Medio | S | Corrige O-11 |
| UX-17 | Dimensionado de columnas: expansión sólo en `Package` y `Via`, anchos fijos para versiones y severidad, elipsis al medio en overflow | Densidad | Medio | M | Corrige O-23 |
| UX-18 | Barra de comando: bajar a 3 filas y mostrar `… +N más` cuando el comando excede la caja; copiar siempre entrega el comando completo | Layout | Medio | M | Corrige O-02 |
| UX-19 | Scans fallidos: pista `presioná r para reintentar` más el error completo accesible en un modal | Errores | Medio | M | Corrige O-17 |
| UX-20 | `Esc` en el árbol pliega el source actual y, si no aplica, responde con un mensaje en vez de silencio | Keymap | Bajo | S | Corrige O-09 |

### P2 — accesibilidad, pulido, persistencia

| ID | Mejora | Área | Impacto | Esfuerzo | Notas |
|----|--------|------|---------|----------|-------|
| UX-21 | Respaldo simbólico de severidad (`▲ major`, `● minor`, `· patch`) y modo `NO_COLOR` / monocromo | A11y | Alto | M | Corrige O-24 |
| UX-22 | Tema configurable en `config.toml` (`[theme]` con colores nombrados), default = paleta actual | A11y | Bajo | L | Complementa UX-21 |
| UX-23 | Desambiguar sources con igual basename mostrando el padre acortado (`work/api`, `personal/api`) | Rastro info | Medio | S | Corrige O-20 |
| UX-24 | Mouse: click en fila para marcar, click en header para ordenar; mencionar el mouse en `?` | Input | Bajo | M | Corrige O-27 |
| UX-25 | Persistir preferencias de UI (estado de plegado por source, `msgsHidden`) en `config.toml` | Persistencia | Bajo | M | Requiere cambio en `config`; mantener compatibilidad hacia atrás |
| UX-26 | Flash del borde de la barra de comando al copiar, además del mensaje de estado | Pulido | Bajo | S | Refuerza la única acción realmente gratificante |

### Secuencia sugerida

1. **Primero seguridad y veracidad** — UX-01, UX-02, UX-07 (hoy la app permite una tecla destructiva invisible y muestra dos significados para `M`).
2. **Hacer la app autoexplicativa** — UX-03, UX-06, UX-05, UX-09.
3. **Hacer legible la espera** — UX-10, UX-11, UX-12.
4. **Después densidad y accesibilidad** — UX-14/15/17, UX-21.

UX-01 a UX-09 juntas son aproximadamente un cambio de dos a tres días y cubren todos los hallazgos P0.
