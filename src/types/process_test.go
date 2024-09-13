package types

import (
	"github.com/f1bonacc1/process-compose/src/health"
	"testing"
)

func TestCompareProcessConfigs(t *testing.T) {
	tests := []struct {
		name     string
		p        *ProcessConfig
		another  *ProcessConfig
		expected bool
	}{
		{
			name: "equal process configs",
			p: &ProcessConfig{
				Name:        "test",
				Command:     "cmd",
				LogLocation: "log",
			},
			another: &ProcessConfig{
				Name:        "test",
				Command:     "cmd",
				LogLocation: "log",
			},
			expected: true,
		},
		{
			name: "inequal process configs (simple fields)",
			p: &ProcessConfig{
				Name:        "test",
				Command:     "cmd",
				LogLocation: "log",
			},
			another: &ProcessConfig{
				Name:        "test2",
				Command:     "cmd",
				LogLocation: "log",
			},
			expected: false,
		},
		{
			name: "inequal process configs (complex fields)",
			p: &ProcessConfig{
				Name:        "test",
				Command:     "cmd",
				LogLocation: "log",
				LoggerConfig: &LoggerConfig{
					TimestampFormat: "format",
				},
			},
			another: &ProcessConfig{
				Name:        "test",
				Command:     "cmd",
				LogLocation: "log",
				LoggerConfig: &LoggerConfig{
					TimestampFormat: "format2",
				},
			},
			expected: false,
		},
		{
			name: "equal process configs with nil fields",
			p: &ProcessConfig{
				Name:         "test",
				Command:      "cmd",
				LogLocation:  "log",
				LoggerConfig: nil,
			},
			another: &ProcessConfig{
				Name:         "test",
				Command:      "cmd",
				LogLocation:  "log",
				LoggerConfig: nil,
			},
			expected: true,
		},
		{
			name: "inequal process configs with one nil and one non-nil field",
			p: &ProcessConfig{
				Name:         "test",
				Command:      "cmd",
				LogLocation:  "log",
				LoggerConfig: nil,
			},
			another: &ProcessConfig{
				Name:        "test",
				Command:     "cmd",
				LogLocation: "log",
				LoggerConfig: &LoggerConfig{
					TimestampFormat: "format",
				},
			},
			expected: false,
		},
		{
			name: "inequal process configs with probes",
			p: &ProcessConfig{
				Name:        "test",
				Command:     "cmd",
				LogLocation: "log",
				ReadinessProbe: &health.Probe{
					Exec: &health.ExecProbe{
						Command: "echo 1",
					},
				},
			},
			another: &ProcessConfig{
				Name:        "test",
				Command:     "cmd",
				LogLocation: "log",
				ReadinessProbe: &health.Probe{
					Exec: &health.ExecProbe{
						Command: "echo 2",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Compare(tt.another); got != tt.expected {
				t.Errorf("Compare() = %v, want %v", got, tt.expected)
			}
		})
	}
}
