package mcp

import (
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
)

func TestSubstituteArguments(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		args    map[string]interface{}
		argDefs []types.MCPArgument
		want    string
		wantErr bool
	}{
		{
			name:    "simple string substitution",
			input:   "grep @{pattern} file.txt",
			args:    map[string]interface{}{"pattern": "error"},
			argDefs: []types.MCPArgument{{Name: "pattern", Type: types.MCPArgTypeString}},
			want:    `grep "error" file.txt`,
			wantErr: false,
		},
		{
			name:    "integer unquoted",
			input:   "head -n @{count}",
			args:    map[string]interface{}{"count": 50},
			argDefs: []types.MCPArgument{{Name: "count", Type: types.MCPArgTypeInteger}},
			want:    "head -n 50",
			wantErr: false,
		},
		{
			name:    "default value used",
			input:   "head -n @{limit:100}",
			args:    map[string]interface{}{},
			argDefs: []types.MCPArgument{{Name: "limit", Type: types.MCPArgTypeInteger, Default: "100"}},
			want:    "head -n 100",
			wantErr: false,
		},
		{
			name:    "default value from pattern",
			input:   "head -n @{limit:50}",
			args:    map[string]interface{}{},
			argDefs: []types.MCPArgument{{Name: "limit", Type: types.MCPArgTypeInteger}},
			want:    "head -n 50",
			wantErr: false,
		},
		{
			name:    "default value overridden",
			input:   "head -n @{limit:100}",
			args:    map[string]interface{}{"limit": 25},
			argDefs: []types.MCPArgument{{Name: "limit", Type: types.MCPArgTypeInteger}},
			want:    "head -n 25",
			wantErr: false,
		},
		{
			name:    "escaped pattern",
			input:   `echo "Use \@{pattern} syntax"`,
			args:    map[string]interface{}{"pattern": "test"},
			argDefs: []types.MCPArgument{{Name: "pattern", Type: types.MCPArgTypeString}},
			want:    `echo "Use @{pattern} syntax"`,
			wantErr: false,
		},
		{
			name:    "string with quotes escaped",
			input:   "echo @{message}",
			args:    map[string]interface{}{"message": `say "hello"`},
			argDefs: []types.MCPArgument{{Name: "message", Type: types.MCPArgTypeString}},
			want:    `echo "say \"hello\""`,
			wantErr: false,
		},
		{
			name:  "multiple arguments",
			input: "grep @{pattern} @{filename} | head -n @{limit:100}",
			args:  map[string]interface{}{"pattern": "error", "filename": "/var/log/app.log"},
			argDefs: []types.MCPArgument{
				{Name: "pattern", Type: types.MCPArgTypeString},
				{Name: "filename", Type: types.MCPArgTypeString},
				{Name: "limit", Type: types.MCPArgTypeInteger, Default: "100"},
			},
			want:    `grep "error" "/var/log/app.log" | head -n 100`,
			wantErr: false,
		},
		{
			name:    "optional arg not provided",
			input:   "echo @{optional} done",
			args:    map[string]interface{}{},
			argDefs: []types.MCPArgument{{Name: "optional", Type: types.MCPArgTypeString, Required: false}},
			want:    "echo  done",
			wantErr: false,
		},
		{
			name:    "boolean value",
			input:   "if @{verbose}; then echo 'verbose'; fi",
			args:    map[string]interface{}{"verbose": true},
			argDefs: []types.MCPArgument{{Name: "verbose", Type: types.MCPArgTypeBoolean}},
			want:    "if true; then echo 'verbose'; fi",
			wantErr: false,
		},
		{
			name:    "number value",
			input:   "sleep @{duration}",
			args:    map[string]interface{}{"duration": 3.5},
			argDefs: []types.MCPArgument{{Name: "duration", Type: types.MCPArgTypeNumber}},
			want:    "sleep 3.5",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SubstituteArguments(tt.input, tt.args, tt.argDefs)
			if (err != nil) != tt.wantErr {
				t.Errorf("SubstituteArguments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SubstituteArguments() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: `"hello"`,
		},
		{
			name:     "with double quotes",
			input:    `say "hello"`,
			expected: `"say \"hello\""`,
		},
		{
			name:     "with backslash",
			input:    `path\to\file`,
			expected: `"path\\to\\file"`,
		},
		{
			name:     "with dollar sign",
			input:    "$HOME",
			expected: `"\$HOME"`,
		},
		{
			name:     "with backtick",
			input:    "`command`",
			expected: "\"\\`command\\`\"",
		},
		{
			name:     "complex string",
			input:    `It's "great" & costs $100!`,
			expected: `"It's \"great\" & costs \$100!"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellQuote(tt.input)
			if got != tt.expected {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSubstituteProcessConfig(t *testing.T) {
	tests := []struct {
		name     string
		proc     *types.ProcessConfig
		args     map[string]interface{}
		wantCmd  string
		wantArgs []string
		wantErr  bool
	}{
		{
			name: "substitute command only",
			proc: &types.ProcessConfig{
				Name:    "test",
				Command: "grep @{pattern} file.txt",
				MCP: &types.MCPProcessConfig{
					Type: types.MCPProcessTypeTool,
					Arguments: []types.MCPArgument{
						{Name: "pattern", Type: types.MCPArgTypeString},
					},
				},
			},
			args:    map[string]interface{}{"pattern": "error"},
			wantCmd: `grep "error" file.txt`,
			wantErr: false,
		},
		{
			name: "substitute args only",
			proc: &types.ProcessConfig{
				Name: "test",
				Args: []string{"grep", "@{pattern}", "file.txt"},
				MCP: &types.MCPProcessConfig{
					Type: types.MCPProcessTypeTool,
					Arguments: []types.MCPArgument{
						{Name: "pattern", Type: types.MCPArgTypeString},
					},
				},
			},
			args:     map[string]interface{}{"pattern": "error"},
			wantCmd:  "",
			wantArgs: []string{`grep`, `"error"`, `file.txt`},
			wantErr:  false,
		},
		{
			name: "substitute both command and args",
			proc: &types.ProcessConfig{
				Name:    "test",
				Command: "@{cmd}",
				Args:    []string{"@{arg1}", "@{arg2}"},
				MCP: &types.MCPProcessConfig{
					Type: types.MCPProcessTypeTool,
					Arguments: []types.MCPArgument{
						{Name: "cmd", Type: types.MCPArgTypeString},
						{Name: "arg1", Type: types.MCPArgTypeString},
						{Name: "arg2", Type: types.MCPArgTypeInteger},
					},
				},
			},
			args:     map[string]interface{}{"cmd": "echo", "arg1": "hello", "arg2": 42},
			wantCmd:  `"echo"`,
			wantArgs: []string{`"hello"`, `42`},
			wantErr:  false,
		},
		{
			name: "non-MCP process returns unchanged",
			proc: &types.ProcessConfig{
				Name:    "test",
				Command: "echo hello",
			},
			args:    map[string]interface{}{},
			wantCmd: "echo hello",
			wantErr: false,
		},
		{
			name: "resource process returns unchanged",
			proc: &types.ProcessConfig{
				Name:    "test",
				Command: "cat file.txt",
				MCP: &types.MCPProcessConfig{
					Type: types.MCPProcessTypeResource,
				},
			},
			args:    map[string]interface{}{},
			wantCmd: "cat file.txt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SubstituteProcessConfig(tt.proc, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("SubstituteProcessConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantCmd != "" && got.Command != tt.wantCmd {
				t.Errorf("SubstituteProcessConfig() Command = %q, want %q", got.Command, tt.wantCmd)
			}
			if tt.wantArgs != nil {
				if len(got.Args) != len(tt.wantArgs) {
					t.Errorf("SubstituteProcessConfig() Args length = %d, want %d", len(got.Args), len(tt.wantArgs))
					return
				}
				for i := range got.Args {
					if got.Args[i] != tt.wantArgs[i] {
						t.Errorf("SubstituteProcessConfig() Args[%d] = %q, want %q", i, got.Args[i], tt.wantArgs[i])
					}
				}
			}
		})
	}
}
