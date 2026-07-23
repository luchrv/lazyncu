# Vulnerability view demo: toggle with v, severity counters, dependency chain.
# Run via `make demos` (needs assets/tapes/setup-demo-env.sh first).

Output assets/demo/vulns.gif

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

# Wait for scans and audits to finish.
Sleep 12s

# Select the webapp project (it has 9 vulnerabilities with chains).
Down
Sleep 2s

# Toggle the vulnerability detail view.
Type "v"
Sleep 5s

# Back to the package view.
Type "v"
Sleep 2s

Type "q"
Sleep 1s
