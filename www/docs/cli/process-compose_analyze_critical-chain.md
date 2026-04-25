## process-compose analyze critical-chain

Print the critical process startup chain

### Synopsis

Print a tree of processes ordered from the processes that nothing
depends on (top-level) down through their dependencies, annotated with
startup timings -- similar to 'systemd-analyze critical-chain'.

For each process two times are printed:

  @<offset>   Time after the project started that the process became ready.
              (For processes without a readiness probe this is the time the
              process was launched.)
  +<duration> Time the process spent between launch and becoming ready.
              Only shown for processes with a readiness signal (readiness
              probe, liveness probe, or 'ready_log_line').

If process names are given as arguments, only those processes (and their
dependency sub-chains) are printed; otherwise every top-level process is
printed.

```
process-compose analyze critical-chain [process...] [flags]
```

### Options

```
  -h, --help   help for critical-chain
```

### Options inherited from parent commands

```
      --address string       address to listen on (env: PC_ADDRESS) (default "localhost")
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

* [process-compose analyze](process-compose_analyze.md)	 - Analyze startup timing and dependency information

