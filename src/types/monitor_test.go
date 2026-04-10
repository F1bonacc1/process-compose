package types

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMonitorForMarshalYAML(t *testing.T) {
	tests := []struct {
		monitor  MonitorFor
		expected string
	}{
		{MonitorForNone, "none"},
		{MonitorForActivity, "activity"},
		{MonitorForSilence, "silence"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got, err := tt.monitor.MarshalYAML()
			if err != nil {
				t.Fatalf("MarshalYAML() error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("MarshalYAML() = %q, want %q", got, tt.expected)
			}

			// Round-trip: marshal to YAML, then unmarshal back
			data, err := yaml.Marshal(tt.monitor)
			if err != nil {
				t.Fatalf("yaml.Marshal() error: %v", err)
			}
			var roundTrip MonitorFor
			if err := yaml.Unmarshal(data, &roundTrip); err != nil {
				t.Fatalf("yaml.Unmarshal() error: %v", err)
			}
			if roundTrip != tt.monitor {
				t.Errorf("round-trip got %v, want %v", roundTrip, tt.monitor)
			}
		})
	}
}

func TestMonitorForUnmarshalYAML(t *testing.T) {
	tests := []struct {
		input    string
		expected MonitorFor
		wantErr  bool
	}{
		{"none", MonitorForNone, false},
		{"activity", MonitorForActivity, false},
		{"silence", MonitorForSilence, false},
		{"", MonitorForNone, false},
		{"invalid", MonitorForNone, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var got MonitorFor
			err := yaml.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("UnmarshalYAML(%q) error: %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("UnmarshalYAML(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMonitorForDefaultIsNone(t *testing.T) {
	var m MonitorFor
	if m != MonitorForNone {
		t.Errorf("zero value of MonitorFor = %v, want MonitorForNone", m)
	}
}

func TestProcessConfigMonitorFor(t *testing.T) {
	yamlData := `
command: "echo hello"
monitor_for: silence
`
	var proc ProcessConfig
	if err := yaml.Unmarshal([]byte(yamlData), &proc); err != nil {
		t.Fatalf("failed to unmarshal ProcessConfig: %v", err)
	}
	if proc.MonitorFor != MonitorForSilence {
		t.Errorf("MonitorFor = %v, want MonitorForSilence", proc.MonitorFor)
	}
}

func TestProcessConfigMonitorForOmitted(t *testing.T) {
	yamlData := `command: "echo hello"`
	var proc ProcessConfig
	if err := yaml.Unmarshal([]byte(yamlData), &proc); err != nil {
		t.Fatalf("failed to unmarshal ProcessConfig: %v", err)
	}
	if proc.MonitorFor != MonitorForNone {
		t.Errorf("MonitorFor = %v, want MonitorForNone when omitted", proc.MonitorFor)
	}
}
