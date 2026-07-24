# Package-selection demo: Tab into the table, mark packages with space,
# the update command narrows to the marked ones.
# Run via `make demos` (needs assets/tapes/setup-demo-env.sh first).

Output assets/demo/select.gif

Set FontSize 15
Set Width 1200
Set Height 700
Set Theme "Dracula"
Set TypingSpeed 60ms
Set Padding 12

Hide
Type "cp /tmp/lazyncu-demo/config-full.toml /tmp/lazyncu-demo/config/lazyncu/config.toml && export XDG_CONFIG_HOME=/tmp/lazyncu-demo/config && export PATH=$PWD:$PATH && clear"
Enter
Show

Type "lazyncu"
Sleep 500ms
Enter

# Wait for the parallel scans to finish.
Sleep 12s

# Select the webapp project and focus the packages table.
Down
Sleep 1.5s
Tab
Sleep 1.5s

# Mark axios, then commander — watch the update command narrow.
Space
Sleep 2s
Down
Sleep 800ms
Down
Sleep 800ms
Space
Sleep 4s

# Clear the marks and go back to the tree.
Type "x"
Sleep 2s
Escape
Sleep 1.5s

Type "q"
Sleep 1s
