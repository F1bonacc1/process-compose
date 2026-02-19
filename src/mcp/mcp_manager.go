package mcp

import (
	"io"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
)

// MCPManager handles MCP server lifecycle
type MCPManager struct {
	server *Server
	runner ProcessRunner
}

// NewMCPManager creates a new MCP manager if MCP is configured
func NewMCPManager(runner ProcessRunner, mcpConfig *types.MCPServerConfig, processes types.Processes) *MCPManager {
	if mcpConfig == nil || !mcpConfig.IsEnabled() {
		return nil
	}

	server := NewServer(runner, mcpConfig)

	// Register all MCP-enabled processes
	for _, proc := range processes {
		if proc.IsMCP() {
			if err := server.RegisterProcess(&proc); err != nil {
				log.Error().Err(err).Str("process", proc.Name).Msg("Failed to register MCP process")
			}
		}
	}

	// Only keep the server if there are registered processes
	if len(server.GetRegisteredProcesses()) == 0 {
		log.Info().Msg("MCP server configured but no MCP processes found - skipping server start")
		return nil
	}

	log.Info().Int("count", len(server.GetRegisteredProcesses())).Msg("MCP server initialized")

	return &MCPManager{
		server: server,
		runner: runner,
	}
}

// Start starts the MCP server
func (m *MCPManager) Start() error {
	if m == nil || m.server == nil {
		return nil
	}
	return m.server.Start()
}

// Stop stops the MCP server
func (m *MCPManager) Stop() error {
	if m == nil || m.server == nil {
		return nil
	}
	return m.server.Stop()
}

// SetStdio sets the stdin and stdout for the MCP server when using stdio transport
func (m *MCPManager) SetStdio(stdin io.Reader, stdout io.Writer) {
	if m == nil || m.server == nil {
		return
	}
	m.server.SetStdio(stdin, stdout)
}
