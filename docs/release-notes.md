# Release Notes

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
