# Hero demo: launch, parallel scan streaming in, navigate sources.
# Run via `make demos` (needs assets/tapes/setup-demo-env.sh first).

Output assets/demo/hero.gif

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

# Sources scan in parallel; results stream in as each finishes.
Sleep 12s

# Walk the tree: global -> webapp -> tools.
Down
Sleep 2.5s
Down
Sleep 2.5s
Up
Sleep 2s

Type "q"
Sleep 1s
