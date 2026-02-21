## process-compose namespace

Perform operations on a namespace (start, stop, restart, list)

```
process-compose namespace [flags]
```

### Options

```
  -h, --help   help for namespace
```

### Options inherited from parent commands

```
      --address string       address to listen on (env: PC_ADDRESS) (default "localhost")
  -L, --log-file string      Specify the log file path (env: PC_LOG_FILE) (default "/tmp/process-compose-<user>.log")
      --no-server            disable HTTP server (env: PC_NO_SERVER)
      --ordered-shutdown     shut down processes in reverse dependency order
  -p, --port int             port number (env: PC_PORT_NUM) (default 8080)
      --read-only            enable read-only mode (env: PC_READ_ONLY)
      --token-file string    path to a file containing the API token (env: PC_API_TOKEN_PATH)
  -u, --unix-socket string   path to unix socket (env: PC_SOCKET_PATH) (default "/tmp/process-compose-<pid>.sock")
  -U, --use-uds              use unix domain sockets instead of tcp
```

### SEE ALSO

* [process-compose](process-compose.md)	 - Processes scheduler and orchestrator
* [process-compose namespace list](process-compose_namespace_list.md)	 - List all available namespaces
* [process-compose namespace restart](process-compose_namespace_restart.md)	 - Restart all processes in a namespace
* [process-compose namespace start](process-compose_namespace_start.md)	 - Start all processes in a namespace
* [process-compose namespace stop](process-compose_namespace_stop.md)	 - Stop all processes in a namespace

