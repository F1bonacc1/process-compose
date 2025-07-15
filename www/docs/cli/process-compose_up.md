## process-compose up

Run process compose project

### Synopsis

Run all the process compose processes.
If one or more process names are passed as arguments,
will start them and their dependencies only

```
process-compose up [PROCESS...] [flags]
```

### Options

```
  -f, --config stringArray       path to config files to load (env: PC_CONFIG_FILES)
      --detach-on-success        detach the process-compose TUI after successful startup. Requires --detached-with-tui
  -D, --detached                 run process-compose in detached mode
      --detached-with-tui        run process-compose in detached mode with TUI
      --disable-dotenv           disable .env file loading (env: PC_DISABLE_DOTENV=1)
      --dry-run                  validate the config and exit
  -e, --env stringArray          path to env files to load (default [.env])
  -h, --help                     help for up
  -d, --hide-disabled            hide disabled processes (env: PC_HIDE_DISABLED_PROC)
      --keep-project             keep the project running even after all processes exit
      --logs-truncate            truncate process logs buffer on startup
  -n, --namespace stringArray    run only specified namespaces (default all)
      --no-deps                  don't start dependent processes
      --recursive-metrics        collect metrics recursively (env: PC_RECURSIVE_METRICS)
  -r, --ref-rate duration        TUI refresh interval in seconds or as a Go duration string (e.g. 1s) (default 1)
  -R, --reverse                  sort in reverse order
      --shortcuts stringArray    paths to shortcut config files to load (env: PC_SHORTCUTS_FILES)
      --slow-ref-rate duration   Slow(er) refresh interval for resources (CPU, RAM) in seconds or as a Go duration string (e.g. 1s). The value should be higher than --ref-rate (default 1)
  -S, --sort string              sort column name. legal values (case insensitive): [AGE, CPU, EXIT, HEALTH, MEM, NAME, NAMESPACE, PID, RESTARTS, STATUS] (default "NAME")
      --theme string             select process compose theme (default "Default")
  -t, --tui                      enable TUI (disable with -t=false) (env: PC_DISABLE_TUI) (default true)
```

### Options inherited from parent commands

```
  -L, --log-file string      Specify the log file path (env: PC_LOG_FILE) (default "/tmp/process-compose-<user>.log")
      --no-server            disable HTTP server (env: PC_NO_SERVER)
      --ordered-shutdown     shut down processes in reverse dependency order
  -p, --port int             port number (env: PC_PORT_NUM) (default 8080)
      --read-only            enable read-only mode (env: PC_READ_ONLY)
  -u, --unix-socket string   path to unix socket (env: PC_SOCKET_PATH) (default "/tmp/process-compose-<pid>.sock")
  -U, --use-uds              use unix domain sockets instead of tcp
```

### SEE ALSO

* [process-compose](process-compose.md)	 - Processes scheduler and orchestrator

