## process-compose attach

Attach the Process Compose TUI Remotely to a Running Process Compose Server

```
process-compose attach [flags]
```

### Options

```
  -a, --address string      address of the target process compose server (default "localhost")
  -h, --help                help for attach
  -l, --log-length int      log length to display in TUI (default 1000)
  -r, --ref-rate duration   TUI refresh rate in seconds or as a Go duration string (e.g. 1s) (default 1)
  -R, --reverse             sort in reverse order
  -S, --sort string         sort column name. legal values (case insensitive): [AGE, CPU, EXIT, HEALTH, MEM, NAME, NAMESPACE, PID, RESTARTS, STATUS] (default "NAME")
      --theme string        select process compose theme (default "Default")
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

