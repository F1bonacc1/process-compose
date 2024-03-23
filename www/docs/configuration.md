# Configuration

## Environment Variables

### Local (Per Process)

```yaml
processes:
  process2:
    environment:
      - "I_AM_LOCAL_EV=42"
```

### Global

```yaml
environment:
  - "I_AM_GLOBAL_EV=42"

processes:
  process2:
    command: "chmod 666 /path/to/file"
  environment:
    - "I_AM_LOCAL_EV=42"
```

Default environment variables:

`PC_PROC_NAME` - Defines the process name as defined in the `process-compose.yaml` file.

`PC_REPLICA_NUM` - Defines the process replica number. Useful for port collision avoidance for processes with multiple replicas.

## .env file
```.env
VERSION='1.2.3'
DB_USER='USERNAME'
DB_PASSWORD='VERY_STRONG_PASSWORD'
WAIT_SEC=60
```
Override ${var} and $var from environment variables or .env values
```yaml hl_lines="3 6"
processes:
  downloader:
    command: "python3 data_downloader_${VERSION}.py -s 'data.source.B.uri'"
    availability:
      restart: "always"
      backoff_seconds: ${WAIT_SEC}
    environment:
      - 'OUTPUT_DIR=/path/to/B/data'
```

## Variables

Variables in Process Compose rely on [Go template engine](https://pkg.go.dev/text/template)

#### Rendered Parameters:

* `processes.process.command`
* `processes.process.working_dir`
* `processes.process.log_location`
* `processes.process.description`
* For `readiness_probe`and `liveness_probe`:
  * `processes.process.<probe>.exec.command`
  * `processes.process.<probe>.http_get.host`
  * `processes.process.<probe>.http_get.path`
  * `processes.process.<probe>.http_get.scheme`

### Local (Per Process)

```yaml hl_lines="3-7 9-10 13"
processes:
  watcher:
    vars:
      LOG_LOCATION: "./watcher.log"
      OK: SUCCESS
      PRE: 2
      POST: 8

    command: "sleep {{.PRE}} && echo {{.OK}} && sleep {{.POST}}"
    log_location: {{.LOG_LOCATION}}
    readiness_probe:
      exec:
        command: "grep -q {{.OK}} {{.LOG_LOCATION}}"
      initial_delay_seconds: 1
      period_seconds: 1
      timeout_seconds: 1
      success_threshold: 1      
```

> :bulb: Notice the `.` (dot) before each `.VARIABLE`

### Global

```yaml
vars:
  VERSION: v1.2.3
  FTR_A_ENABLED: true
  FTR_B_ENABLED: true

processes:
  version:
    # Environment and Process Compose variables can complement each other
    command: "echo 'version {{or \"${VERSION}\" .VERSION}}'"
  feature:
    command: "echo '{{if .FTR_A_ENABLED}}Feature A Enabled{{else}}Feature A Disalbed{{end}}'"
  not_supported:
    command: "echo 'Hi {{if and .FTR_A_ENABLED .FTR_B_ENABLED}}Not Supported{{end}}'"
```
```shell
#output:
version v1.2.3 #if $VERSION environment variable is undefined. The value of $VERSION if it is. 
Feature A Enabled
Not Supported
```

### Template Escaping

In a scenario where Go template syntax is part of your command, you will want to escape it:

```go
{{ "{{ .SOME_VAR }}" }}
```

For example:

```yaml hl_lines="6"
processes:
  nginx:
    command: "docker run -d --rm -p80:80 --name nginx_test nginx"
    liveness_probe:
      exec:
        command: '[ $(docker inspect -f "{{.State.Running}}" nginx_test) = true ]'
```

Will become:

```yaml hl_lines="6"
processes:
  nginx:
    command: "docker run -d --rm -p80:80 --name nginx_test nginx"
    liveness_probe:
      exec:
        command: '[ $(docker inspect -f {{ "{{.State.Running}}" }} nginx_test) = true ]'
```

> :bulb: For backward compatibility, if neither global nor local variables exist in `process-compose.yaml` the template engine won't run.

## Specify which configuration files to use

```shell
process-compose -f "path/to/process-compose-file.yaml"
```

## Auto discover configuration files

The following discovery order is used: `compose.yml, compose.yaml, process-compose.yml, process-compose.yaml`. If multiple files are present the first one will be used.

## Merge 2 or more configuration files with override values

```shell
process-compose -f "path/to/process-compose-file.yaml" -f "path/to/process-compose-override-file.yaml"
```

Using multiple `process-compose` files lets you customize a `process-compose` application for different environments or different workflows.

See the [Merging Configuration](merge.md) for more information on merging files.

## Backend

For cases where your process compose requires a non default or transferable backend definition, setting an environment variable won't do. For that, you can configure it directly in the `process-compose.yaml` file:

```yaml
version: "0.5"
shell:
  shell_command: "python3"
  shell_argument: "-m"
processes:
  http:
    command: "server.py"
```

> **Note**: please make sure that the `shell.shell_command` value is in your `$PATH`

#### Linux

The default backend is `bash`. You can define a different backend with a `COMPOSE_SHELL` environment variable.

#### Windows

The default backend is `cmd`. You can define a different backend with a `COMPOSE_SHELL` environment variable.

```yaml
process1:
  command: "python -c print(str(40+2))"
  #note that the same command for bash/zsh would look like: "python -c 'print(str(40+2))'"
```

Using `powershell` backend had some funky behavior (like missing `command1 && command2` functionality in older versions). If you need to run powershell scripts, use the following syntax:

```yaml
process2:
  command: "powershell.exe ./test.ps1 arg1 arg2 argN"
```

#### macOS

The default backend is `bash`. You can define a different backend with a `COMPOSE_SHELL` environment variable.

## Namespaces

Assigning namespaces to processes allows better grouping and sorting, especially in TUI:

```yaml
processes:
  process1:
    command: "tail -f -n100 process-compose-${USER}.log"
    working_dir: "/tmp"
    namespace: debug # if not defined 'default' namespace is automatically assigned to each process
```

Note: By default `process-compose` will start process from all the configured namespaces. To start a sub set of the configured namespaces (`ns1`, `ns2`, `ns3`):

```shell
process-compose -n ns1 -n ns3
# will start only ns1 and ns3. ns2 namespace won't run and won't be visible in the TUI
```

## Misc

#### Strict Configuration Validation
To avoid minor `proces-compose.yaml` configuration errors and typos it is recommended to enable `is_strict` flag:

```yaml hl_lines="2 5"
version: "0.5"
is_strict: true
processes:
  process1:
   commnad: "sleep 1" # <-- notice the typo here
```
The above configuration will fail the Process Compose start and exit with error code `1`:
```shell
unknown key commnad found in process process1
```

#### Pseudo Terminals

Certain processes check if they are running within a terminal, to simulate a TTY mode you can use a `is_tty` flag:

```yaml hl_lines="4"
processes:  
  process0:
    command: "ls -lFa --color=tty"
    is_tty: true
```

> :bulb: `STDIN` and `Windows` are not supported at this time.

#### Multiline Command Support
Process Compose respects all the multiline `YAML` [specification](https://yaml-multiline.info/) variations. 

Examples:

```yaml
processes:
  block_folded:
    command: >
      echo 1
      && echo 2

      echo 3

  block_literal:
    command: |
      echo 4
      echo 5
    depends_on:
      block_folded:
        condition: process_completed

  flow_single:
    command: 'echo 6
      && echo 7

      echo 8'
    depends_on:
      block_literal:
        condition: process_completed

  flow_double:
    command: "echo 9
      && echo 10
      
      echo 11"
    depends_on:
      flow_single:
        condition: process_completed
  
  flow_plain:
    command: echo 12
      && echo 13
      
      echo 14
    depends_on:
      flow_double:
        condition: process_completed
```
> :bulb: The extra blank lines (`\n`) in the command string are to introduce a newline to the command.
