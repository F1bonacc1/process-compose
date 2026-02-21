# Release Notes

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
