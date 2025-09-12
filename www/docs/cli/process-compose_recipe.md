## process-compose recipe

Manage process-compose recipes

### Synopsis

Manage process-compose recipes from the community repository.

Recipes are pre-configured process-compose.yaml files for common use cases
like databases, message queues, and other services.

### Options

```
  -h, --help   help for recipe
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
* [process-compose recipe list](process-compose_recipe_list.md)	 - List locally installed recipes
* [process-compose recipe pull](process-compose_recipe_pull.md)	 - Pull a recipe from the repository
* [process-compose recipe remove](process-compose_recipe_remove.md)	 - Remove a locally installed recipe
* [process-compose recipe search](process-compose_recipe_search.md)	 - Search for recipes in the repository
* [process-compose recipe show](process-compose_recipe_show.md)	 - Show the content of a local recipe

