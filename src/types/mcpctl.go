package types

import "fmt"

// MCPCtlServerConfig defines the configuration for the Control MCP server,
// which exposes tools for introspecting and controlling process-compose itself
// (list/get/start/stop/restart processes, read logs, search logs, dependency graph).
//
// This is distinct from MCPServerConfig (mcp_server:), which wraps user-defined
// processes as MCP tools. The two servers are independent and may run simultaneously.
//
// Transport is always SSE. stdio isn't offered because under `up` the TUI or
// the sibling mcp_server stdio transport already owns the process stdio.
type MCPCtlServerConfig struct {
	Host string `yaml:"host,omitempty"`
	Port int    `yaml:"port,omitempty"`
}

// IsEnabled returns true if the Control MCP server is configured.
//
// An empty block (mcpctl_server: {}) is treated as disabled — a user must
// set at least one field to opt in. Mirrors MCPServerConfig.IsEnabled.
func (m *MCPCtlServerConfig) IsEnabled() bool {
	return m != nil && (m.Host != "" || m.Port > 0)
}

// Validate checks if the Control MCP server configuration is valid.
func (m *MCPCtlServerConfig) Validate() error {
	if m == nil {
		return nil
	}
	if m.Host == "" {
		return fmt.Errorf("mcpctl_server requires host")
	}
	if m.Port <= 0 {
		return fmt.Errorf("mcpctl_server requires a valid port")
	}
	return nil
}
