## process-compose project update

Update an already running process-compose instance by passing an updated process-compose.yaml file

```
process-compose project update [flags]
```

### Options

```
  -f, --config stringArray      path to config files to load (env: PC_CONFIG_FILES)
  -h, --help                    help for update
  -n, --namespace stringArray   run only specified namespaces (default all)
  -v, --verbose                 verbose output
```

### Options inherited from parent commands

```
  -a, --address string       address of the target process compose server (default "localhost")
  -L, --log-file string      Specify the log file path (env: PC_LOG_FILE) (default "/tmp/process-compose-<user>.log")
      --no-server            disable HTTP server (env: PC_NO_SERVER)
      --ordered-shutdown     shut down processes in reverse dependency order
  -p, --port int             port number (env: PC_PORT_NUM) (default 8080)
      --read-only            enable read-only mode (env: PC_READ_ONLY)
  -u, --unix-socket string   path to unix socket (env: PC_SOCKET_PATH) (default "/tmp/process-compose-<pid>.sock")
  -U, --use-uds              use unix domain sockets instead of tcp
```

### SEE ALSO

* [process-compose project](process-compose_project.md)	 - Execute operations on a running Process Compose project

