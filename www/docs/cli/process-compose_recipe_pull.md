## process-compose recipe pull

Pull a recipe from the repository

### Synopsis

Download and install a recipe from the process-compose recipes repository.

The recipe will be downloaded to your local recipes directory and can be used
with 'process-compose -f ~/.process-compose/recipes/[recipe-name]/process-compose.yaml'

```
process-compose recipe pull [recipe-name] [flags]
```

### Options

```
  -f, --force           Force pull even if recipe exists locally
  -h, --help            help for pull
  -o, --output string   Output path for the recipe
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

