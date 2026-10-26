# Release Notes

## [v1.124.0] - 2026-10-27

### New Features

- Extended [template rendering](https://f1bonacc1.github.io/process-compose/configuration/#rendered-parameters) to more process fields: `entrypoint`, `shutdown.command`, `ready_log_line`, `env_file`, `log_configuration.rotation.directory` and `filename`, the exec probe `working_dir`, and HTTP probe `headers`. A template that fails to render now names the process and the field it came from. [`is_template_disabled`](https://f1bonacc1.github.io/process-compose/configuration/#disable-command-rendering) now covers `entrypoint` and `shutdown.command` along with `command`, so a shutdown command or entrypoint with literal Go template syntax (e.g. `docker inspect -f '{{.State.Running}}'`) needs [escaping](https://f1bonacc1.github.io/process-compose/configuration/#template-escaping) or that flag.
- Added `PC_CLIENT_LOG_FILE` to set the [log file of client commands](https://f1bonacc1.github.io/process-compose/logging/#client-commands-and-pc_log_file). Client commands no longer share the server's `PC_LOG_FILE` path (truncating the server's log on every call): they now log to a `-client` sibling of it, e.g. `pc.log` becomes `pc-client.log`.

### Bug Fixes

- Fixed processes with a [`schedule`](https://f1bonacc1.github.io/process-compose/scheduled-processes/) that were added or changed by a project update or reload running once immediately and never being scheduled. They are now registered as they are at startup (a disabled one stays paused), a schedule change alone triggers the update, and a removed or replaced process no longer leaves its old schedule behind, addresses issue #561, by @jihtsan.
- Fixed a `process_healthy` dependent being skipped when it started while its still-running dependency was temporarily not ready after a nonfatal [readiness probe](https://f1bonacc1.github.io/process-compose/health/#readiness-probe) failure, which also stamped the dependency with `exit_code=1`. The dependent now waits for the dependency to become ready again, addresses issue #544.
- Fixed the first [`project update`](https://f1bonacc1.github.io/process-compose/cli/process-compose_project_update/) after a server start restarting every process even when its configuration had not changed.
- Fixed replicas of a scaled process sharing one copy of their probes and log rotation settings, so every replica used the values rendered for the first one: the documented readiness probe with [`port: "404{{.PC_REPLICA_NUM}}"`](https://f1bonacc1.github.io/process-compose/configuration/#auto-inserted-variables) probed replica 0's port from every replica.
- Fixed several log streaming issues: in `attach` mode a process whose log was not being followed showed an empty log pane, log backlogs larger than 256 lines sent over the API were mostly dropped, a log request for several processes ended all of them when the first one finished, and a line written while a viewer subscribed could be shown twice or not at all, addresses issue #539.
- Fixed [`process-compose process logs`](https://f1bonacc1.github.io/process-compose/cli/process-compose_process_logs/) without `--follow` printing a spurious `write close: ... broken pipe` error on stderr after a successful run, by @emeka.
- Fixed CJK and other wide characters being garbled in the terminal view of [interactive processes](https://f1bonacc1.github.io/process-compose/interactive-processes/), where each character overwrote the right half of the wide character before it, addresses issue #535.
- Fixed the TUI process table reshuffling stopped processes on every refresh when sorted by CPU or memory, and the selection jumping to a different process when a refresh reordered the rows. Ties now sort by name, and the column chosen with `--sort` is marked in the header from the start, addresses issue #547.
- Client commands no longer check GitHub for a newer release, since scripts running them in a loop could exhaust the machine's unauthenticated GitHub API rate limit. The remaining update checks identify themselves with a `process-compose/<version>` User-Agent, addresses issue #552.
- Fixed `environment` entries whose value contains `=` (e.g. `DSN=host=db user=app`) being dropped when [merging configuration files](https://f1bonacc1.github.io/process-compose/merge/#adding-and-overriding-configuration) or using [`extends`](https://f1bonacc1.github.io/process-compose/merge/#process-inheritance-with-extends), by @bensynapse.
- Fixed the Swagger UI always targeting `localhost:8080` when the [API server](https://f1bonacc1.github.io/process-compose/client/#rest-api) runs on a different address or port, by @icyleaf.
- Fixed the [REST API](https://f1bonacc1.github.io/process-compose/client/#rest-api) returning a process `namespace` as a plain string when it holds a single namespace, while the OpenAPI spec declares an array of strings, which broke clients generated from the spec. `namespace` is now always an array in JSON output (REST responses, the process state WebSocket, and `process list`, `process get` and `process monitor` with `-o json`). Requests and YAML configs still accept a plain string. Anything that expects a string needs updating, including scripts that compare `namespace` to a string and process-compose client binaries older than v1.122.0, addresses issue #538.

---

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
