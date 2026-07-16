# Health Checks

Many applications running for long periods of time eventually transition to broken states, and cannot recover except by being restarted. Process Compose provides liveness and readiness probes to detect and remedy such situations.

Probes configuration and functionality are designed to work similarly to [Kubernetes liveness and readiness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).

## Liveness Probe

Liveness probes are used to monitor a [**daemon's**](launcher.md#background-detached-processes) state. If unhealthy, the daemon will transition to the next defined [`availability`](launcher.md#auto-restart-on-exit) state. If it's `always` this will cause a restart, if none the daemon will transition to the completed or failed state.

```yaml
processes:
  nginx:
    command: "docker run -d --rm -p80:80 --name nginx_test nginx"
    is_daemon: true
    shutdown:
      command: "docker stop nginx_test"
      signal: 15
      timeout_seconds: 5
    liveness_probe:
      exec:
        command: "[ $(docker inspect -f '{{.State.Running}}' nginx_test) = 'true' ]"
        working_dir: /tmp # if not specified the process working dir will be used
      initial_delay_seconds: 5
      period_seconds: 2
      timeout_seconds: 5
      success_threshold: 1
      fail_after_seconds: 4
```

## Readiness Probe

Readiness probes are used to monitor a **process's** state. If the process is unhealthy, it can be restarted according to the [`availability`](launcher.md#auto-restart-on-exit) policy.

```yaml
processes:
  nginx:
    command: "docker run -d --rm -p80:80 --name nginx_test nginx"
    is_daemon: true
    shutdown:
      command: "docker stop nginx_test"
    readiness_probe:
      http_get:
        host: 127.0.0.1
        scheme: http
        path: "/"
        port: 80
      initial_delay_seconds: 5
      startup_period_seconds: 1
      period_seconds: 10
      timeout_seconds: 5
      success_threshold: 1
      startup_fail_after_seconds: 60
      fail_after_seconds: 20
```

Each probe type (`liveness_probe` or `readiness_probe`) can be configured to use one of the 2 mutually exclusive modes:

1. `exec`: Will run a configured `command` and based on the `exit code` decide if the process is in a correct state. 0 indicates success. Any other value indicates failure.
2. `http_get`: For an HTTP probe, the Process Compose sends an HTTP request to the specified path and port to perform the check. Response code 200 indicates success. Any other value indicates failure.
   - `host`: Host name to connect to.
   - `scheme`: Scheme to use for connecting to the host (HTTP or HTTPS). Defaults to HTTP.
   - `path`: Path to access on the HTTP server. Defaults to /.
   - `port`: Number of port to access the process. The number must be in the range 1 to 65535.
   - `headers`: Optional headers to send.
   - `status_code`: Optional. Defaults to 200.

## Configure Probes

Probes have a number of fields that you can use to control the behavior of liveness and readiness checks more precisely:

- `initial_delay_seconds`: Number of seconds after the container has started before liveness or readiness probes are initiated. Defaults to 0 seconds. The minimum value is 0.
- `period_seconds`: How often (in seconds) to perform the probe while the service is healthy, after the first success. Defaults to 10 seconds. The minimum value is 1.
- `startup_period_seconds`: How often (in seconds) to perform the probe until it first succeeds. This allows faster detection of readiness for quick-starting processes while keeping a longer steady-state `period_seconds` afterwards. Defaults to 1 second. The minimum value is 1.
- `unhealthy_period_seconds`: How often (in seconds) to perform the probe while the process is unhealthy, after it had previously become healthy. This allows faster detection of recovery. Defaults to `period_seconds`. The minimum value is 1.
- `timeout_seconds`: Number of seconds after which the probe times out. Defaults to 1 second. The minimum value is 1.
- `success_threshold`: Minimum consecutive successes for the probe to be considered successful after failing. Defaults to 1. Must be 1 for liveness and startup Probes. The minimum value is 1. **Note**: this value is not respected and was added as a placeholder for future implementation.
- `fail_after_seconds`: Wall-clock budget for a consecutive-failure streak after the probe has first succeeded. The budget starts at the first failed check and resets on any success. Defaults to `2 × period_seconds` (20 seconds with the default 10-second period). If the legacy `failure_threshold` is explicitly set, it defaults to `(failure_threshold - 1) × period_seconds` instead. An explicit value overrides both. Minimum 1; 0 or omitted derives the default.
- `startup_fail_after_seconds`: Wall-clock budget while the probe has never succeeded. Defaults to `fail_after_seconds`. Minimum 1; 0 or omitted derives the default.
- `failure_threshold`: Legacy field, still supported but superseded by `fail_after_seconds` ([#573](https://github.com/F1bonacc1/process-compose/issues/573)). When it is explicitly set and `fail_after_seconds` is unset, process-compose derives `fail_after_seconds` as `(failure_threshold - 1) × period_seconds`, using the effective `period_seconds` (which defaults to 10). A `failure_threshold` of 1 derives 0 for `fail_after_seconds`, making the first steady-state failure fatal; do not replace it with `fail_after_seconds: 0`, because 0 is treated as unset and derives the default budget instead.

The `*_period_seconds` fields are polling intervals; the `*_fail_after_seconds` fields are deadlines. The legacy count-to-budget translation preserves the old deadline only when each check completes within `period_seconds`. If a check can run longer (for example `timeout_seconds > period_seconds`), the old count advanced once per completed check and could allow a longer deadline; the wall-clock budget is measured from the first failed check and can give up earlier.

## Auto Restart if not Healthy

In order to ensure that the process is restarted (and not transitioned to a completed state) in case of readiness check fail, please make sure to define the `availability` configuration. For background (`is_daemon=true`) processes, the `restart` policy should be `always`.
