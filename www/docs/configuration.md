# Configuration

## Environment Variables

### Per Process

```yaml
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
