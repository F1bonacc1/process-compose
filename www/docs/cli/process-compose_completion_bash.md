## process-compose completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(process-compose completion bash)

To load completions for every new session, execute once:

#### Linux:

	process-compose completion bash > /etc/bash_completion.d/process-compose

#### macOS:

	process-compose completion bash > $(brew --prefix)/etc/bash_completion.d/process-compose

You will need to start a new shell for this setup to take effect.


```
process-compose completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
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

* [process-compose completion](process-compose_completion.md)	 - Generate the autocompletion script for the specified shell

