## process-compose process restart

Restart a process

```
process-compose process restart [PROCESS] [flags]
```

### Options

```
  -h, --help   help for restart
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

* [process-compose process](process-compose_process.md)	 - Execute operations on the available processes

