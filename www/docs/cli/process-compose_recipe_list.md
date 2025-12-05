## process-compose recipe list

List locally installed recipes

### Synopsis

List all recipes that have been pulled and are available locally.

```
process-compose recipe list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --address string       address to listen on (env: PC_ADDRESS)
  -L, --log-file string      Specify the log file path (env: PC_LOG_FILE) (default "/tmp/process-compose-<user>.log")
      --no-server            disable HTTP server (env: PC_NO_SERVER)
      --ordered-shutdown     shut down processes in reverse dependency order
  -p, --port int             port number (env: PC_PORT_NUM) (default 8080)
      --read-only            enable read-only mode (env: PC_READ_ONLY)
  -u, --unix-socket string   path to unix socket (env: PC_SOCKET_PATH) (default "/tmp/process-compose-<pid>.sock")
  -U, --use-uds              use unix domain sockets instead of tcp
```

### SEE ALSO

* [process-compose recipe](process-compose_recipe.md)	 - Manage process-compose recipes

