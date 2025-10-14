package app

import (
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func getTestLogger(t *testing.T) pclog.PcLogger {
	return pclog.NewNilLogger()
}

func getTestLogBuffer() *pclog.ProcessLogBuffer {
	return pclog.NewLogBuffer(100)
}

func getTestShellConfig() *command.ShellConfig {
	return command.DefaultShellConfig()
}

func TestReadinessProbeRestart(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	proc := &types.ProcessConfig{
		Name:    "test",
		Command: "sleep 10",
		RestartPolicy: types.RestartPolicyConfig{
			Restart:     types.RestartPolicyOnFailure,
			MaxRestarts: 1,
		},
		ReadinessProbe: &health.Probe{
			Exec: &health.ExecProbe{
				Command: "cat /unexisting",
			},
			FailureThreshold:    2,
			PeriodSeconds:       1,
			InitialDelay: 1,
		},
	}
	proc.AssignProcessExecutableAndArgs(getTestShellConfig(), "")

	p := NewProcess(
		withProcConf(proc),
		withLogger(getTestLogger(t)),
		withProcState(&types.ProcessState{}),
		withProcLog(getTestLogBuffer()),
		withShellConfig(*getTestShellConfig()),
	)

	go func() {
		_ = p.run()
	}()

	time.Sleep(4 * time.Second)
	assert.Equal(t, 1, p.procState.Restarts)
	_ = p.shutDown()
	p.waitForCompletion()
}

func TestLivenessProbeRestart(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	proc := &types.ProcessConfig{
		Name:    "test",
		Command: "sleep 10",
		RestartPolicy: types.RestartPolicyConfig{
			Restart:     types.RestartPolicyOnFailure,
			MaxRestarts: 1,
		},
		LivenessProbe: &health.Probe{
			Exec: &health.ExecProbe{
				Command: "cat /unexisting",
			},
			FailureThreshold:    2,
			PeriodSeconds:       1,
			InitialDelay: 1,
		},
	}
	proc.AssignProcessExecutableAndArgs(getTestShellConfig(), "")

	p := NewProcess(
		withProcConf(proc),
		withLogger(getTestLogger(t)),
		withProcState(&types.ProcessState{}),
		withProcLog(getTestLogBuffer()),
		withShellConfig(*getTestShellConfig()),
	)

	go func() {
		_ = p.run()
	}()

	time.Sleep(4 * time.Second)
	assert.Equal(t, 1, p.procState.Restarts)
	_ = p.shutDown()
	p.waitForCompletion()
}