# Interactive Processes

## Description

Interactive processes allow you to interact directly with a running process through the TUI. This is useful for processes that require user input, such as a REPL, a debugger, or a command-line editor.

When a process is configured as interactive, Process Compose allocates a pseudo-terminal (PTY) for it and allows you to attach to its standard input and output.

## Configuration

To enable interactive mode for a process, set `is_interactive: true` in your `process-compose.yaml` configuration.

```yaml
processes:
  my-interactive-process:
    command: "python3"
    is_interactive: true
```

## Usage

### Attaching to a Process

In the TUI, select the interactive process from the list. The process output will be displayed in the logs view.

### Switching Focus

To interact with the process, you need to switch focus to the terminal view.

- **Switch Focus**: Press `TAB` to switch focus from the process list/logs to the interactive terminal.
- **Exit Focus**: Press `CTRL+A` followed by `ESC` to release focus from the interactive terminal and return to the TUI navigation.

### Keybindings

| Key Combination | Action |
| :--- | :--- |
| `TAB` | Switch focus to the interactive process terminal (Input Mode). |
| `CTRL+A` followed by `ESC` | Exit Input Mode and return to TUI navigation (Normal Mode). |
