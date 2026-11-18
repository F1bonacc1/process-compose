package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
)

// Regression test for https://github.com/F1bonacc1/process-compose/issues/558
//
// A process that failed and was restarted automatically keeps the failed run's
// exit code in its state while the new run is up. That stale code must not make
// the process report as not ready.
func TestRunningProcessIsReadyAfterAutomaticRestart(t *testing.T) {
	const (
		proc1    = "proc1"
		exitCode = 7
	)
	shell := command.DefaultShellConfig()
	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes: map[string]types.ProcessConfig{
				proc1: {
					Name:        proc1,
					ReplicaName: proc1,
					Executable:  shell.ShellCommand,
					Args:        []string{shell.ShellArgument, fmt.Sprintf("%s && exit %d", getSleepCommand(1.0), exitCode)},
					RestartPolicy: types.RestartPolicyConfig{
						Restart:        types.RestartPolicyOnFailure,
						BackoffSeconds: 1,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	startRunner(t, runner)

	// Assert on the very snapshot that showed the restarted run, so the process
	// cannot move on to its next failure in between.
	var state *types.ProcessState
	if !waitFor(30*time.Second, func() bool {
		state, err = runner.GetProcessState(proc1)
		return err == nil && state.Restarts > 0 && state.Status == types.ProcessStateRunning
	}) {
		t.Fatalf("%s was never running again after a restart", proc1)
	}
	// The failed run's exit code must still be in the state, otherwise the
	// readiness check below would pass without exercising the stale code.
	if state.ExitCode != exitCode {
		t.Fatalf("%s: exit code = %d, want the failed run's %d", proc1, state.ExitCode, exitCode)
	}
	if ready, reason := state.IsReadyReason(); !ready {
		t.Errorf("%s: IsReadyReason() = false (%s), want true for a running process", proc1, reason)
	}
}
