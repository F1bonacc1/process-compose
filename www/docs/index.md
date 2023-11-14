# Process Compose 🔥

Process Compose is a simple and flexible scheduler and orchestrator to manage non-containerized applications.

## Why was it made?

Because sometimes you just don't want to deal with docker files, volume definitions, networks and docker registries.

<img src="https://github.com/F1bonacc1/process-compose/raw/main/imgs/tui.png" alt="TUI" style="zoom:67%;" />

## Features

- Processes execution (in parallel or/and serially)
- Processes dependencies and startup order
- Process recovery policies
- Manual process [re]start
- Processes arguments `bash` or `zsh` style (or define your own shell)
- Per process and global environment variables
- Per process or global (single file) logs
- Health checks (liveness and readiness)
- Terminal User Interface (TUI) or CLI modes
- Forking (services or daemons) processes
- REST API (OpenAPI a.k.a Swagger)
- Logs caching
- Functions as both server and client
- Configurable shortcuts
- Merge Configuration Files
- Namespaces
- Run Multiple Replicas of a Process
