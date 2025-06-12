## process-compose

Processes scheduler and orchestrator

```
process-compose [flags]
```

### Options

```
  -f, --config stringArray       path to config files to load (env: PC_CONFIG_FILES)
      --detach-on-success        detach the process-compose TUI after successful startup. Requires --detached-with-tui
  -D, --detached                 run process-compose in detached mode
      --detached-with-tui        run process-compose in detached mode with TUI
      --disable-dotenv           disable .env file loading (env: PC_DISABLE_DOTENV=1)
  -e, --env stringArray          path to env files to load (default [.env])
  -h, --help                     help for process-compose
  -d, --hide-disabled            hide disabled processes (env: PC_HIDE_DISABLED_PROC)
      --keep-project             keep the project running even after all processes exit
  -L, --log-file string          Specify the log file path (env: PC_LOG_FILE) (default "/tmp/process-compose-<user>.log")
      --logs-truncate            truncate process logs buffer on startup
  -n, --namespace stringArray    run only specified namespaces (default all)
      --no-server                disable HTTP server (env: PC_NO_SERVER)
      --ordered-shutdown         shut down processes in reverse dependency order
  -p, --port int                 port number (env: PC_PORT_NUM) (default 8080)
      --read-only                enable read-only mode (env: PC_READ_ONLY)
      --recursive-metrics        collect metrics recursively (env: PC_RECURSIVE_METRICS)
  -r, --ref-rate duration        TUI refresh interval in seconds or as a Go duration string (e.g. 1s) (default 1)
  -R, --reverse                  sort in reverse order
      --shortcuts stringArray    paths to shortcut config files to load (env: PC_SHORTCUTS_FILES) (default [/home/<user>/.config/process-compose/shortcuts.yml])
      --slow-ref-rate duration   Slow(er) refresh interval for resources (CPU, RAM) in seconds or as a Go duration string (e.g. 1s). The value should be higher than --ref-rate (default 1)
  -S, --sort string              sort column name. legal values (case insensitive): [AGE, CPU, EXIT, HEALTH, MEM, NAME, NAMESPACE, PID, RESTARTS, STATUS] (default "NAME")
      --theme string             select process compose theme (default "Default")
  -t, --tui                      enable TUI (disable with -t=false) (env: PC_DISABLE_TUI) (default true)
      --tui-fs                   enable TUI full screen (env: PC_TUI_FULL_SCREEN=1)
  -u, --unix-socket string       path to unix socket (env: PC_SOCKET_PATH) (default "/tmp/process-compose-<pid>.sock")
  -U, --use-uds                  use unix domain sockets instead of tcp
```

### SEE ALSO

* [process-compose attach](process-compose_attach.md)	 - Attach the Process Compose TUI Remotely to a Running Process Compose Server
* [process-compose completion](process-compose_completion.md)	 - Generate the autocompletion script for the specified shell
* [process-compose down](process-compose_down.md)	 - Stops all the running processes and terminates the Process Compose
* [process-compose info](process-compose_info.md)	 - Print configuration info
* [process-compose list](process-compose_list.md)	 - List available processes
* [process-compose process](process-compose_process.md)	 - Execute operations on the available processes
* [process-compose project](process-compose_project.md)	 - Execute operations on a running Process Compose project
* [process-compose run](process-compose_run.md)	 - Run PROCESS in the foreground, and its dependencies in the background
* [process-compose up](process-compose_up.md)	 - Run process compose project
* [process-compose version](process-compose_version.md)	 - Print version and build info

