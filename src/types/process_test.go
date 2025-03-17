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

func TestProcessStateIsReady(t *testing.T) {
	tests := []struct {
		name    string
		p       *ProcessState
		isReady bool
	}{
		{
			name: "pending, no health probe",
			p: &ProcessState{
				Status:         ProcessStatePending,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
			},
			isReady: false,
		},
		{
			name: "launching, no health probe",
			p: &ProcessState{
				Status:         ProcessStateLaunching,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
			},
			isReady: false,
		},
		{
			name: "restarting, exit ok, no health probe",
			p: &ProcessState{
				Status:         ProcessStateRestarting,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
				ExitCode:       0,
			},
			isReady: true,
		},
		{
			name: "restarting, exit failed, no health probe",
			p: &ProcessState{
				Status:         ProcessStateRestarting,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
				ExitCode:       1,
			},
			isReady: false,
		},
		{
			name: "terminating, no health probe",
			p: &ProcessState{
				Status:         ProcessStateTerminating,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
			},
			isReady: false,
		},
		{
			name: "running, no health probe",
			p: &ProcessState{
				Status:         ProcessStateRunning,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
			},
			isReady: true,
		},
		{
			name: "foreground, no health probe",
			p: &ProcessState{
				Status:         ProcessStateForeground,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
			},
			isReady: true,
		},
		{
			name: "launched, no health probe",
			p: &ProcessState{
				Status:         ProcessStateLaunched,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
			},
			isReady: true,
		},
		{
			name: "completed, exit success, no health probe",
			p: &ProcessState{
				Status:         ProcessStateCompleted,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
				ExitCode:       0,
			},
			isReady: true,
		},
		{
			name: "completed, exit failure, no health probe",
			p: &ProcessState{
				Status:         ProcessStateCompleted,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
				ExitCode:       1,
			},
			isReady: false,
		},
		{
			name: "skipped, no health probe",
			p: &ProcessState{
				Status:         ProcessStateSkipped,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
			},
			isReady: true,
		},
		{
			name: "error, no health probe",
			p: &ProcessState{
				Status:         ProcessStateError,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
			},
			isReady: false,
		},
		{
			name: "disabled, no health probe (disabled processes will only start manually)",
			p: &ProcessState{
				Status:         ProcessStateDisabled,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
			},
			isReady: true,
		},
		{
			name: "running, unhealthy",
			p: &ProcessState{
				Status:         ProcessStateRunning,
				HasHealthProbe: true,
				Health:         ProcessHealthNotReady,
			},
			isReady: false,
		},
		{
			name: "running, healthy",
			p: &ProcessState{
				Status:         ProcessStateRunning,
				HasHealthProbe: true,
				Health:         ProcessHealthReady,
			},
			isReady: true,
		},
		{
			name: "running, no probe, unhealthy",
			p: &ProcessState{
				Status: ProcessStateRunning,
				// This state probably should not be possible, but the type system does not prevent it...
				HasHealthProbe: false,
				Health:         ProcessHealthNotReady,
			},
			isReady: false,
		},
		{
			name: "garbage status and health",
			p: &ProcessState{
				// This is garbage, but again the type system allows it.
				Status:         "puppy",
				HasHealthProbe: true,
				Health:         "doggy",
			},
			isReady: false,
		},
		{
			name: "garbage health",
			p: &ProcessState{
				Status:         ProcessStateRunning,
				HasHealthProbe: true,
				Health:         "doggy",
			},
			isReady: false,
		},
		{
			name: "no health probe, garbage health",
			p: &ProcessState{
				Status:         ProcessStateRunning,
				HasHealthProbe: false,
				Health:         "doggy",
			},
			isReady: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.p.IsReady() != tt.isReady {
				t.Errorf("Expected IsReady() = %v for state %v", tt.isReady, tt.p)
			}
		})
	}
}
