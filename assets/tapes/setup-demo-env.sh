#!/usr/bin/env bash
# Prepares an isolated environment for VHS demo recordings:
# - copies fixtures to /tmp so GIFs never show personal paths
# - seeds throwaway lazyncu configs (recordings never touch ~/.config/lazyncu)
# - builds the binary the tapes launch
# npm audit only needs the committed lockfiles, so fixtures are not installed.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DEMO_HOME="/tmp/lazyncu-demo"

rm -rf "$DEMO_HOME"
mkdir -p "$DEMO_HOME/fixtures" "$DEMO_HOME/config/lazyncu"

cp -R "$ROOT/demo/fixtures/webapp" "$ROOT/demo/fixtures/tools" "$DEMO_HOME/fixtures/"

cat > "$DEMO_HOME/config-full.toml" <<EOF
timeout_ms = 60000

[[paths]]
path = "$DEMO_HOME/fixtures/webapp"

[[paths]]
path = "$DEMO_HOME/fixtures/tools"
EOF

cat > "$DEMO_HOME/config-addpath.toml" <<EOF
timeout_ms = 60000

[[paths]]
path = "$DEMO_HOME/fixtures/webapp"
EOF

# Tapes copy the variant they need over this file before launching.
cp "$DEMO_HOME/config-full.toml" "$DEMO_HOME/config/lazyncu/config.toml"

make -C "$ROOT" build

echo "demo env ready at $DEMO_HOME"
