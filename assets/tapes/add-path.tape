# Add-path demo: press a, type a path, the new source scans immediately.
# Run via `make demos` (needs assets/tapes/setup-demo-env.sh first).

Output assets/demo/add-path.gif

Set FontSize 15
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set TypingSpeed 60ms
Set Padding 12

Hide
Type "cp /tmp/lazyncu-demo/config-addpath.toml /tmp/lazyncu-demo/config/lazyncu/config.toml && export XDG_CONFIG_HOME=/tmp/lazyncu-demo/config && export PATH=$PWD:$PATH && clear"
Enter
Show

Type "lazyncu"
Sleep 500ms
Enter

# Initial scan: global + webapp only.
Sleep 10s

# Register a new path from the UI.
Type "a"
Sleep 1s
Type@35ms "/tmp/lazyncu-demo/fixtures/tools"
Sleep 1s
Enter

# The new source is persisted and scanned immediately.
Sleep 8s
Down
Sleep 1.5s
Down
Sleep 2.5s

Type "q"
Sleep 1s
