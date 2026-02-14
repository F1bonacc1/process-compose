package mcp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/f1bonacc1/process-compose/src/types"
)

// SubstituteArguments replaces @{arg} and @{arg:default} patterns with actual values
// Returns the modified command string
func SubstituteArguments(input string, args map[string]interface{}, argDefs []types.MCPArgument) (string, error) {
	// Build argument definition lookup
	argDefLookup := make(map[string]types.MCPArgument)
	for _, def := range argDefs {
		argDefLookup[def.Name] = def
	}

	// Find escaped patterns and replace with placeholders
	escapedPattern := regexp.MustCompile(`\\@\{([^}]+)\}`)
	escapedMap := make(map[string]string)
	escapedCount := 0

	result := escapedPattern.ReplaceAllStringFunc(input, func(match string) string {
		placeholder := fmt.Sprintf("__ESCAPED_%d__", escapedCount)
		escapedCount++
		// Store without the backslash
		escapedMap[placeholder] = match[1:]
		return placeholder
	})

	// Track if we have any unmatched required arguments
	var missingArgs []string

	// Find and replace argument patterns
	pattern := regexp.MustCompile(`@\{([^}]+)\}`)

	result = pattern.ReplaceAllStringFunc(result, func(match string) string {
		content := match[2 : len(match)-1] // Remove @{ and }

		var argName string
		var hasDefault bool
		var defaultValue string

		// Check for default value syntax
		if idx := strings.Index(content, ":"); idx > 0 {
			argName = content[:idx]
			hasDefault = true
			defaultValue = content[idx+1:]
		} else {
			argName = content
		}

		// Look up argument value
		value, valueExists := args[argName]

		// If value not provided, check for default
		if !valueExists {
			if hasDefault {
				// Use the default from the pattern
				return defaultValue
			}

			// Check for default from argument definition
			if def, exists := argDefLookup[argName]; exists && def.Default != "" {
				value = def.Default
				valueExists = true
			}
		}

		// If still no value, check if required
		if !valueExists {
			if def, exists := argDefLookup[argName]; exists && def.Required {
				// Track missing required argument
				missingArgs = append(missingArgs, argName)
				return match // Return unchanged for now
			}
			// Optional arg without value - return empty string
			return ""
		}

		// Format the value based on type
		def := argDefLookup[argName]
		formatted, err := formatArgValue(value, def.Type)
		if err != nil {
			// Return unchanged on error, let validation handle it
			return match
		}

		return formatted
	})

	// Restore escaped patterns (remove placeholder, keep as @{arg})
	for placeholder, original := range escapedMap {
		result = strings.Replace(result, placeholder, original, -1)
	}

	// Return error if any required arguments were missing
	if len(missingArgs) > 0 {
		return result, fmt.Errorf("missing required argument(s): %s", strings.Join(missingArgs, ", "))
	}

	return result, nil
}

// SubstituteProcessConfig creates a copy of the process config with substituted arguments
func SubstituteProcessConfig(proc *types.ProcessConfig, args map[string]interface{}) (*types.ProcessConfig, error) {
	if proc.MCP == nil || !proc.MCP.IsTool() {
		return proc, nil
	}

	// Create a copy
	modified := *proc

	// Substitute in command
	if proc.Command != "" {
		substituted, err := SubstituteArguments(proc.Command, args, proc.MCP.Arguments)
		if err != nil {
			return nil, fmt.Errorf("failed to substitute arguments in command: %w", err)
		}
		modified.Command = substituted
	}

	// Substitute in args
	if len(proc.Args) > 0 {
		modified.Args = make([]string, len(proc.Args))
		for i, arg := range proc.Args {
			substituted, err := SubstituteArguments(arg, args, proc.MCP.Arguments)
			if err != nil {
				return nil, fmt.Errorf("failed to substitute arguments in args[%d]: %w", i, err)
			}
			modified.Args[i] = substituted
		}
	}

	// Truncate logs before starting
	modified.TruncateLog = true

	return &modified, nil
}

// formatArgValue formats a value based on its type for shell command substitution
// Strings are always double-quoted with proper escaping
// Numbers and booleans are unquoted
func formatArgValue(value interface{}, argType types.MCPArgumentType) (string, error) {
	switch argType {
	case types.MCPArgTypeString:
		str, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("expected string, got %T", value)
		}
		return shellQuote(str), nil

	case types.MCPArgTypeInteger:
		switch v := value.(type) {
		case int:
			return strconv.Itoa(v), nil
		case int64:
			return strconv.FormatInt(v, 10), nil
		case float64:
			// JSON numbers are float64
			if v == float64(int64(v)) {
				return fmt.Sprintf("%.0f", v), nil
			}
			return "", fmt.Errorf("expected integer, got float")
		case string:
			// Try to parse string as integer
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return strconv.FormatInt(i, 10), nil
			}
			return "", fmt.Errorf("cannot convert string '%s' to integer", v)
		default:
			return "", fmt.Errorf("expected integer, got %T", value)
		}

	case types.MCPArgTypeNumber:
		switch v := value.(type) {
		case int:
			return strconv.Itoa(v), nil
		case int64:
			return strconv.FormatInt(v, 10), nil
		case float64:
			return fmt.Sprintf("%g", v), nil
		case string:
			// Try to parse string as number
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return fmt.Sprintf("%g", f), nil
			}
			return "", fmt.Errorf("cannot convert string '%s' to number", v)
		default:
			return "", fmt.Errorf("expected number, got %T", value)
		}

	case types.MCPArgTypeBoolean:
		switch v := value.(type) {
		case bool:
			return strconv.FormatBool(v), nil
		case string:
			// Try to parse string as boolean
			if b, err := strconv.ParseBool(v); err == nil {
				return strconv.FormatBool(b), nil
			}
			return "", fmt.Errorf("cannot convert string '%s' to boolean", v)
		default:
			return "", fmt.Errorf("expected boolean, got %T", value)
		}

	default:
		return "", fmt.Errorf("unknown argument type: %s", argType)
	}
}

// shellQuote wraps a string in double quotes and escapes special characters
func shellQuote(s string) string {
	// Escape backslashes first
	s = strings.ReplaceAll(s, `\`, `\\`)
	// Escape double quotes
	s = strings.ReplaceAll(s, `"`, `\"`)
	// Escape dollar signs to prevent shell expansion
	s = strings.ReplaceAll(s, `$`, `\$`)
	// Escape backticks
	s = strings.ReplaceAll(s, "`", "\\`")

	return fmt.Sprintf(`"%s"`, s)
}
