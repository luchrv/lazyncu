# Proposal: add-node-version-awareness

## Why

ncu happily suggests upgrades whose `engines.node` the project cannot satisfy — e.g. `pkg@9` requiring node >= 20 for a project pinned to node 16 — so the dashboard shows updates that fail at install time. lazyncu also gives no visibility into which node version each project targets, even when the project declares it via `.nvmrc` or `engines.node`.

Exploration debunked the original framing (running ncu under the project's `.nvmrc` node via nvm/fnm): ncu's output does not depend on the node version executing it, so that route adds high complexity for zero benefit. The real levers are ncu's `--enginesNode` flag (filters suggestions by the project's declared `engines.node`) and surfacing the node context in the UI.

## What Changes

- Project scans (single and deep) pass `--enginesNode` to ncu, so suggested upgrades are limited to versions the project's declared `engines.node` supports. Projects without `engines.node` see unchanged results (the flag is a no-op for them). The global scan (`-g`) is unaffected — there is no project manifest.
- Scan results capture each project's node context: the `.nvmrc` content and the `engines.node` constraint, when present.
- The detail panel title shows the selected project's node context (e.g. `node 18 (.nvmrc)` or `node >=18 (engines)`), following the existing "titles that teach" convention; omitted when the project declares neither.

Out of scope (explicitly rejected during exploration):
- Executing ncu under the project's `.nvmrc` node version (nvm/fnm/asdf/volta orchestration) — no effect on results.
- Marking individual incompatible packages in the table — requires per-package registry `engines` lookups.
- Any new configuration; the behavior is automatic.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `package-scanning`: single and deep scan requirements gain engines-aware filtering (`--enginesNode`); a new requirement captures per-project node context (`.nvmrc`, `engines.node`) in scan results.
- `dashboard-ui`: the detail panel title additionally surfaces the selected project's node context.

## Impact

- `detect`: new reader for a project's node context (`.nvmrc`, `engines.node`).
- `scanner`: `--enginesNode` on project scan invocations; `Project` struct gains node-context fields.
- `ui`: `detailTitle` composition and its call site.
- Behavior change for users: projects with `engines.node` may show fewer (now actually installable) upgrades than before.
