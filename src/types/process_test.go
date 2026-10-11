package types

import (
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/health"
	"gopkg.in/yaml.v3"
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
		{
			name: "completed, signal exit code in success_exit_codes",
			p: &ProcessState{
				Status:           ProcessStateCompleted,
				HasHealthProbe:   false,
				Health:           ProcessHealthUnknown,
				ExitCode:         130,
				SuccessExitCodes: []int{130},
			},
			isReady: true,
		},
		{
			name: "completed, signal exit code not in success_exit_codes",
			p: &ProcessState{
				Status:           ProcessStateCompleted,
				HasHealthProbe:   false,
				Health:           ProcessHealthUnknown,
				ExitCode:         130,
				SuccessExitCodes: []int{143},
			},
			isReady: false,
		},
		{
			name: "completed, non-zero exit, no success_exit_codes",
			p: &ProcessState{
				Status:         ProcessStateCompleted,
				HasHealthProbe: false,
				Health:         ProcessHealthUnknown,
				ExitCode:       130,
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

func TestRestartPolicyMarshalYAML(t *testing.T) {
	tests := []struct {
		policy   RestartPolicy
		expected string
	}{
		{RestartPolicyNo, "no"},
		{RestartPolicyAlways, "always"},
		{RestartPolicyOnFailure, "on_failure"},
		{RestartPolicyExitOnFailure, "exit_on_failure"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// Verify MarshalYAML returns the expected string
			got, err := tt.policy.MarshalYAML()
			if err != nil {
				t.Fatalf("MarshalYAML() error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("MarshalYAML() = %q, want %q", got, tt.expected)
			}

			// Verify round-trip through yaml.Marshal/Unmarshal
			data, err := yaml.Marshal(tt.policy)
			if err != nil {
				t.Fatalf("yaml.Marshal() error: %v", err)
			}
			var roundTripped RestartPolicy
			if err := yaml.Unmarshal(data, &roundTripped); err != nil {
				t.Fatalf("yaml.Unmarshal() error: %v", err)
			}
			if roundTripped != tt.policy {
				t.Errorf("Round-trip failed: got %v, want %v", roundTripped, tt.policy)
			}
		})
	}
}

func TestProcessConditionMarshalYAML(t *testing.T) {
	tests := []struct {
		condition ProcessCondition
		expected  string
	}{
		{ProcessConditionCompleted, "process_completed"},
		{ProcessConditionCompletedSuccessfully, "process_completed_successfully"},
		{ProcessConditionHealthy, "process_healthy"},
		{ProcessConditionStarted, "process_started"},
		{ProcessConditionLogReady, "process_log_ready"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// Verify MarshalYAML returns the expected string
			got, err := tt.condition.MarshalYAML()
			if err != nil {
				t.Fatalf("MarshalYAML() error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("MarshalYAML() = %q, want %q", got, tt.expected)
			}

			// Verify round-trip through yaml.Marshal/Unmarshal
			data, err := yaml.Marshal(tt.condition)
			if err != nil {
				t.Fatalf("yaml.Marshal() error: %v", err)
			}
			var roundTripped ProcessCondition
			if err := yaml.Unmarshal(data, &roundTripped); err != nil {
				t.Fatalf("yaml.Unmarshal() error: %v", err)
			}
			if roundTripped != tt.condition {
				t.Errorf("Round-trip failed: got %v, want %v", roundTripped, tt.condition)
			}
		})
	}
}

func TestDisplayProcessStatus(t *testing.T) {
	now := time.Now()
	next := now.Add(time.Minute)

	tests := []struct {
		name     string
		state    ProcessState
		expected string
	}{
		{
			name: "running process",
			state: ProcessState{
				Status:    ProcessStateRunning,
				IsRunning: true,
			},
			expected: ProcessStateRunning,
		},
		{
			name: "running process with next run (should still show running)",
			state: ProcessState{
				Status:      ProcessStateRunning,
				IsRunning:   true,
				NextRunTime: &next,
			},
			expected: ProcessStateRunning,
		},
		{
			name: "completed process with next run (should show scheduled)",
			state: ProcessState{
				Status:      ProcessStateCompleted,
				IsRunning:   false,
				NextRunTime: &next,
			},
			expected: ProcessStateScheduled,
		},
		{
			name: "failed process with next run (should show scheduled)",
			state: ProcessState{
				Status:      ProcessStateCompleted,
				IsRunning:   false,
				ExitCode:    1,
				NextRunTime: &next,
			},
			expected: ProcessStateScheduled,
		},
		{
			name: "completed process success (no next run)",
			state: ProcessState{
				Status:      ProcessStateCompleted,
				IsRunning:   false,
				ExitCode:    0,
				NextRunTime: nil,
			},
			expected: ProcessStateCompleted,
		},
		{
			name: "completed process failed (no next run)",
			state: ProcessState{
				Status:      ProcessStateCompleted,
				IsRunning:   false,
				ExitCode:    1,
				NextRunTime: nil,
			},
			expected: "Failed",
		},
		{
			name: "completed with signal exit code in success_exit_codes (not failed)",
			state: ProcessState{
				Status:           ProcessStateCompleted,
				IsRunning:        false,
				ExitCode:         130,
				SuccessExitCodes: []int{130},
				NextRunTime:      nil,
			},
			expected: ProcessStateCompleted,
		},
		{
			name: "completed with signal exit code not in success_exit_codes (failed)",
			state: ProcessState{
				Status:           ProcessStateCompleted,
				IsRunning:        false,
				ExitCode:         130,
				SuccessExitCodes: []int{143},
				NextRunTime:      nil,
			},
			expected: "Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DisplayProcessStatus(tt.state); got != tt.expected {
				t.Errorf("DisplayProcessStatus() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsExitCodeSuccess(t *testing.T) {
	tests := []struct {
		name         string
		code         int
		successCodes []int
		want         bool
	}{
		{name: "zero is always success", code: 0, successCodes: nil, want: true},
		{name: "zero success even with a list", code: 0, successCodes: []int{130}, want: true},
		{name: "non-zero not listed is failure", code: 1, successCodes: nil, want: false},
		{name: "non-zero not in list is failure", code: 1, successCodes: []int{130}, want: false},
		{name: "SIGINT exit in list is success", code: 130, successCodes: []int{130}, want: true},
		{name: "SIGTERM exit in list is success", code: 143, successCodes: []int{0, 130, 143}, want: true},
		{name: "in-list among many", code: 130, successCodes: []int{2, 130, 143}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExitCodeSuccess(tt.code, tt.successCodes); got != tt.want {
				t.Errorf("isExitCodeSuccess(%d, %v) = %v, want %v", tt.code, tt.successCodes, got, tt.want)
			}
			cfg := &ProcessConfig{SuccessExitCodes: tt.successCodes}
			if got := cfg.IsExitCodeSuccess(tt.code); got != tt.want {
				t.Errorf("ProcessConfig.IsExitCodeSuccess(%d) = %v, want %v", tt.code, got, tt.want)
			}
			state := &ProcessState{ExitCode: tt.code, SuccessExitCodes: tt.successCodes}
			if got := state.IsExitCodeSuccess(); got != tt.want {
				t.Errorf("ProcessState.IsExitCodeSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessConfigCompareSuccessExitCodes(t *testing.T) {
	a := &ProcessConfig{Name: "p", SuccessExitCodes: []int{0, 130}}
	same := &ProcessConfig{Name: "p", SuccessExitCodes: []int{0, 130}}
	diff := &ProcessConfig{Name: "p", SuccessExitCodes: []int{0, 143}}

	if !a.Compare(same) {
		t.Errorf("Compare() = false for identical SuccessExitCodes, want true")
	}
	if a.Compare(diff) {
		t.Errorf("Compare() = true for differing SuccessExitCodes, want false")
	}
}

func TestValidateProcessConfigSuccessExitCodes(t *testing.T) {
	if err := (&ProcessConfig{Name: "p", SuccessExitCodes: []int{0, 130, 143, 255}}).ValidateProcessConfig(); err != nil {
		t.Errorf("ValidateProcessConfig() unexpected error for valid codes: %v", err)
	}
	if err := (&ProcessConfig{Name: "p", SuccessExitCodes: []int{256}}).ValidateProcessConfig(); err == nil {
		t.Errorf("ValidateProcessConfig() expected error for out-of-range code 256, got nil")
	}
	if err := (&ProcessConfig{Name: "p", SuccessExitCodes: []int{-1}}).ValidateProcessConfig(); err == nil {
		t.Errorf("ValidateProcessConfig() expected error for negative code, got nil")
	}
}

func TestValidateProcessConfigSendKeys(t *testing.T) {
	keys := ShutDownParams{SendKeys: "q"}
	if err := (&ProcessConfig{Name: "p", IsInteractive: true, ShutDownParams: keys}).ValidateProcessConfig(); err != nil {
		t.Errorf("ValidateProcessConfig() unexpected error for interactive send_keys: %v", err)
	}
	if err := (&ProcessConfig{Name: "p", IsTty: true, ShutDownParams: keys}).ValidateProcessConfig(); err != nil {
		t.Errorf("ValidateProcessConfig() unexpected error for tty send_keys: %v", err)
	}
	if err := (&ProcessConfig{Name: "p", ShutDownParams: keys}).ValidateProcessConfig(); err == nil {
		t.Errorf("ValidateProcessConfig() expected error for send_keys without is_interactive, got nil")
	}
}
