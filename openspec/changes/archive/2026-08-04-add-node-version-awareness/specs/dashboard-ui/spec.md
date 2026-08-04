# dashboard-ui Delta Specification

## ADDED Requirements

### Requirement: Detail panel surfaces the selected project's node context
When the selection resolves to a project that declares a node version, the detail panel title SHALL include a node-context segment: the `.nvmrc` value labeled `(.nvmrc)` when present, otherwise the `engines.node` constraint labeled `(engines)`. The segment SHALL be omitted entirely when the project declares neither, and for the global source. Existing title segments (view name, mark counter, sort, filter) keep their positions; the node segment never displaces them.

#### Scenario: Project pinned via .nvmrc
- **WHEN** the selected project has `.nvmrc` content `18.19.0`
- **THEN** the detail panel title includes `node 18.19.0 (.nvmrc)`

#### Scenario: Project with engines.node only
- **WHEN** the selected project declares `engines.node: ">=18"` and has no `.nvmrc`
- **THEN** the detail panel title includes `node >=18 (engines)`

#### Scenario: Project without node declarations
- **WHEN** the selected project declares neither `.nvmrc` nor `engines.node`
- **THEN** the detail panel title shows no node segment

#### Scenario: Global source selected
- **WHEN** the global source is selected
- **THEN** the detail panel title shows no node segment
