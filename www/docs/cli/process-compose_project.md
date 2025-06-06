## process-compose project

Execute operations on a running Process Compose project

### Options

```
  -a, --address string   address of the target process compose server (default "localhost")
  -h, --help             help for project
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
* [process-compose project is-ready](process-compose_project_is-ready.md)	 - Check if Process Compose project is ready (or wait for it to be ready)
* [process-compose project state](process-compose_project_state.md)	 - Get Process Compose project state
* [process-compose project update](process-compose_project_update.md)	 - Update an already running process-compose instance by passing an updated process-compose.yaml file

