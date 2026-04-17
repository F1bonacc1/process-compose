package types

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// MCPServerConfig defines the top-level MCP server configuration
type MCPServerConfig struct {
	Host               string `yaml:"host,omitempty"`
	Port               int    `yaml:"port,omitempty"`
	Transport          string `yaml:"transport,omitempty"`            // Optional: defaults to "sse"
	Timeout            string `yaml:"timeout,omitempty"`              // Optional: defaults to "5m"
	ExposeControlTools bool   `yaml:"expose_control_tools,omitempty"` // Optional: when true, expose built-in pc_* control tools
}

// IsEnabled returns true if MCP server is configured
func (m *MCPServerConfig) IsEnabled() bool {
	return m != nil && (m.IsStdio() || m.Transport != "" || m.Host != "" || m.Port > 0 || m.ExposeControlTools)
}

// IsSSE returns true if transport is sse (or default)
func (m *MCPServerConfig) IsSSE() bool {
	if m == nil {
		return false
	}
	// If transport is not specified, default to sse
	return m.Transport == "" || m.Transport == "sse"
}

// IsStdio returns true if transport is stdio
func (m *MCPServerConfig) IsStdio() bool {
	if m == nil {
		return false
	}
	return m.getTransport() == "stdio"
}

// getTransport returns the transport type, defaulting to "sse" if not specified
func (m *MCPServerConfig) getTransport() string {
	if m == nil || m.Transport == "" {
		return "sse"
	}
	return strings.ToLower(m.Transport)
}

// Validate checks if the MCP server configuration is valid
func (m *MCPServerConfig) Validate() error {
	if m == nil {
		return nil
	}

	transport := m.getTransport()
	if transport != "sse" && transport != "stdio" {
		return fmt.Errorf("invalid MCP transport: %s (must be 'sse' or 'stdio')", m.Transport)
	}

	if transport == "stdio" {
		return nil
	}

	// SSE is the default, so always require host and port
	if m.Host == "" {
		return fmt.Errorf("MCP SSE transport requires host")
	}
	if m.Port <= 0 {
		return fmt.Errorf("MCP SSE transport requires a valid port")
	}

	return nil
}

// GetTimeout returns the timeout duration for the MCP server
// Returns 0 if not set (caller should use default)
func (m *MCPServerConfig) GetTimeout() (time.Duration, error) {
	if m == nil || m.Timeout == "" {
		return 0, nil
	}
	return time.ParseDuration(m.Timeout)
}

// MCPProcessType represents the type of MCP process
type MCPProcessType string

const (
	MCPProcessTypeTool     MCPProcessType = "tool"
	MCPProcessTypeResource MCPProcessType = "resource"
)

// MCPArgumentType represents the type of an MCP argument
type MCPArgumentType string

const (
	MCPArgTypeString  MCPArgumentType = "string"
	MCPArgTypeNumber  MCPArgumentType = "number"
	MCPArgTypeBoolean MCPArgumentType = "boolean"
	MCPArgTypeInteger MCPArgumentType = "integer"
)

// IsValid checks if the argument type is valid
func (t MCPArgumentType) IsValid() bool {
	switch t {
	case MCPArgTypeString, MCPArgTypeNumber, MCPArgTypeBoolean, MCPArgTypeInteger:
		return true
	}
	return false
}

// MCPArgument defines a single argument for an MCP tool
type MCPArgument struct {
	Name        string          `yaml:"name"`
	Type        MCPArgumentType `yaml:"type"`
	Description string          `yaml:"description,omitempty"`
	Required    bool            `yaml:"required,omitempty"`
	Default     string          `yaml:"default,omitempty"`
}

// MCPProcessConfig defines the MCP-specific configuration for a process
type MCPProcessConfig struct {
	Type      MCPProcessType `yaml:"type"`
	Arguments []MCPArgument  `yaml:"arguments,omitempty"`
	Timeout   string         `yaml:"timeout,omitempty"` // Optional: overrides global timeout
}

// IsTool returns true if this is a tool-type process
func (m *MCPProcessConfig) IsTool() bool {
	return m != nil && m.Type == MCPProcessTypeTool
}

// IsResource returns true if this is a resource-type process
func (m *MCPProcessConfig) IsResource() bool {
	return m != nil && m.Type == MCPProcessTypeResource
}

// GetTimeout returns the timeout duration for this MCP process
// Returns 0 if not set (caller should fall back to global timeout)
func (m *MCPProcessConfig) GetTimeout() (time.Duration, error) {
	if m == nil || m.Timeout == "" {
		return 0, nil
	}
	return time.ParseDuration(m.Timeout)
}

// Validate checks if the MCP process configuration is valid
func (m *MCPProcessConfig) Validate(processName string, command string, args []string) error {
	if m == nil {
		return nil
	}

	// Validate type
	if m.Type != MCPProcessTypeTool && m.Type != MCPProcessTypeResource {
		return fmt.Errorf("process %s: invalid MCP type: %s (must be 'tool' or 'resource')", processName, m.Type)
	}

	// Resources should not have arguments
	if m.IsResource() && len(m.Arguments) > 0 {
		return fmt.Errorf("process %s: resources cannot have arguments", processName)
	}

	// Build argument name lookup
	argLookup := make(map[string]bool)
	for _, arg := range m.Arguments {
		if arg.Name == "" {
			return fmt.Errorf("process %s: argument must have a name", processName)
		}
		if !arg.Type.IsValid() {
			return fmt.Errorf("process %s: argument %s has invalid type: %s", processName, arg.Name, arg.Type)
		}
		argLookup[arg.Name] = true
	}

	// For tools, validate @{...} patterns in command and args
	if m.IsTool() {
		// Check command
		if command != "" {
			refs, err := ExtractArgReferences(command)
			if err != nil {
				return fmt.Errorf("process %s: %w", processName, err)
			}
			for _, ref := range refs {
				if !argLookup[ref.Name] {
					validArgs := make([]string, 0, len(m.Arguments))
					for _, a := range m.Arguments {
						validArgs = append(validArgs, a.Name)
					}
					return fmt.Errorf("process %s: undefined argument '@{%s}' in command. Valid arguments: %s",
						processName, ref.Name, strings.Join(validArgs, ", "))
				}
			}
		}

		// Check args
		for i, arg := range args {
			refs, err := ExtractArgReferences(arg)
			if err != nil {
				return fmt.Errorf("process %s: invalid pattern in args[%d]: %w", processName, i, err)
			}
			for _, ref := range refs {
				if !argLookup[ref.Name] {
					validArgs := make([]string, 0, len(m.Arguments))
					for _, a := range m.Arguments {
						validArgs = append(validArgs, a.Name)
					}
					return fmt.Errorf("process %s: undefined argument '@{%s}' in args[%d]. Valid arguments: %s",
						processName, ref.Name, i, strings.Join(validArgs, ", "))
				}
			}
		}
	}

	// Validate timeout if specified
	if m.Timeout != "" {
		if _, err := time.ParseDuration(m.Timeout); err != nil {
			return fmt.Errorf("process %s: invalid timeout '%s': %v", processName, m.Timeout, err)
		}
	}

	return nil
}

// ArgReference represents a found argument reference in a command
type ArgReference struct {
	Name         string
	HasDefault   bool
	DefaultValue string
	IsEscaped    bool
}

// ExtractArgReferences extracts @{...} patterns from a command string
// Supports @{arg}, @{arg:default}, and escaped \@{arg}
func ExtractArgReferences(command string) ([]ArgReference, error) {
	var refs []ArgReference
	seen := make(map[string]bool) // Track seen argument names for deduplication

	// Regex to match @{arg}, @{arg:default}, and \@{arg}
	// Pattern: (?<!\)@\{([^}]+)\}
	// But Go regex doesn't support negative lookbehind, so we handle escaping differently
	escapedPattern := regexp.MustCompile(`\\@\{([^}]+)\}`)
	pattern := regexp.MustCompile(`@\{([^}]+)\}`)

	// Find escaped patterns first and replace with placeholder to avoid matching
	escapedMap := make(map[string]string)
	escapedCount := 0
	commandWithPlaceholders := escapedPattern.ReplaceAllStringFunc(command, func(match string) string {
		placeholder := fmt.Sprintf("__ESCAPED_%d__", escapedCount)
		escapedCount++
		escapedMap[placeholder] = match[2:] // Remove the backslash
		return placeholder
	})

	// Now find non-escaped patterns
	matches := pattern.FindAllStringSubmatch(commandWithPlaceholders, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		content := match[1]
		ref := ArgReference{}

		// Check for default value syntax: arg:default
		if idx := strings.Index(content, ":"); idx > 0 {
			ref.Name = content[:idx]
			ref.HasDefault = true
			ref.DefaultValue = content[idx+1:]
		} else {
			ref.Name = content
		}

		// Skip duplicates
		if seen[ref.Name] {
			continue
		}
		seen[ref.Name] = true

		// Validate argument name (alphanumeric and underscore only)
		if !isValidArgName(ref.Name) {
			return nil, fmt.Errorf("invalid argument name '%s' in pattern '@{%s}'", ref.Name, content)
		}

		refs = append(refs, ref)
	}

	return refs, nil
}

// isValidArgName checks if an argument name contains only valid characters
func isValidArgName(name string) bool {
	if name == "" {
		return false
	}
	for _, ch := range name {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
			return false
		}
	}
	return true
}
