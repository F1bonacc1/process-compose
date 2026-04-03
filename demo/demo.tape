Output imgs/demo.gif
Set Shell zsh
Set FontSize 14
Set Width 1400
Set Height 800
Set Padding 10
Set TypingSpeed 60ms
Set Theme "Catppuccin Mocha"
Set CursorBlink false

Type "process-compose -f demo/process-compose.yaml"
Enter

# Wait for processes to start and health checks to pass
Sleep 5s

# Navigate down through the process list once
Down
Sleep 1s
Down
Sleep 1s
Down
Sleep 1s
Down
Sleep 1s
Down
Sleep 1s

Sleep 2s

# Search for the interactive debug-shell process
Type "/"
Sleep 0.5s
Type "lazygit"
Enter
Sleep 1.5s

# Tab to focus the interactive terminal
Tab
Sleep 1s

Tab
Sleep 0.5s
Down
Sleep 0.5s
Down
Sleep 0.5s

# Press Tab to go back to the process table
Ctrl+A
Escape
Escape
Sleep 2s

# Navigate back up to see all processes running
Down
Sleep 1s
Down
Sleep 1s
Down
Sleep 1s
Down
Sleep 1s

