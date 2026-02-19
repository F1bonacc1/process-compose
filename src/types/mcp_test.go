package types

import (
	"testing"
	"time"
)

func TestMCPServerConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *MCPServerConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid stdio config",
			config: &MCPServerConfig{
				Transport: "stdio",
			},
			wantErr: false,
		},
		{
			name: "valid sse config",
			config: &MCPServerConfig{
				Host:      "localhost",
				Port:      3000,
				Transport: "sse",
			},
			wantErr: false,
		},
		{
			name: "valid config with default transport",
			config: &MCPServerConfig{
				Host: "localhost",
				Port: 3000,
				// Transport defaults to "sse"
			},
			wantErr: false,
		},
		{
			name: "invalid transport",
			config: &MCPServerConfig{
				Transport: "invalid",
			},
			wantErr: true,
		},
		{
			name: "missing host with default transport",
			config: &MCPServerConfig{
				Port: 3000,
				// Transport defaults to "sse"
			},
			wantErr: true,
		},
		{
			name: "missing port with default transport",
			config: &MCPServerConfig{
				Host: "localhost",
				// Transport defaults to "sse"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MCPServerConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMCPServerConfigIsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		config  *MCPServerConfig
		enabled bool
	}{
		{
			name:    "nil config",
			config:  nil,
			enabled: false,
		},
		{
			name: "empty transport no host/port",
			config: &MCPServerConfig{
				Transport: "",
			},
			enabled: false,
		},
		{
			name: "sse transport without host/port",
			config: &MCPServerConfig{
				Transport: "sse",
			},
			enabled: true,
		},
		{
			name: "host set with default transport",
			config: &MCPServerConfig{
				Host: "localhost",
				// Transport defaults to "sse"
			},
			enabled: true,
		},
		{
			name: "port set with default transport",
			config: &MCPServerConfig{
				Port: 3000,
				// Transport defaults to "sse"
			},
			enabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsEnabled(); got != tt.enabled {
				t.Errorf("MCPServerConfig.IsEnabled() = %v, want %v", got, tt.enabled)
			}
		})
	}
}

func TestMCPProcessConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *MCPProcessConfig
		processName string
		command     string
		args        []string
		wantErr     bool
	}{
		{
			name:        "nil config",
			config:      nil,
			processName: "test",
			command:     "echo test",
			wantErr:     false,
		},
		{
			name: "valid tool",
			config: &MCPProcessConfig{
				Type: MCPProcessTypeTool,
				Arguments: []MCPArgument{
					{Name: "pattern", Type: MCPArgTypeString},
				},
			},
			processName: "test-tool",
			command:     "grep @{pattern} file.txt",
			wantErr:     false,
		},
		{
			name: "valid resource",
			config: &MCPProcessConfig{
				Type: MCPProcessTypeResource,
			},
			processName: "test-resource",
			command:     "cat file.txt",
			wantErr:     false,
		},
		{
			name: "invalid type",
			config: &MCPProcessConfig{
				Type: "invalid",
			},
			processName: "test",
			command:     "echo test",
			wantErr:     true,
		},
		{
			name: "resource with arguments",
			config: &MCPProcessConfig{
				Type: MCPProcessTypeResource,
				Arguments: []MCPArgument{
					{Name: "arg1", Type: MCPArgTypeString},
				},
			},
			processName: "test",
			command:     "echo test",
			wantErr:     true,
		},
		{
			name: "invalid argument type",
			config: &MCPProcessConfig{
				Type: MCPProcessTypeTool,
				Arguments: []MCPArgument{
					{Name: "arg1", Type: "invalid"},
				},
			},
			processName: "test",
			command:     "echo @{arg1}",
			wantErr:     true,
		},
		{
			name: "argument without name",
			config: &MCPProcessConfig{
				Type: MCPProcessTypeTool,
				Arguments: []MCPArgument{
					{Type: MCPArgTypeString, Description: "No name"},
				},
			},
			processName: "test",
			command:     "echo test",
			wantErr:     true,
		},
		{
			name: "undefined argument in command",
			config: &MCPProcessConfig{
				Type: MCPProcessTypeTool,
				Arguments: []MCPArgument{
					{Name: "pattern", Type: MCPArgTypeString},
				},
			},
			processName: "test",
			command:     "grep @{undefined} file.txt",
			wantErr:     true,
		},
		{
			name: "undefined argument in args",
			config: &MCPProcessConfig{
				Type: MCPProcessTypeTool,
				Arguments: []MCPArgument{
					{Name: "pattern", Type: MCPArgTypeString},
				},
			},
			processName: "test",
			command:     "echo test",
			args:        []string{"@{undefined}"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.processName, tt.command, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("MCPProcessConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractArgReferences(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []ArgReference
		wantErr bool
	}{
		{
			name:    "no references",
			command: "echo hello",
			want:    []ArgReference{},
			wantErr: false,
		},
		{
			name:    "single reference",
			command: "grep @{pattern} file.txt",
			want: []ArgReference{
				{Name: "pattern"},
			},
			wantErr: false,
		},
		{
			name:    "multiple references",
			command: "grep @{pattern} @{filename}",
			want: []ArgReference{
				{Name: "pattern"},
				{Name: "filename"},
			},
			wantErr: false,
		},
		{
			name:    "with default value",
			command: "head -n @{limit:100}",
			want: []ArgReference{
				{Name: "limit", HasDefault: true, DefaultValue: "100"},
			},
			wantErr: false,
		},
		{
			name:    "escaped reference",
			command: `echo "Use \@{pattern} syntax"`,
			want:    []ArgReference{}, // Escaped patterns are not extracted
			wantErr: false,
		},
		{
			name:    "mixed escaped and regular",
			command: `echo \@{literal} and @{arg}`,
			want: []ArgReference{
				{Name: "arg"},
			},
			wantErr: false,
		},
		{
			name:    "duplicates deduplicated",
			command: "echo @{arg} and @{arg}",
			want: []ArgReference{
				{Name: "arg"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractArgReferences(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractArgReferences() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ExtractArgReferences() returned %d refs, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("ExtractArgReferences()[%d].Name = %v, want %v", i, got[i].Name, tt.want[i].Name)
				}
				if got[i].HasDefault != tt.want[i].HasDefault {
					t.Errorf("ExtractArgReferences()[%d].HasDefault = %v, want %v", i, got[i].HasDefault, tt.want[i].HasDefault)
				}
				if got[i].DefaultValue != tt.want[i].DefaultValue {
					t.Errorf("ExtractArgReferences()[%d].DefaultValue = %v, want %v", i, got[i].DefaultValue, tt.want[i].DefaultValue)
				}
			}
		})
	}
}

func TestMCPServerConfigGetTimeout(t *testing.T) {
	tests := []struct {
		name    string
		config  *MCPServerConfig
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			want:    0,
			wantErr: false,
		},
		{
			name: "empty timeout",
			config: &MCPServerConfig{
				Host: "localhost",
				Port: 3000,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "valid timeout 5m",
			config: &MCPServerConfig{
				Host:    "localhost",
				Port:    3000,
				Timeout: "5m",
			},
			want:    5 * time.Minute,
			wantErr: false,
		},
		{
			name: "valid timeout 30s",
			config: &MCPServerConfig{
				Host:    "localhost",
				Port:    3000,
				Timeout: "30s",
			},
			want:    30 * time.Second,
			wantErr: false,
		},
		{
			name: "invalid timeout",
			config: &MCPServerConfig{
				Host:    "localhost",
				Port:    3000,
				Timeout: "invalid",
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.GetTimeout()
			if (err != nil) != tt.wantErr {
				t.Errorf("MCPServerConfig.GetTimeout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MCPServerConfig.GetTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMCPProcessConfigGetTimeout(t *testing.T) {
	tests := []struct {
		name    string
		config  *MCPProcessConfig
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			want:    0,
			wantErr: false,
		},
		{
			name: "empty timeout",
			config: &MCPProcessConfig{
				Type: MCPProcessTypeResource,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "valid timeout 10s",
			config: &MCPProcessConfig{
				Type:    MCPProcessTypeResource,
				Timeout: "10s",
			},
			want:    10 * time.Second,
			wantErr: false,
		},
		{
			name: "valid timeout 1h",
			config: &MCPProcessConfig{
				Type:    MCPProcessTypeTool,
				Timeout: "1h",
			},
			want:    time.Hour,
			wantErr: false,
		},
		{
			name: "invalid timeout",
			config: &MCPProcessConfig{
				Type:    MCPProcessTypeResource,
				Timeout: "not-a-duration",
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.GetTimeout()
			if (err != nil) != tt.wantErr {
				t.Errorf("MCPProcessConfig.GetTimeout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MCPProcessConfig.GetTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMCPProcessConfigValidationWithTimeout(t *testing.T) {
	tests := []struct {
		name        string
		config      *MCPProcessConfig
		processName string
		command     string
		wantErr     bool
	}{
		{
			name: "valid timeout",
			config: &MCPProcessConfig{
				Type:    MCPProcessTypeResource,
				Timeout: "30s",
			},
			processName: "test",
			command:     "echo test",
			wantErr:     false,
		},
		{
			name: "invalid timeout format",
			config: &MCPProcessConfig{
				Type:    MCPProcessTypeResource,
				Timeout: "invalid",
			},
			processName: "test",
			command:     "echo test",
			wantErr:     true,
		},
		{
			name: "empty timeout is valid",
			config: &MCPProcessConfig{
				Type: MCPProcessTypeResource,
			},
			processName: "test",
			command:     "echo test",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.processName, tt.command, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("MCPProcessConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
