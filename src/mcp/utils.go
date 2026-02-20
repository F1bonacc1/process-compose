package mcp

import (
	"encoding/json"
	"strings"
)

// parseJSONIfValid attempts to parse the output as JSON.
// Returns the parsed data and true if valid JSON, nil and false otherwise.
func parseJSONIfValid(output string) (any, bool) {
	trimmed := strings.TrimSpace(output)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		var data any
		if err := json.Unmarshal([]byte(trimmed), &data); err == nil {
			return data, true
		}
	}
	return nil, false
}
