## process-compose process monitor

Subscribe to process state changes and print them as they happen

### Synopsis

Subscribe to the server's process state stream and print one line per state event.

With no arguments, all processes are monitored. With one or more process names,
output is filtered to those processes only.

Output formats:
  text  Human-readable columns: timestamp, process, kind, status (default).
  json  One ProcessStateEvent JSON object per line, ready to pipe into jq.

By default, an initial snapshot of every process is emitted on connect; pass
--no-snapshot to suppress it.

```
process-compose process monitor [process-name...] [flags]
```

### Options

```
  -h, --help            help for monitor
      --no-snapshot     Skip the initial state snapshot, only print live changes
  -o, --output string   Output format: text or json (default "text")
```

### Options inherited from parent commands

```
  -a, --address string       address of the target process compose server (default "localhost")
  -L, --log-file string      Specify the log file path (env: PC_LOG_FILE) (default "/tmp/process-compose-<user>.log")
      --log-no-color         disable color output in the log file (env: PC_LOG_NO_COLOR)
      --no-server            disable HTTP server (env: PC_NO_SERVER)
      --ordered-shutdown     shut down processes in reverse dependency order
  -p, --port int             port number (env: PC_PORT_NUM) (default 8080)
      --read-only            enable read-only mode (env: PC_READ_ONLY)
      --token-file string    path to a file containing the API token (env: PC_API_TOKEN_PATH)
  -u, --unix-socket string   path to unix socket (env: PC_SOCKET_PATH) (default "/tmp/process-compose-<pid>.sock")
  -U, --use-uds              use unix domain sockets instead of tcp
```

### SEE ALSO

* [process-compose process](process-compose_process.md)	 - Execute operations on the available processes

