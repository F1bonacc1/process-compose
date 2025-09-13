# Process Compose 🔥

Process Compose is a simple and flexible scheduler and orchestrator to manage non-containerized applications.

## Why was it made?

Because sometimes you just don't want to deal with docker files, volume definitions, networks and docker registries.
Since it's written in Go, Process Compose is a single binary file and has no other dependencies.

Once [installed](installation.md), you just need to describe your workflow using a simple [YAML](http://yaml.org/) schema in a file called `process-compose.yaml`:

```yaml
version: "0.5"

processes:
  hello:
    command: echo 'Hello World from Process Compose'
```

And start it by running `process-compose up` from your terminal.

Check the [Documentation](launcher.md) for more advanced use cases.

<img src="https://github.com/F1bonacc1/process-compose/raw/main/imgs/tui.png" alt="TUI" style="zoom:67%;" />

## Features

- Processes execution (in parallel or/and serially)
- Processes dependencies and startup order
- Process recovery policies
- Manual process [re]start
- Processes arguments `bash` or `zsh` style (or define your own shell)
- Per process and global environment variables using [envsubst](https://github.com/drone/envsubst)
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
- Run a Foreground Process 
- Themes Support
- On the fly Process configuration edit
- On the fly Project update
- [Recipes](https://github.com/F1bonacc1/process-compose-recipes) Management
