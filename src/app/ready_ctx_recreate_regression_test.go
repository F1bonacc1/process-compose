package app

import (
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
)

// Regression for https://github.com/F1bonacc1/process-compose/issues/544
//
// A late process_healthy dependent started while a still-running dependency is
// temporarily NotReady must wait for readiness to recover, instead of treating
// the historical Ready cancel as "aborted" and skipping.
func TestLateHealthyDependentWaitsThroughTransientNotReady(t *testing.T) {
	shell := command.DefaultShellConfig()
	const (
		parent = "parent"
		child  = "child"
	)
	gate := newReadinessGate(t)

	runner, err := NewProjectRunner(&ProjectOpts{
		processesToRun: []string{parent},
		project: &types.Project{
			ShellConfig: shell,
			Processes: map[string]types.ProcessConfig{
				parent: {
					Name:           parent,
					ReplicaName:    parent,
					Executable:     shell.ShellCommand,
					Args:           []string{shell.ShellArgument, getSleepCommand(120.0)},
					ReadinessProbe: gate.probe(t),
				},
				child: {
					Name:        child,
					ReplicaName: child,
					Executable:  shell.ShellCommand,
					Args:        []string{shell.ShellArgument, getSleepCommand(30.0)},
					DependsOn: map[string]types.ProcessDependency{
						parent: {Condition: types.ProcessConditionHealthy},
					},
					RestartPolicy: types.RestartPolicyConfig{ExitOnSkipped: true},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	go func() { _ = runner.Run() }()
	t.Cleanup(func() { _ = runner.ShutDownProject() })

	waitForProcessState(t, runner, parent, types.ProcessStateRunning, 10*time.Second)
	if t.Failed() {
		return
	}

	// First Ready cancel of procReadyCtx.
	gate.release()
	waitForProcessHealth(t, runner, parent, types.ProcessHealthReady, 10*time.Second)
	if t.Failed() {
		return
	}

	// Nonfatal readiness loss while parent keeps running.
	gate.block()
	waitForProcessHealth(t, runner, parent, types.ProcessHealthNotReady, 10*time.Second)
	if t.Failed() {
		return
	}

	// processesToRun marked child Disabled; clear it before the late start.
	cfg := runner.project.Processes[child]
	cfg.Disabled = false
	runner.project.Processes[child] = cfg
	if err := runner.StartProcess(child); err != nil {
		t.Fatalf("start child: %v", err)
	}

	// Child must wait (Pending), not skip as aborted.
	waitForProcessState(t, runner, child, types.ProcessStatePending, 10*time.Second)
	if t.Failed() {
		return
	}

	// Parent recovers without restart; child should then start.
	gate.release()
	waitForProcessHealth(t, runner, parent, types.ProcessHealthReady, 10*time.Second)
	if t.Failed() {
		return
	}
	waitForProcessState(t, runner, child, types.ProcessStateRunning, 15*time.Second)
	if t.Failed() {
		return
	}

	// Parent must remain Running (not synthetic exit_code from a failed wait).
	waitForProcessState(t, runner, parent, types.ProcessStateRunning, 2*time.Second)
}

func waitForProcessHealth(t *testing.T, runner *ProjectRunner, name, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		proc := runner.getRunningProcess(name)
		if proc == nil {
			proc = runner.getDoneOrRunningProcess(name)
		}
		if proc != nil {
			var got string
			proc.getStateData(func(state *types.ProcessState) {
				got = state.Health
			})
			if got == want {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s health=%s", name, want)
}
