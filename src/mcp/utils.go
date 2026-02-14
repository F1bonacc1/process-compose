package mcp

import (
	"encoding/json"
	"strings"
)

// parseJSONIfValid attempts to parse the output as JSON.
// Returns the parsed data and true if valid JSON, nil and false otherwise.
func parseJSONIfValid(output string) (string, bool) {
	trimmed := strings.TrimSpace(output)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		if json.Valid([]byte(trimmed)) {
			return trimmed, true
		}
	}
	return "", false
}
