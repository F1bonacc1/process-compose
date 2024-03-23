# TUI (Terminal User Interface)

TUI Allows you to:

- Review processes status
- Start processes (only completed or disabled)
- Stop processes
- Review logs
- Restart running processes

TUI is the default run mode, but it's possible to disable it:

```shell
./process-compose -t=false
```

Control the UI log buffer size:

```yaml
log_level: info
log_length: 1200 #default: 1000
processes:
  process2:
    command: "ls -R /"
```

> :bulb: Using a too large buffer will put more penalty on your CPU.

By default `process-compose` uses the standard ANSI colors mode to display logs. However, you can disable it for each process:

```yaml
processes:
  process_name:
    command: "ls -R /"
    disable_ansi_colors: true #default false
```

> :bulb: Too long log lines (above 2^16 bytes long) can cause the log collector to hang.

## Shortcuts Configuration

Default shortcuts can be changed by placing `shortcuts.yaml` in your `$XDG_CONFIG_HOME/process-compose/` directory.  
The default `process-compose` configuration is defined as:

```yaml
# $XDG_CONFIG_HOME/process-compose/shortcuts.yaml
shortcuts:
  log_follow: # action name - don't edit
    toggle_description: # optional description for toggle buttons. Will use default if not defined
      false: Follow Off
      true: Follow On
    shortcut: F5 # shortcut to be used
  log_screen:
    toggle_description:
      false: Half Screen
      true: Full Screen
    shortcut: F4
  log_wrap:
    toggle_description:
      false: Wrap Off
      true: Wrap On
    shortcut: F6
  process_restart:
    description: Restart # optional description for a button. Will use default if not defined
    shortcut: Ctrl-R
  process_screen:
    toggle_description:
      false: Half Screen
      true: Full Screen
    shortcut: F8
  process_start:
    description: Start
    shortcut: F7
  process_stop:
    description: Stop
    shortcut: F9
  quit:
    description: Quit
    shortcut: F10
```

`shortcuts.yaml` can contain only the values you wish to change, default values will be used for the rest.  
For example if you want to replace the default `quit` shortcut to be `F3` instead of `F10` and rename the `process_stop` to be `Terminate`, the configurion will be as follows:
```yaml
# $XDG_CONFIG_HOME/process-compose/shortcuts.yaml
shortcuts:
  process_stop:
    description: Terminate
  quit:
    shortcut: F3
```

## TUI Themes

The default shortcut for theme selection is `CTRL-T`. Process Compose comes with 4 pre-loaded themes.  These can be extended in 2 ways:

1. By contributing a new theme, by creating a PR with a new theme in the `src/config/themes` [directory](https://github.com/F1bonacc1/process-compose/tree/main/src/config/themes).
2. Adding your own theme by placing `theme.yaml` in your `$XDG_CONFIG_HOME/process-compose/` directory.

The default `process-compose` theme is defined as:

```yaml
style:
  body:
    fgColor: white
    bgColor: black
    secondaryTextColor: yellow
    tertiaryTextColor: green
    borderColor: white
  stat_table:
    keyFgColor: yellow
    valueFgColor: white
    logoColor: yellow
  proc_table:
    fgColor: lightskyblue
    fgWarning: yellow
    fgPending: grey
    fgCompleted: lightgreen
    fgError: red
    headerFgColor: white
  help:
    fgColor: black
    keyColor: white
    hlColor: green
    categoryFgColor: lightskyblue
  dialog:
    fgColor: cadetblue
    bgColor: black
    buttonFgColor: black
    buttonBgColor: lightskyblue
    buttonFocusFgColor: black
    buttonFocusBgColor: dodgerblue
    labelFgColor: yellow
    fieldFgColor: black
    fieldBgColor: lightskyblue
```

`theme.yaml` can contain only the values you wish to change, default values will be used for the rest.  
For example if you want to change the default background color to `green` instead of `black` and the logo color to `blue`, the configuration will be as follows:

```yaml
style:
  body:
    bgColor: green
  stat_table:
    logoColor: blue
```

> :bulb: To apply the new values it's enough to select the `Custom Style` (`F` shortcut) theme from the theme selector menu (`CTRL-T`).

For Color names W3C approved color names should be used. Note that on various terminals colors may be approximated, or not supported at all.  If no suitable representation for a color is known, the no color will be set, deferring to whatever default attributes the terminal uses.

By default `process-compose` will respect any theme used in your terminal which might result in less accurate color fidelity. To force the use of "True Colors", please use the `HEX` color notation:

```yaml
style:
  body:
    bgColor: '#00FF00' #green
  stat_table:
    logoColor: '#0000FF' #blue
```

