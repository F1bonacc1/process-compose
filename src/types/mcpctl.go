package types

import (
	"fmt"
	"strings"
	"time"
)

const (
	mcpCtlTransportSSE   = "sse"
	mcpCtlTransportStdio = "stdio"
)

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
	Timeout   string `yaml:"timeout,omitempty"`   // Optional: defaults to "5m"
}

// IsEnabled returns true if the Control MCP server is configured.
func (m *MCPCtlServerConfig) IsEnabled() bool {
	return m != nil && (m.IsStdio() || m.Transport != "" || m.Host != "" || m.Port > 0)
}

// IsSSE returns true if transport is sse (or default).
func (m *MCPCtlServerConfig) IsSSE() bool {
	if m == nil {
		return false
	}
	return m.Transport == "" || m.Transport == mcpCtlTransportSSE
}

// IsStdio returns true if transport is stdio.
func (m *MCPCtlServerConfig) IsStdio() bool {
	if m == nil {
		return false
	}
	return m.getTransport() == mcpCtlTransportStdio
}

func (m *MCPCtlServerConfig) getTransport() string {
	if m == nil || m.Transport == "" {
		return mcpCtlTransportSSE
	}
	return strings.ToLower(m.Transport)
}

// Validate checks if the Control MCP server configuration is valid.
func (m *MCPCtlServerConfig) Validate() error {
	if m == nil {
		return nil
	}

	transport := m.getTransport()
	if transport != mcpCtlTransportSSE && transport != mcpCtlTransportStdio {
		return fmt.Errorf("invalid mcpctl_server transport: %s (must be %q or %q)", m.Transport, mcpCtlTransportSSE, mcpCtlTransportStdio)
	}

	if transport == mcpCtlTransportStdio {
		return nil
	}

	if m.Host == "" {
		return fmt.Errorf("mcpctl_server SSE transport requires host")
	}
	if m.Port <= 0 {
		return fmt.Errorf("mcpctl_server SSE transport requires a valid port")
	}

	return nil
}

// GetTimeout returns the timeout duration for the Control MCP server.
// Returns 0 if not set (caller should use default).
func (m *MCPCtlServerConfig) GetTimeout() (time.Duration, error) {
	if m == nil || m.Timeout == "" {
		return 0, nil
	}
	return time.ParseDuration(m.Timeout)
}
