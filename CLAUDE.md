# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Process Compose is a process orchestrator for non-containerized applications, written in Go. It uses YAML configuration (similar to docker-compose) to define and manage processes with dependency tracking, health checks, scheduling, and more. It operates in multiple modes: TUI (default), HTTP API server, daemon, and MCP server.

## Build & Development Commands

```bash
make build              # Build binary to bin/process-compose (CGO_ENABLED=0)
make test               # Run tests with coverage
make testrace           # Run tests with race detector
make testrace-clean     # Clear test cache, then run with race detector
make lint               # Run golangci-lint (downloads to ./bin/ if needed)
make coverhtml          # Generate HTML coverage report
make swag               # Regenerate Swagger/OpenAPI docs
make schema             # Regenerate JSON schema
make docs               # Regenerate CLI documentation
```

**Run a single test:**
```bash
go test -run TestName ./src/package/...
```

**Run with debug mode:**
```bash
PC_DEBUG_MODE=1 ./bin/process-compose -e .env
```

## Architecture

**Entry point:** `main.go` → `src/cmd/root.go` (Cobra CLI framework)

### Key Packages (all under `src/`)

| Package | Purpose |
|---------|---------|
| `cmd/` | Cobra CLI command definitions |
| `app/` | Core process execution engine — `ProjectRunner` manages lifecycle of all processes, `Process` wraps individual process execution |
| `types/` | Domain models — `Project`, `ProcessConfig`, `ProcessState`, dependency graph |
| `loader/` | YAML config parsing, env var expansion, project merging, validation |
| `config/` | CLI flags, global settings, themes, version info (embedded via ldflags) |
| `api/` | REST API using Gin framework, WebSocket support, Swagger docs |
| `tui/` | Terminal UI using `rivo/tview` — process table, log viewer, terminal view |
| `client/` | HTTP client for remote API communication |
| `health/` | Liveness/readiness probe implementations (HTTP, exec) |
| `scheduler/` | Cron/interval-based process scheduling via `gocron/v2` |
| `mcp/` | MCP server that wraps user-defined processes as MCP tools/resources |
| `mcpctl/` | Control MCP server — exposes tools for agents to introspect and control process-compose itself (list/get/start/stop/restart, logs, dep graph); independent from `mcp/` |
| `pclog/` | Process log buffers with rotation and ANSI color tracking |
| `command/` | Shell command execution abstraction (cross-platform, PTY support) |
| `admitter/` | Namespace-based process filtering |
| `templater/` | Variable/template expansion using envsubst |

### Data Flow

```
CLI/API Input → Cobra Commands (cmd/) → ProjectRunner (app/)
  → Process execution with dependency resolution
  → Health checks (health/) + Scheduling (scheduler/)
  → State updates → TUI (tui/) / API (api/) output
  → Logs → ProcessLogBuffer (pclog/)
```

### Process Lifecycle

States: `Pending → Launching → Running → [Restarting] → Terminating → Completed/Error`

Dependency conditions: `process_started`, `process_healthy`, `process_completed`, `process_log_ready`

### Config Auto-Discovery

Searches for: `compose.yml`, `compose.yaml`, `process-compose.yml`, `process-compose.yaml` with optional `.override.*` files.

## Key Frameworks & Libraries

- **CLI:** `spf13/cobra`
- **HTTP API:** `gin-gonic/gin`
- **TUI:** `rivo/tview` + `gdamore/tcell/v2`
- **YAML:** `gopkg.in/yaml.v3`
- **Logging:** `rs/zerolog`
- **Process metrics:** `shirou/gopsutil/v4`
- **Scheduling:** `go-co-op/gocron/v2`
- **MCP:** `mark3labs/mcp-go`

## Concurrency Model

Heavy use of goroutines, mutexes, and channels for process coordination. `ProjectRunner` uses fine-grained mutexes to manage shared process state maps. Always run `make testrace` to catch data races.

## Rust Component

A Rust client library lives in `crates/` (workspace with `client` and `example` crates). This is a separate component with its own CI workflow.
