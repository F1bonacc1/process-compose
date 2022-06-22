## Process Compose

[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](https://go.dev/) [![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)](https://GitHub.com/Naereen/StrapDown.js/graphs/commit-activity) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com) ![Go Report](https://goreportcard.com/badge/github.com/F1bonacc1/process-compose) [![Releases](https://img.shields.io/github/downloads/F1bonacc1/process-compose/total.svg)]()

Process Compose is a simple and flexible scheduler and orchestrator to manage non-containerized applications.

**Why?** Because sometimes you just don't want to deal with docker files, volume definitions, networks and docker registries.

Main use cases would be:

- Processes execution (in parallel or serially)
- Defining processes dependencies and startup order
- Defining recovery policies (restart `on-failure`, `always`, `no`). Manual recovery is also supported.
- Declaring processes arguments
- Declaring processes environment variables

It is heavily inspired by [docker-compose](https://github.com/docker/compose), but without the need for containers. The configuration syntax tries to follow the docker-compose specifications, with a few minor additions and lots of subtractions.

<img src="./imgs/tui.png" alt="TUI" style="zoom:67%;" />

### Quick Start

Imaginary system diagram:

![Diagram](./imgs/diagram.png)

`process-compose.yaml` definitions for the system above:

```yaml
version: "0.5"

environment:
  - "GLOBAL_ENV_VAR=1"
log_location: /path/to/combined/output/logfile.log
log_level: debug

processes:
  Manager:
    command: "/path/to/manager"
    availability:
      restart: "always"
    depends_on:
      ClientA:
        condition: process_started
      ClientB:
        condition: process_started

  ClientA:
    command: "/path/to/ClientA"
    availability:
      restart: "always"
    depends_on:
      Server_1A:
        condition: process_started
      Server_2A:
        condition: process_started
    environment:
      - "LOCAL_ENV_VAR=1"

  ClientB:
    command: "/path/to/ClientB -some -arg"
    availability:
      restart: "always"
    depends_on:
      Server_1B:
        condition: process_started
      Server_2B:
        condition: process_started
    environment:
      - "LOCAL_ENV_VAR=2"

  Server_1A:
    command: "/path/to/Server_1A"
    availability:
      restart: "always"

  Server_2A:
    command: "/path/to/Server_2A"
    availability:
      restart: "always"

  Server_1B:
    command: "/path/to/Server_1B"
    availability:
      restart: "always"

  Server_2B:
    command: "/path/to/Server_2B"
    availability:
      restart: "always"
```

Finally, run `process-compose` in the `process-compose.yaml` directory. Or give it a direct path:

```shell
process-compose -f /path/to/process-compose.yaml
```

### Installation

- Go to the [releases](https://github.com/F1bonacc1/process-compose/releases/latest), download the package for your OS, and copy the binary to somewhere on your PATH.
- If you have the Nix package manager installed with Flake support, just run:

```sh
nix run github:F1bonacc1/process-compose
```

### Documentation

- See [examples](https://github.com/F1bonacc1/process-compose/tree/main/examples) of workflows for best practices
- See below

#### List of Features and Planned Features

##### ✅ Mostly implemented

##### ❌ Implementation not started (Your feedback and ⭐ will motivate further development 😃)

#### ✅ <u>Launcher</u>

##### ✅ Parallel

```yaml
process1:
  command: "sleep 3"
process2:
  command: "sleep 3"
```

##### ✅ Serial

```yaml
process1:
  command: "sleep 3"
  depends_on:
    process2:
      condition: process_completed_successfully # or "process_completed" if you don't care about errors
process2:
  command: "sleep 3"
  depends_on:
    process3:
      condition: process_completed_successfully # or "process_completed" if you don't care about errors
```

##### ❌ Instance Number

##### ✅ Define process dependencies

```yaml
process2:
  depends_on:
    process3:
      condition: process_completed_successfully # or "process_started" (default)
  process3:
    condition: process_completed_successfully
```

##### ✅ Termination Parameters

```yaml
process1:
  command: "pg_ctl start"
  shutdown:
    command: "pg_ctl stop"
    timeout_seconds: 10 # default 10
    signal: 15 # default 15, but only if command is not defined or empty
```

`shutdown` is optional and can be omitted. The default behaviour in this case: `SIGTERM` is issued to the running process.

In case only `shutdown.signal` is defined `[1..31] ` the running process will be terminated with its value.

In case the the `shutdown.command` is defined:

1. The `shutdown.command` is executed with all the Environment Variables of the main process
2. Wait `shutdown.timeout_seconds` for its completion (if not defined wait for 10 seconds)
3. In case of timeout the process will receive the `SIGKILL` signal

#### ✅ <u>Output Handling</u>

##### ✅ Show process name

##### ✅ Different colors per process

##### ✅ StdErr is printed in Red

<img src="./imgs/output.png" alt="output" style="zoom:50%;" />

##### ❌ Silence specific processes

#### ✅ <u>TUI</u> (Terminal User Interface)

##### ✅ Review processes status

##### ✅ Start processes (only completed or disabled)

##### ✅ Stop processes

##### ✅ Review logs

TUI is the default run mode, but it's possible to disable it:

```shell
./process-compose -t=false
```

Control the UI log buffer size:

```yaml
log_level: info
log_length: 1200 #default: 1000
processes:
  process2:
    command: "ls -R /"
```

**Note**: Using a too large buffer will put a significant penalty on your CPU.

#### ✅ <u>Logger</u>

##### ✅ Per Process Log Collection

```yaml
process2:
  log_location: ./pc.process2.log #if undefined or empty no logs will be saved
```

##### ✅ Capture StdOut output

##### ✅ Capture StdErr output

##### ✅ Merge into a single file

```yaml
processes:
  process2:
    command: "chmod 666 /path/to/file"
environment:
  - "ABC=42"
log_location: ./pc.global.log #if undefined or empty no logs will be saved (if also not defined per process)
```

##### ❌ Silence specific processes

##### ✅ Process compose console log level

```yaml
log_level: info # other options: "trace", "debug", "info", "warn", "error", "fatal", "panic"
processes:
  process2:
    command: "chmod 666 /path/to/file"
```

This setting controls the `process-compose` log level. The processes log level should be defined inside the process. It is recommended to support its definition with an environment variable that can be defined in `process-compose.yaml`

#### ❌ <u>Health Checks</u>

##### ❌ Is Alive

##### ❌ Is Ready

##### ❌ Auto Restart if not healthy

##### ✅ Auto Restart on exit

```yaml
process2:
  availability:
    restart: on-failure # other options: "always", "no" (default)
    backoff_seconds: 2 # default: 1
    max_restarts: 5 # default: 0 (unlimited)
```

#### ✅ <u>Environment Variables</u>

##### ✅ Per Process

```yaml
process2:
  environment:
    - "I_AM_LOCAL_EV=42"
```

##### ✅ Global

```yaml
processes:
  process2:
    command: "chmod 666 /path/to/file"
  environment:
    - "I_AM_LOCAL_EV=42"
environment:
  - "I_AM_GLOBAL_EV=42"
```

Default environment variables:

`PC_PROC_NAME` - Defines the process name as defined in the `process-compose.yaml` file.

`PC_REPLICA_NUM` - Defines the process replica number. Useful for port collision avoidance for processes with multiple replicas.

#### ✅ <u>REST API</u>

A convenient Swagger API is provided: http://localhost:8080/swagger/index.html

<img src="./imgs/swagger.png" alt="Swagger" style="zoom:67%;" />

#### ✅ <u>Configuration</u>

##### ✅ Support .env file

##### ✅ Override ${var} and $var from environment variables or .env values

##### ❌ Merge 2 or more configuration files with override values

##### ✅ Specify which configuration files to use

```shell
process-compose -f "path/to/process-compose-file.yaml"
```

##### ✅ Auto discover configuration files

The following discovery order is used: `compose.yml, compose.yaml, process-compose.yml, process-compose.yaml`. If multiple files are present the first one will be used.

#### ✅ <u>Multi-platform</u>

##### ✅ Linux

The default backend is `bash`. You can define a different backend with a `SHELL` environment variable.

##### ✅ Windows

The default backend is `cmd`. You can define a different backend with a `SHELL` environment variable.

```yaml
process1:
  command: "python -c print(str(40+2))"
  #note that the same command for bash/zsh would look like: "python -c 'print(str(40+2))'"
```

Using `powershell` backend had some funky behaviour (like missing `command1 && command2` functionality in older versions). If you need to run powershell scripts, use the following syntax:

```yaml
process2:
  command: "powershell.exe ./test.ps1 arg1 arg2 argN"
```

##### ❌ macOS
