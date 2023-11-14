# Health Checks

Many applications running for long periods of time eventually transition to broken states, and cannot recover except by being restarted. Process Compose provides liveness and readiness probes to detect and remedy such situations.

Probes configuration and functionality are designed to work similarly to [Kubernetes liveness and readiness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).

## Liveness Probe

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
      failure_threshold: 3
```

## Readiness Probe

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
      period_seconds: 10
      timeout_seconds: 5
      success_threshold: 1
      failure_threshold: 3
```

Each probe type (`liveness_probe` or `readiness_probe`) can be configured to use one of the 2 mutually exclusive modes:

1. `exec`: Will run a configured `command` and based on the `exit code` decide if the process is in a correct state. 0 indicates success. Any other value indicates failure.
2. `http_get`: For an HTTP probe, the Process Compose sends an HTTP request to the specified path and port to perform the check. Response code 200 indicates success. Any other value indicates failure.
   - `host`: Host name to connect to.
   - `scheme`: Scheme to use for connecting to the host (HTTP or HTTPS). Defaults to HTTP.
   - `path`: Path to access on the HTTP server. Defaults to /.
   - `port`: Number of port to access the process. The number must be in the range 1 to 65535.

## Configure Probes

Probes have a number of fields that you can use to control the behavior of liveness and readiness checks more precisely:

- `initial_delay_seconds`: Number of seconds after the container has started before liveness or readiness probes are initiated. Defaults to 0 seconds. The minimum value is 0.
- `period_seconds`: How often (in seconds) to perform the probe. Defaults to 10 seconds. The minimum value is 1.
- `timeout_seconds`: Number of seconds after which the probe times out. Defaults to 1 second. The minimum value is 1.
- `success_threshold`: Minimum consecutive successes for the probe to be considered successful after failing. Defaults to 1. Must be 1 for liveness and startup Probes. The minimum value is 1. **Note**: this value is not respected and was added as a placeholder for future implementation.
- `failure_threshold`: When a probe fails, Process Compose will try `failure_threshold` times before giving up. Giving up in case of liveness probe means restarting the process. In case of readiness probe, the Pod will be marked Unready. Defaults to 3. The minimum value is 1.

## Auto Restart if not Healthy

In order to ensure that the process is restarted (and not transitioned to a completed state) in case of readiness check fail, please make sure to define the `availability` configuration. For background (`is_daemon=true`) processes, the `restart` policy should be `always`.
