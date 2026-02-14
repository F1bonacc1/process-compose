package mcp

import (
	"testing"
)

func TestParseJSONIfValid(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantData bool
		wantJSON bool
	}{
		{
			name:     "empty string",
			output:   "",
			wantData: false,
			wantJSON: false,
		},
		{
			name:     "plain text",
			output:   "Hello World",
			wantData: false,
			wantJSON: false,
		},
		{
			name:     "valid JSON object",
			output:   `{"status": "success", "data": [1, 2, 3]}`,
			wantData: true,
			wantJSON: true,
		},
		{
			name:     "valid JSON array",
			output:   `[{"name": "item1"}, {"name": "item2"}]`,
			wantData: true,
			wantJSON: true,
		},
		{
			name:     "JSON with whitespace",
			output:   "  \n\t  {\"key\": \"value\"}  \n  ",
			wantData: true,
			wantJSON: true,
		},
		{
			name:     "invalid JSON - looks like object but not valid",
			output:   `{invalid json}`,
			wantData: false,
			wantJSON: false,
		},
		{
			name:     "JSON with trailing comma",
			output:   `{"key": "value",}`,
			wantData: false,
			wantJSON: false,
		},
		{
			name:     "just braces - not JSON",
			output:   "{}",
			wantData: true,
			wantJSON: true,
		},
		{
			name:     "empty array",
			output:   "[]",
			wantData: true,
			wantJSON: true,
		},
		{
			name:     "text starting with brace",
			output:   "{not json content",
			wantData: false,
			wantJSON: false,
		},
		{
			name:     "text ending with brace",
			output:   "not json content}",
			wantData: false,
			wantJSON: false,
		},
		{
			name:     "nested JSON object",
			output:   `{"outer": {"inner": "value"}}`,
			wantData: true,
			wantJSON: true,
		},
		{
			name:     "JSON with numbers",
			output:   `{"count": 42, "price": 19.99}`,
			wantData: true,
			wantJSON: true,
		},
		{
			name:     "JSON with booleans and null",
			output:   `{"active": true, "deleted": false, "data": null}`,
			wantData: true,
			wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, isJSON := parseJSONIfValid(tt.output)
			if isJSON != tt.wantJSON {
				t.Errorf("parseJSONIfValid() isJSON = %v, want %v", isJSON, tt.wantJSON)
			}
			if tt.wantData && data == "" {
				t.Errorf("parseJSONIfValid() data = nil, want non-nil")
			}
			if !tt.wantData && data != "" {
				t.Errorf("parseJSONIfValid() data = %v, want nil", data)
			}
		})
	}
}
