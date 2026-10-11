package types

import (
	"testing"

	"github.com/f1bonacc1/process-compose/src/health"
)

func TestLivenessProbeDoesNotGateReadiness(t *testing.T) {
	tests := []struct {
		name      string
		config    ProcessConfig
		wantReady bool
	}{
		{name: "liveness only", config: ProcessConfig{LivenessProbe: &health.Probe{}}, wantReady: true},
		{name: "readiness only", config: ProcessConfig{ReadinessProbe: &health.Probe{}}, wantReady: false},
		{name: "liveness and readiness", config: ProcessConfig{LivenessProbe: &health.Probe{}, ReadinessProbe: &health.Probe{}}, wantReady: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, status := range []string{ProcessStateRunning, ProcessStateLaunched} {
				t.Run(status, func(t *testing.T) {
					state := NewProcessState(&tt.config)
					state.Status = status
					if ready, reason := state.IsReadyReason(); ready != tt.wantReady {
						t.Fatalf("IsReadyReason() = %v (%s), want %v", ready, reason, tt.wantReady)
					}
					state.Health = ProcessHealthReady
					if ready, reason := state.IsReadyReason(); !ready {
						t.Fatalf("process with successful readiness check is not ready: %s", reason)
					}
				})
			}
		})
	}
}
