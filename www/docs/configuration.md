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

By default the `.env` file in the current directory is used if exists. It is possible to specify other file(s) to be used instead:

```shell
process-compose -e .env -e .env.local -e .env.dev
```

For situations where the you would like to disable the automatic `.env` file loading you might want to use the `--disable-dotenv` flag.

## Disabling Dotenv Environment Variable Injection (`is_dotenv_disabled`)

By default, Process Compose reads variables from specified `.env` files (or a default `.env` file) and injects them into the environment of *all* managed processes. This is often convenient, but can interfere with processes that have their own built-in mechanisms for loading configuration directly from a `.env` file, especially when relying on that mechanism for configuration updates or hot-reloading.

The core issue arises from environment variable precedence: environment variables explicitly set for a process (including those injected by Process Compose from `.env` files) typically override variables loaded by the process itself from its *own* `.env` file reading mechanism (e.g., using libraries like `godotenv`, `python-dotenv`, etc.). This means that if you change a variable in your `.env` file and restart only the *process* within Process Compose, the process might not see the change because the old value, injected by Process Compose when *it* started, takes precedence. A full restart of Process Compose would be required to pick up the changes.

### The `is_dotenv_disabled` Option

To address this, Process Compose provides the `is_dotenv_disabled` option within the process configuration.

When set to `true`, this option instructs Process Compose **not** to inject environment variables sourced from *its* loaded `.env` files into the environment of *that specific* process. The process is then free to read and interpret its specified `.env` file using its own logic, without interference from variables injected by Process Compose's dotenv loading mechanism.

**Syntax:**

```yaml
processes:
  <process-name>:
    command: <your_command>
    # ... other process options
    # Set to true to prevent Process Compose from injecting variables
    # loaded from its .env files into this specific process.
    is_dotenv_disabled: true # Defaults to false if not specified
```

### Important Considerations

- `is_dotenv_disabled: true` only affects the injection of variables loaded by Process Compose from `.env` files.
- It does **not** prevent the injection of variables defined in the top-level `environment:` section or the process-specific `environment:` section. These variables are always passed.
- Process Compose itself still loads the `.env` files as configured; it simply refrains from passing those specific variables to processes and their health probes marked with `is_dotenv_disabled: true`.
- Other processes managed by the same Process Compose instance will continue to receive variables from Process Compose's `.env` loading unless they also have `is_dotenv_disabled: true` set.

## .pc_env file

`.pc_env` file allows you to control Process Compose local, user environment specific settings.  
Ideally it should contain Process Compose specific environment variables:

```.env
PC_DISABLE_TUI=1
PC_PORT_NUM=8080
PC_NO_SERVER=1
```

## Disable Automatic Expansion

Process Compose provides 2 ways to disable the automatic environment variables expansion:

1. Escape the environment variables with `$$`. Example:
   ```yaml
   processes:
   	foo:
     	command: echo I am $$ENV_TEST
       environment:
       	- 'ENV_TEST=ready'
   ```

   **Output**: `I am ready`

2. Globally disable the automatic expansion with `disable_env_expansion: true`. Example:
   ```yaml
   disable_env_expansion: true
   processes:
   	foo:
     	command: echo I am $ENV_TEST
       environment:
       	- 'ENV_TEST=ready'
   ```

   **Output**: `I am ready`

>  :bulb: Note: The default behavior for the following `process-compose.yaml`:
>
>  ```yaml
>  processes:
>  	foo:
>    	command: echo I am $ENV_TEST
>      environment:
>      	- 'ENV_TEST=ready'
>  ```
>
>  **Output**: `I am `

## Environment Commands

The `env_cmds` feature allows you to dynamically populate environment variables by executing short commands before starting your processes. This is useful when you need environment values that are determined at runtime or need to be fetched from the system.

### Configuration

Environment commands are defined in the `env_cmds` section of your `process-compose.yaml` file. Each entry consists of:
- An environment variable name (key)
- A command to execute (value)

```yaml
env_cmds:
  ENV_VAR_NAME: "command to execute"
```

### Example Configuration

```yaml
env_cmds:
  DATE: "date"
  OS_NAME: "awk -F= '/PRETTY/ {print $2}' /etc/os-release"
  UPTIME: "uptime -p"
```

### Usage

To use the environment variables populated by `env_cmds`, reference them in your process definitions using `$${VAR_NAME}` syntax:

```yaml
processes:
  my-process:
    command: "echo Current date is: $${DATE}"
```

### Constraints and Considerations

1. **Execution Time**: Commands should complete within 2 seconds. Longer-running commands may cause process-compose startup delays or timeouts.

2. **Command Output**: 
   - Commands should output a single line of text
   - The output will be trimmed of leading/trailing whitespace
   - The output becomes the value of the environment variable

3. **Error Handling**:
   - If a command fails, the environment variable will not be set
   - Process-compose will log any command execution errors

### Best Practices

1. Keep commands simple and fast-executing
2. Use commands that produce consistent, predictable output
3. Validate command output format before using in production
4. Consider caching values that don't need frequent updates

### Example Use Cases

1. **System Information**:
```yaml
env_cmds:
  HOSTNAME: "hostname"
  KERNEL_VERSION: "uname -r"
```

2. **Time-based Values**:
```yaml
env_cmds:
  TIMESTAMP: "date +%s"
  DATE_ISO: "date -u +%Y-%m-%dT%H:%M:%SZ"
```

3. **Resource Information**:
```yaml
env_cmds:
  AVAILABLE_MEMORY: "free -m | awk '/Mem:/ {print $7}'"
  CPU_CORES: "nproc"
```

## Variables

Variables in Process Compose rely on [Go template engine](https://pkg.go.dev/text/template)

#### Rendered Parameters:

* `processes.process.command`
* `processes.process.working_dir`
* `processes.process.log_location`
* `processes.process.description`
* `processes.process.environment` values.
* For `readiness_probe`and `liveness_probe`:
  * `processes.process.<probe>.exec.command`
  * `processes.process.<probe>.http_get.host`
  * `processes.process.<probe>.http_get.path`
  * `processes.process.<probe>.http_get.scheme`
  * `processes.process.<probe>.http_get.port`

### Local (Per Process)

```yaml hl_lines="3-7 9-10 13"
processes:
  watcher:
    vars:
      LOG_LOCATION: "./watcher.log"
      OK: SUCCESS
      PRE: 2
      POST: 8
      BASH: "/bin/bash"

    command: "sleep {{.PRE}} && echo {{.OK}} && sleep {{.POST}}"
    log_location: {{.LOG_LOCATION}}
    environment:
      - 'SHELL={{.BASH}}'
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

### Auto Inserted Variables

Process Compose will insert `PC_REPLICA_NUM` variable that will represent the replica number of the process. This will allow to conveniently scale processes using the following example configuration:

```yaml hl_lines="3 8"
processes:  
  server:
    command: "python3 -m http.server 404{{.PC_REPLICA_NUM}}"
    is_tty: true
    readiness_probe:
      http_get:
        host: "127.0.0.1"
        port: "404{{.PC_REPLICA_NUM}}"
        scheme: "http"
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

## On the Fly Configuration Edit

Process Compose allows you to edit processes configuration without restarting the entire project. To achieve that, select one of the following options:

### Project Edit

Modify your `process-compose.yaml` file (or files) and apply the changes by running:

```shell
process-compose project update -f process-compose.yaml # add -v for verbose output, add -f for additional files to be merged
```

This command will:

1. If there are changes to existing processes in the updated `process-compose.yaml` file, stop the old instances of these processes and start new instances with the updated config.
2. If there are only new processes in the updated `process-compose.yaml` file, start the new processes without affecting the others.
3. If some processes no longer exist in the updated `process-compose.yaml` file, stop only those old processes without touching the others.

**Note:** If TUI or TUI client is being used, you can trigger the original files reload with the `Ctrl+L` shortcut.

### Process Edit

To edit a single process:

1. Select it in the TUI or in the TUI client.
2. Press `CTRL+E`
3. Apply the changes, save and quit the editor.
4. The process will restart with the new configuration, or won't restart if there are no changes.

:bulb: **Notes:**

1. These changes are not persisted and not applied to your `process-compose.yaml`
2. In case of parsing errors or unrecognized fields:
   1. All the changes will be reverted to the last known correct state.
   2. The editor will open again with a detailed error description at the top of the file.
3. Process Compose will use one of:
   1. Your default editor defined in `$EDITOR` environment variable. If empty:
   2. For non-Windows OSs: `vim`, `nano`, `vi` in that order.
   3. For Windows OS: `notepad.exe`, `notepad++.exe`, `code.exe`, `gvim.exe` in that order.
4. Some of the fields are read only.

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

Note: By default `process-compose` will start processes from all the configured namespaces. To start a subset of the configured namespaces (`ns1`, `ns2`, `ns3`):

```shell
process-compose -n ns1 -n ns3
# will start only ns1 and ns3. ns2 namespace won't run and won't be visible in the TUI
```

## Misc

#### Strict Configuration Validation
To avoid minor `process-compose.yaml` configuration errors and typos it is recommended to enable `is_strict` flag:

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

#### Elevated Processes

Process Compose uses `sudo` (on Linux and macOS) and `runas` (on Windows) to enable execution of elevated processes in both TUI and headless modes.

```yaml hl_lines="5"
processes:
  elevated_ls:
    description: "run an elevated process"
    command: "ls -l /root"
    is_elevated: true
    shutdown:
      signal: 9
```
* In TUI mode, elevated processes awaiting password input are marked with a yellow ▲.
* To enter a password in TUI mode:
  1. Select the elevated process.
  2. Type the password.
  3. Press the `Enter` key.
* To exit the password prompt, press the `ESC` key at any time.
* To re-enter password mode, select the process again.
* The entered password will be applied to all elevated processes in pending status.

#### Ordered Shutdown
```yaml
ordered_shutdown: true
```
Shut down processes in reverse dependency order.
> :bulb: ordered-shutdown can be passed as a command-line parameter when starting process-compose (see [CLI](/cli/process-compose/)), set permanently in `process-compose.yaml` (see this section), or by setting the environment variable `PC_ORDERED_SHUTDOWN`.


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
