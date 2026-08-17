# Release Notes

## [v1.122.0] - 2026-08-18

### New Features

- Added [file watching](https://f1bonacc1.github.io/process-compose/watch/): a per-process `watch` block restarts a process when its files change, with include/exclude globs, a configurable `debounce`, and `cascade: true` to restart the process's dependents transitively (in dependency order, so a rebuilt artifact is ready before its consumers restart). A process that has exited while its watch is still armed reports the new `Watching` state, and the TUI names the file that caused each restart. Watch-triggered restarts skip the restart backoff and reset the restart counter, since they are user intent rather than a crash loop. File watching can be turned off entirely with `--no-watch` / `PC_NO_WATCH`.
- Added support for [multiple namespaces per process](https://f1bonacc1.github.io/process-compose/configuration/#multiple-namespaces): `namespace` now accepts either a single string or a list of strings (similar to docker-compose profiles), so a shared process (e.g. a database) can be selected and operated as part of any of its namespaces, addresses issue #519.

### Bug Fixes

- Fixed restarting a process shutting the whole project down when that process was configured with `availability.exit_on_end` or `restart: exit_on_failure`. Every exit routed through the same path, and a restarted process does not exit cleanly (a `SIGTERM`ed process reports a non-zero code), so an intentional restart was indistinguishable from the process ending. This affected manual restarts from the TUI, the CLI and the API; it is also what would have made every watch-triggered restart end the project.
- Fixed [namespace operations](https://f1bonacc1.github.io/process-compose/configuration/#namespace-operations) skipping processes that were not started: `process-compose namespace start <ns>` now starts every configured member of the namespace, including processes excluded from an `up <process>...` selection or marked `disabled: true`, instead of failing with "namespace not found (no processes assigned)", addresses issue #528.
- Fixed a [project update or reload](https://f1bonacc1.github.io/process-compose/configuration/#project-edit) starting the processes that were excluded from a `process-compose up <process>...` selection: the selection is now re-applied to the reloaded configuration, the same way the `--namespace` admission policies already were.
- Fixed a project or process update shutting the project down when it restarted the only running process: the momentary gap while the process is stopped is no longer mistaken for the project having completed.
- Fixed stopping or restarting a process while it is still waiting on its [dependencies](https://f1bonacc1.github.io/process-compose/launcher/#define-process-dependencies): the process was left wedged in `Terminating` and its still-waiting instance later started an untracked copy, which `stop`, `start` and `restart` could not manage, which a subsequent `start` duplicated, and which survived `down` as an orphan. A pending process now stops immediately instead of waiting for its dependencies to resolve, and a superseded instance can no longer deregister its replacement, addresses issue #530.
- Fixed the TUI **Namespace Operations** modal silently ignoring failures: a failed namespace start, stop or restart is now reported in an error dialog instead of only being written to the log file.
- Fixed the HTTP API rejecting process names that contain slashes: routes are now matched against the raw percent-encoded path and the client encodes process names as path segments, addresses issue #523.
- Client-mode commands (`down`, `attach`, `list`, `process`, `project`, `namespace`) now write their internal logs to a [separate log file](https://f1bonacc1.github.io/process-compose/logging/#client-commands-and-pc_log_file) (`/tmp/process-compose-$USER-client.log` by default) instead of truncating the log file of a running server, documentation by @soy-chrislo.

---

## [v1.120.0] - 2026-07-11

### New Features

- Added [Solarized Dark and Solarized Light](https://f1bonacc1.github.io/process-compose/tui/#tui-themes) TUI themes, by Danial Pearce.
- Added pruning of cross-namespace dependencies: when running a subset of [namespaces](https://f1bonacc1.github.io/process-compose/configuration/#namespaces), a `depends_on` reference to a process from an unselected namespace is dropped (with a warning) so the selected processes can start, and the namespace selection now persists across project updates and reloads instead of resurrecting excluded processes, addresses issue #517.
- Added debug logging of the exec probe command during health checks - once when probing starts and on probe failures, addresses issue #454.
- Added a notice when requesting CLI logs for an [interactive process](https://f1bonacc1.github.io/process-compose/interactive-processes/#attaching-to-a-process), whose PTY-rendered output is not captured in the log buffer, addresses issue #511.

### Bug Fixes

- Fixed the TUI log viewer mangling log lines containing bracketed text with punctuation or spaces, which was misinterpreted as tview region tags, addresses issue #515.
- Fixed incorrect line wrapping of interactive process output by deferring terminal initialization until the TUI layout is settled and the pane size is known, addresses issue #512.

### Security Fixes

- Fixed a DNS-rebinding vulnerability (GHSA-5gm3-9crp-6g3v) in the [MCP SSE transport](https://f1bonacc1.github.io/process-compose/mcp-server/#sse-transport-security): the SSE listener now validates the `Host` and `Origin` headers (loopback plus optional `trusted_hosts`) and, when an API token is configured, enforces it via `X-PC-Token-Key` or `Authorization: Bearer` before dispatching any MCP request.

---

## [v1.116.0] - 2026-06-16

### New Features

- Added a [`process-compose process send-keys`](https://f1bonacc1.github.io/process-compose/cli/process-compose_process_send-keys/) command to inject keystrokes into a running [interactive process's](https://f1bonacc1.github.io/process-compose/interactive-processes/#sending-keys-programmatically) stdin, plus a [`shutdown.send_keys`](https://f1bonacc1.github.io/process-compose/launcher/#stopping-interactive-processes-with-keystrokes) option to cleanly stop programs that only quit on a keypress, addresses issue #507.
- Added per-process [`success_exit_codes`](https://f1bonacc1.github.io/process-compose/launcher/#successful-exit-codes) to treat selected non-zero exit codes (e.g. `130` from a signal-driven shutdown) as a success, addresses issue #506.
- Added process-level [`extends`](https://f1bonacc1.github.io/process-compose/merge/#process-inheritance-with-extends) so a process can inherit and override the configuration of another process in the same project.
- Added automatic [replica name expansion in `depends_on`](https://f1bonacc1.github.io/process-compose/launcher/#multiple-replicas-of-a-process), letting processes depend on a replicated group or on a specific replica, addresses issue #496, by Pablo Castellazzi.
- Added a `PC_NAMESPACES` environment variable as an alternative to `--namespace`/`-n` for selecting which [namespaces](https://f1bonacc1.github.io/process-compose/configuration/#namespaces) to run, addresses issue #491, by David Danier.
- Added the [`pc_process_logs_search` (BM25 log search) and `pc_project_dependency_graph`](https://f1bonacc1.github.io/process-compose/mcp-server/#built-in-control-tools) built-in MCP control tools, by Jean-Luc Thumm.

### Bug Fixes

- Fixed unfocused interactive processes blocking on a full PTY buffer by ensuring background drainage, addresses issue #508.
- Fixed a readiness cascade where dependents of an updated or restarted process could spuriously fail on a stale, cancelled process object, by Bo He.
- Fixed a deadlock that wedged the logs WebSocket for an entire process under backpressure, by Bo He.
- Fixed reloading a project that uses file-level `extends`, which previously failed with an "already specified in files to load" error after the first successful load, by Eike Haß.

---

## [v1.110.0] - 2026-05-01

### New Features

- Added [process activity and silence monitoring](https://f1bonacc1.github.io/process-compose/tui/#process-activity-monitor) in the TUI, with deduplication of silence notifications.
- Added a process state push-notification stream over WebSocket, plus a [`process-compose process monitor`](https://f1bonacc1.github.io/process-compose/cli/process-compose_process_monitor/) CLI subcommand to subscribe to it, addresses issue #470.
- Added a [command palette](https://f1bonacc1.github.io/process-compose/tui/#command-palette) to the TUI for process management - start, stop, restart, scale, signal, create, and delete.
- Added a [`process-compose analyze critical-chain`](https://f1bonacc1.github.io/process-compose/cli/process-compose_analyze_critical-chain/) subcommand that prints a tree of processes with startup timings, in the spirit of `systemd-analyze critical-chain`, by Ryan Mulligan.
- Added [built-in MCP control tools](https://f1bonacc1.github.io/process-compose/mcp-server/#built-in-control-tools) (`pc_*`) so MCP clients can manage the running project, opt-in via `expose_control_tools: true`.
- Added Shift+Tab support and xterm-style modifier key sequence encoding in the terminal view.
- Added OSC 52 clipboard status notifications via the glippy v1.2.0 upgrade.

### Bug Fixes

- Fixed daemons being included in total CPU and RAM calculations.
- Fixed process CPU metric retrieval to use `PercentWithContext` with interval 0, addresses issue #471.
- Fixed the casing mismatch between Swagger docs (lower) and the REST API (capital), addresses issue #457.

---

## [v1.103.0] - 2026-04-03

### New Features

- Added text selection and clipboard copying support in terminal views.
- Added application cursor key mode (DECCKM) support for interactive terminal processes.
- Added context-aware help that updates dynamically when the terminal view is focused.

### Bug Fixes

- Fixed extra brackets appearing in text containing ANSI escape sequences, addresses issue #449.
- Fixed zombie processes not being reaped after exit.
- Fixed stale terminating states and pipe blocking during process shutdown, addresses issue #450.
- Fixed missing ANSI escape code support for disabling bold, underline, and reverse text styles in terminal views.

---

## [v1.100.0] - 2026-03-20

### New Features

- **POSIX Signal Support**: Added support for sending custom POSIX signals to processes via the TUI, by Kevin J. Lynagh.
- **Process Environment Files**: Added process-specific `env_file` support to load environment variables from dedicated files, addresses issue ([#406](https://github.com/F1bonacc1/process-compose/issues/406)).
- **Log Color Control**: Added the `--log-no-color` CLI flag and `PC_LOG_NO_COLOR` environment variable to disable color output in log files, addresses issue ([#440](https://github.com/F1bonacc1/process-compose/issues/440)).
- **Self-Update Capability**: Added a self-update command to securely download and install new versions.
- **Shutdown Logging**: Added explicit logging when a process exits or is skipped and triggers project shutdown.
- **TUI Footer Links**: Added "Donate" and "Ask Question" links to the TUI footer that open in the default browser.

### Bug Fixes

- Fixed a race condition by waiting for the detached daemon's HTTP server to be ready before proceeding, addresses issue ([#443](https://github.com/F1bonacc1/process-compose/issues/443)) and ([#424](https://github.com/F1bonacc1/process-compose/issues/424)).
- Fixed the process editing loop to correctly exit if a user exits without changing the configuration.
- Fixed incorrectly typed `RestartPolicy` and `ProcessCondition` properties when marshaling for editing.
- Improved concurrency safety in the project runner.

---

## [v1.94.0] - 2026-02-21

### New Features

- **API Token Authentication**: Added support for token-based authentication for the REST API and WebSocket connections. Configurable via `PC_API_TOKEN`, `PC_API_TOKEN_PATH`, or the `--token-file` flag.
- **MCP Server Support**: Integrated Model Context Protocol (MCP) server for dynamic process management and tool execution, supporting both `stdio` and `sse` transports.
- **Template Rendering Control**: Added `is_template_disabled` option to skip Go template rendering for processes containing JSON strings in their commands.
- **JSON Pretty-Print**: Added a toggle for pretty-printing JSON logs in the TUI terminal view.

### Bug Fixes

- Fixed JSON autodetection for processes running in MCP `stdio` mode.

---

## [v1.90.0] - 2026-01-31

### New Features

- **Namespace Operations**: Added support for starting, stopping, and restarting namespaces via CLI (`namespace` command), TUI (new namespace modal), and REST API.
- **Enhanced Port Monitoring**: Added UDP port detection and child process listener detection, so processes that spawn worker children (e.g., uvicorn, npm) now correctly report all open ports, by Jesse Dhillon.
- **Interactive Process Scrolling**: Added scrollback support for interactive processes with mouse wheel and keyboard navigation (`Ctrl+A` followed by arrow keys).

### Bug Fixes

- Fixed `PC_ADDRESS` environment variable not being read correctly, by Lazar Bodor.
- Improved Windows process stopping by dynamically building taskkill arguments and gracefully handling process not found errors.

### Maintenance

- **Testing**: Improved test reliability and cross-platform compatibility, particularly for Windows.
- **CI/CD**: Added `clean-testrace` target to Makefile.

---

## [v1.87.0] - 2026-01-03

### New Features

- **Process Dependency Graph**: Added a comprehensive visualization feature available via CLI (`graph` command), TUI (`Ctrl+Q`), and REST API (`graph`). Supports multiple output formats: ASCII, Mermaid, JSON, and YAML.
- **Scheduled Processes**: Introduced support for cron and interval-based process execution.
- **Enhanced TUI Interactivity**: Added mouse support to the terminal view and a configurable escape character for interactive processes.

### Bug Fixes

- Fixed a bug where environment variables were not correctly applied to foreground processes ([#427](https://github.com/F1bonacc1/process-compose/issues/427)).
- Fixed missing version information when the application is installed using `go install` ([#426](https://github.com/F1bonacc1/process-compose/issues/426)).
- Resolved various styling and layout issues for interactive processes in the TUI.

### Maintenance

- **CI/CD**: Expanded the CI build matrix to include Windows and improved test coverage across all packages.
- **Dependencies**: Updated Go modules dependencies.
