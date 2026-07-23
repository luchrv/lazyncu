# demo-recording Specification (delta)

## ADDED Requirements

### Requirement: Demos are recorded from versioned VHS tapes
The repository SHALL contain VHS tape files under `assets/tapes/` that render the demo GIFs under `assets/demo/`, and a `make demos` target SHALL regenerate every GIF from its tape in one command. Tapes SHALL pin terminal dimensions, font, and theme so output is consistent across machines.

#### Scenario: Regenerating all demos
- **WHEN** a contributor with VHS installed runs `make demos`
- **THEN** every tape in `assets/tapes/` renders its GIF into `assets/demo/`, overwriting stale versions

### Requirement: Recordings are isolated from the user's real configuration
Demo recording SHALL run lazyncu with a temporary `XDG_CONFIG_HOME` seeded with a config that registers only the demo fixtures, so recordings never read or modify `~/.config/lazyncu` and GIFs never show personal paths.

#### Scenario: Recording on a developer machine
- **WHEN** `make demos` runs on a machine with an existing `~/.config/lazyncu/config.toml`
- **THEN** the recording uses only fixture paths and the user's real config is untouched

### Requirement: Synthetic fixtures drive the demo content
The repository SHALL contain npm fixture projects under `demo/fixtures/` with committed lockfiles, pinned outdated dependencies spanning major/minor/patch severities, and at least one dependency with known published vulnerabilities so the vulnerability view shows severity counters and a dependency chain.

#### Scenario: Fixture scan yields demo-worthy output
- **WHEN** lazyncu scans the fixtures during recording
- **THEN** the dashboard shows pending updates in all three severities and the vulnerability view shows at least one vulnerable package with its dependency chain

### Requirement: README presents badges and the three demos
The README SHALL show a badge row at the top (release version, Go version, license, release downloads, Homebrew tap) and SHALL embed three GIFs: a hero demo after the intro, the vulnerability view demo in the audit section, and the add-path demo in the configuration section.

#### Scenario: Visitor opens the repository page
- **WHEN** a visitor views the README on GitHub
- **THEN** badges render at the top and the hero GIF plays immediately after the intro bullets

### Requirement: Demo workflow is documented for contributors
The repository SHALL include `docs/DEMOS.md` covering: installing VHS, tape file anatomy, regenerating all demos, adding a new demo, and the network requirement for recording.

#### Scenario: Contributor adds a new demo
- **WHEN** a contributor follows `docs/DEMOS.md` to create a tape for a new feature
- **THEN** they can produce a consistent GIF and reference it from the README without help
