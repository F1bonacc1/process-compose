package mcpctl

import (
	"io"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
)

// Manager handles the Control MCP server lifecycle, mirroring the shape of
// src/mcp.MCPManager but entirely independent from it.
type Manager struct {
	server *Server
	runner ProcessRunner
}

// NewManager constructs a Control MCP manager. Returns nil if cfg is nil or
// not enabled. A nil manager is safe to Start/Stop (no-op).
func NewManager(runner ProcessRunner, cfg *types.MCPCtlServerConfig, processes types.Processes) *Manager {
	if cfg == nil || !cfg.IsEnabled() {
		return nil
	}

	srv := NewServer(runner, cfg, processes)
	if srv == nil {
		return nil
	}

	log.Info().Msg("Control MCP server initialized")

	return &Manager{
		server: srv,
		runner: runner,
	}
}

// Start starts the Control MCP server.
func (m *Manager) Start() error {
	if m == nil || m.server == nil {
		return nil
	}
	return m.server.Start()
}

// Stop stops the Control MCP server.
func (m *Manager) Stop() error {
	if m == nil || m.server == nil {
		return nil
	}
	return m.server.Stop()
}

// SetStdio sets stdin/stdout for the stdio transport.
func (m *Manager) SetStdio(stdin io.Reader, stdout io.Writer) {
	if m == nil || m.server == nil {
		return
	}
	m.server.SetStdio(stdin, stdout)
}
