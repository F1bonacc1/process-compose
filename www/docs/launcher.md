# Processes Lifetime

## Start in Parallel

```yaml
processes:
  process1:
    command: "sleep 3"
  process2:
    command: "sleep 3"
```

## Start Serially

```yaml
processes:
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

## Multiple Replicas of a Process

You can run multiple replicas of a process by adding `processes.process_name.replicas` parameter (default: 1)

```yaml
processes:
  process_name:
    command: "sleep 2"
    log_location: ./log_file.{PC_REPLICA_NUM}.log  # <- {PC_REPLICA_NUM} will be replaced with replica number. If more than one replica and PC_REPLICA_NUM is not specified, the replica number will be concatenated to the file end.
    replicas: 2
```

To scale a process on the fly CLI:

```shell
process-compose process scale process_name 3
```

To scale a process on the fly TUI: `F2` or Process Compose in client mode (`process-compose attach`).

> :bulb: Starting multiple processes using the same port, will fail. Please use the injected `PC_REPLICA_NUM` environment variable to increment the used port number.

## Specify a working directory

```yaml
processes:
  process1:
    command: "ls -laF --color=always"
    working_dir: "/path/to/your/working/directory"
```

Make sure that you have the proper access permissions to the specified `working_dir`. If not, the command will fail with a `permission denied` error. The process status in TUI will be `Error`.

## Define process dependencies

```yaml
processes:
  process2:
    depends_on:
      process3:
        condition: process_completed_successfully
      process4:
        condition: process_completed_successfully
```

There are 4 condition types that can be used in process dependencies:

* `process_completed` - is the type for waiting until a process has been completed (any exit code)
* `process_completed_successfully` - is the type for waiting until a process has been completed successfully (exit code 0)
* `process_healthy` - is the type for waiting until a process is healthy
* `process_started` - is the type for waiting until a process has started (default)

## Run only specific processes

For testing and debugging purposes, especially when your `process-compose.yaml` file contains many processes, you might want to specify only a subset of processes to run. For example:

```yaml
#process-compose.yaml
processes:
  process1:
    command: "echo 'Hi from Process1'"
    depends_on:
      process2:
        condition: process_completed_successfully
  process2:
    command: "echo 'Hi from Process2'"
  process3:
    command: "echo 'Hi from Process3'"
```

```bash
process-compose up # will run all the processes - equal to 'process-compose'

#output:
#Hi from Process3
#Hi from Process2
#Hi from Process1
```

```bash
process-compose up process1 process3 # will run 'process1', 'process3' and all of their dependencies - 'process2'

#output:
#Hi from Process3
#Hi from Process2
#Hi from Process1
```

```bash
process-compose up process1 process3 --no-deps # will run 'process1', 'process3' without any dependencies

#output:
#Hi from Process3
#Hi from Process1
```



## Termination Parameters

```yaml
processes:
  nginx:
    command: "docker run --rm --name nginx_test nginx"
    shutdown:
      command: "docker stop nginx_test"
      timeout_seconds: 10 # default 10
      signal: 15 # default 15, but only if the 'command' is not defined or empty
      parent_only: no  # default no. If yes, only signal the running process instead of its whole process group
```

`shutdown` is optional and can be omitted. The default behavior in this case: `SIGTERM` is issued to the process group of the running process.

In case only `shutdown.signal` is defined `[1..31] ` the running process group will be terminated with its value.

If `shutdown.parent_only` is yes, the signal is only sent to the running process and not to the whole process group.

In case the `shutdown.command` is defined:

1. The `shutdown.command` is executed with all the Environment Variables of the primary process
2. Wait for `shutdown.timeout_seconds` for its completion (if not defined wait for 10 seconds)
3. In case of timeout, the process group will receive the `SIGKILL` signal (irrespective of the `shutdown.parent_only` option).

## Background (detached) Processes

```yaml hl_lines="4"
processes:
  nginx:
    command: "docker run -d --rm --name nginx_test nginx" # note the '-d' for detached mode
    is_daemon: true # this flag is required for background processes (default false)
    shutdown:
      command: "docker stop nginx_test"
      timeout_seconds: 10 # default 10
      signal: 15 # default 15, but only if command is not defined or empty
```

1. For processes that start services / daemons in the background, please use the `is_daemon` flag set to `true`.

2. In case a process is daemon it will be considered running until stopped.

3. Daemon processes can only be stopped with the `$PROCESSNAME.shutdown.command` as in the example above.

## Disabled Processes

Process execution can be disabled:

```yaml hl_lines="4"
processes:
  process_name:
    command: "ls -R /"
    disabled: true #default false
```

Even if disabled, it is still listed in the TUI and the REST client can be started manually when needed.

## Auto Restart on Exit

```yaml hl_lines="4"
processes:
  process2:
    availability:
      restart: on_failure # other options: "exit_on_failure", "always", "no" (default)
      backoff_seconds: 2 # default: 1
      max_restarts: 5 # default: 0 (unlimited)
```

## Terminate Process Compose on Failure

There are cases when you might want `process-compose` to terminate immediately when one of the processes exits with a non `0` exit code. This can be useful when you would like to perform "pre-flight" validation checks on the environment.

To achieve that, use `exit_on_failure` restart policy. If defined, `process-compose` will gracefully shut down all the other running processes and exit with the same exit code as the failed process.

```yaml  hl_lines="5"
processes:
  sanitycheck:
    command: "which go"
    availability:
      restart: "exit_on_failure"

  other_proc:
    command: "go test ./..."
    depends_on:
      sanitycheck:
        condition: process_completed_successfuly
```

## Terminate Process Compose once given process ends

There are cases when you might want `process-compose` to terminate immediately when one of the processes exits (regardless of the exit code). For example when running tests that depend on other processes like databases etc. You might want the processes, on which the test process depends, to start first, then run the tests, and finally terminate all processes once the test process exits, reporting the code returned by the test process.

To achieve that, set `availability.exit_on_end` to `true`, and `process-compose` will gracefully shut down all the other running processes and exit with the same exit code as the given process.

```yaml hl_lines="7"
processes:
  tests:
    command: tests-run
    availability:
      # NOTE: `restart: exit_on_failure` is not needed since
      # exit_on_end implies it.
      exit_on_end: true
    depends_on:
      redis: process_healthy
      postgres: process_healthy

  redis:
    command: redis-start
    readiness_probe:
      exec:
        command: redis-health-check

  postgres:
    command: postgres-start
    readiness_probe:
      exec:
        command: postgres-health-check
```

> :bulb:
> setting `restart: exit_on_failure` together with `exit_on_end: true` is not needed as the latter causes termination regardless of the exit code. However, it might be sometimes useful to `exit_on_end` with `restart: on_failure` and `max_restarts` in case you want the process to recover from failure and only cause termination on success.

> :bulb:
> `exit_on_end` can be set on more than one process, for example when running multiple tasks in parallel and wishing to terminate as soon as any one finished.
