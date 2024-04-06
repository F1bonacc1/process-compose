# Remote Client

## REST API

Open API client and documentation is available on: http://localhost:8080

Default port is `8080`. Specify your own port:

```shell
process-compose -p 8080
```

Alternatively use `PC_PORT_NUM` environment variable:

```shell
PC_PORT_NUM=8080 process-compose
```

## Unix Domain Sockets (UDS)

Instead of TCP communication mode, on *nix based systems, you can use Unix Domain Sockets (on the same host only).

There are 3 configuration options:

1. Auto socket path based on `PID`: `process-compose -U` will start Process Compose in UDS mode and create a socket file under `<TempDir>/process-compose-<pid>.sock`
2. Manual socket path with CLI flag: `process-compose --unix-socket /path/to/socket/file` will start Process Compose in UDS mode and create the specified socket file. The directory should exist.
3. Manual socket path with environment variable: `PC_SOCKET_PATH="/path/to/socket/file" process-compose` will start Process Compose in UDS mode and create the specified socket file. The directory should exist.

## Client Mode

Process compose can also connect to itself as a client. Available commands:

#### Processes List

```shell
process-compose process list #lists available processes
```

#### Process Start

```shell
process-compose process start [PROCESS] #starts one of the available non running processes
```

#### Process Stop

```shell
process-compose process stop [PROCESS] #stops one of the running processes
```

#### Process Restart

```shell
process-compose process restart [PROCESS] #restarts one of the available processes
```

Restart will wait `process.availability.backoff_seconds` seconds between `stop` and `start` of the process. If not configured the default value is 1s.

> :bulb: New remote commands are added constantly. For full list run:
```shell
process-compose --help
```

By default, the client will try to use the default port `8080` and default address `localhost` to connect to the locally running instance of process-compose. You can provide deferent values:

```shell
process-compose -p PORT process -a ADDRESS list
```

## TUI Client

For situations when process-compose was started in headless mode `-t=false`, another process-compose instance (client) can run in a fully remote TUI mode:

```bash
process-compose attach
```

The client can connect to a:

- Remote server
- Docker container
- Headless and TUI process-compose instances

In remote mode the Process Compose logo will be replaced from 🔥 to ⚡and show a remote server `hostname` instead of a local `hostname`.