package types

import (
	"fmt"
	"strings"
)

const mcpCtlTransportSSE = "sse"

// MCPCtlServerConfig defines the configuration for the Control MCP server,
// which exposes tools for introspecting and controlling process-compose itself
// (list/get/start/stop/restart processes, read logs, search logs, dependency graph).
//
// This is distinct from MCPServerConfig (mcp_server:), which wraps user-defined
// processes as MCP tools. The two servers are independent and may run simultaneously.
type MCPCtlServerConfig struct {
	Host      string `yaml:"host,omitempty"`
	Port      int    `yaml:"port,omitempty"`
	Transport string `yaml:"transport,omitempty"` // Optional: defaults to "sse"
}

// IsEnabled returns true if the Control MCP server is configured.
//
// An empty block (mcpctl_server: {}) is treated as disabled — a user must
// set at least one field to opt in. Mirrors MCPServerConfig.IsEnabled.
func (m *MCPCtlServerConfig) IsEnabled() bool {
	return m != nil && (m.Transport != "" || m.Host != "" || m.Port > 0)
}

// IsSSE returns true if transport is sse (or default). Only SSE is supported.
func (m *MCPCtlServerConfig) IsSSE() bool {
	if m == nil {
		return false
	}
	return m.Transport == "" || strings.ToLower(m.Transport) == mcpCtlTransportSSE
}

// Validate checks if the Control MCP server configuration is valid.
//
// Only SSE transport is supported. Under `up`, stdio is already owned by the
// TUI or the sibling mcp_server stdio transport, so mcpctl over stdio has no
// practical place to send its traffic.
func (m *MCPCtlServerConfig) Validate() error {
	if m == nil {
		return nil
	}

	if m.Transport != "" && strings.ToLower(m.Transport) != mcpCtlTransportSSE {
		return fmt.Errorf("invalid mcpctl_server transport: %s (only %q is supported)", m.Transport, mcpCtlTransportSSE)
	}

	if m.Host == "" {
		return fmt.Errorf("mcpctl_server requires host")
	}
	if m.Port <= 0 {
		return fmt.Errorf("mcpctl_server requires a valid port")
	}

	return nil
}
