package app

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
)

var waitResultNames = map[waitResult]string{
	waitOk:      "waitOk",
	waitFailed:  "waitFailed",
	waitAborted: "waitAborted",
}

// newProbedProcess returns a process with a readiness probe that is never
// started, so the test drives readiness transitions itself through
// onReadinessCheckEnd.
func newProbedProcess() *Process {
	shell := command.DefaultShellConfig()
	return NewProcess(
		withProcConf(&types.ProcessConfig{
			Name:        "parent",
			ReplicaName: "parent",
			Executable:  shell.ShellCommand,
			Args:        []string{shell.ShellArgument, getSleepCommand(120.0)},
			ReadinessProbe: &health.Probe{
				Exec:             &health.ExecProbe{Command: "true"},
				PeriodSeconds:    1,
				SuccessThreshold: 1,
				FailureThreshold: 1 << 20,
			},
		}),
		withLogger(pclog.NewNilLogger()),
		withProcState(&types.ProcessState{}),
		withProcLog(pclog.NewLogBuffer(100)),
	)
}

// startReadyWaiter waits on p the way a process_healthy dependent does and
// reports the outcome on the returned channel.
func startReadyWaiter(p *Process, abort <-chan struct{}) <-chan waitResult {
	res := make(chan waitResult, 1)
	go func() { res <- p.waitUntilReady(abort) }()
	return res
}

func expectStillWaiting(t *testing.T, res <-chan waitResult) {
	t.Helper()
	select {
	case r := <-res:
		t.Fatalf("waitUntilReady returned %s while the dependency is running but not ready, want it to keep waiting", waitResultNames[r])
	case <-time.After(300 * time.Millisecond):
	}
}

func expectWaitResult(t *testing.T, res <-chan waitResult, want waitResult) {
	t.Helper()
	select {
	case r := <-res:
		if r != want {
			t.Fatalf("waitUntilReady returned %s, want %s", waitResultNames[r], waitResultNames[want])
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("waitUntilReady did not return, want %s", waitResultNames[want])
	}
}

// Regression test for https://github.com/F1bonacc1/process-compose/issues/544
//
// A dependency that was ready once and then lost readiness through a nonfatal
// probe failure is still running and may recover. A dependent that starts
// waiting on it in that window must wait for the recovery instead of being
// told the dependency was terminated, and the dependency's exit code must be
// left alone.
func TestWaitUntilReadyAfterTransientNotReady(t *testing.T) {
	t.Run("waits for recovery", func(t *testing.T) {
		p := newProbedProcess()
		p.onReadinessCheckEnd(true, false, "", nil)
		p.onReadinessCheckEnd(false, false, "probe failed", nil)

		res := startReadyWaiter(p, make(chan struct{}))
		expectStillWaiting(t, res)

		p.onReadinessCheckEnd(true, false, "", nil)
		expectWaitResult(t, res, waitOk)
		if code := p.getExitCode(); code != 0 {
			t.Errorf("dependency exit code = %d, want 0 - it never exited", code)
		}
	})

	t.Run("fails once the dependency ends", func(t *testing.T) {
		p := newProbedProcess()
		p.onReadinessCheckEnd(true, false, "", nil)
		p.onReadinessCheckEnd(false, false, "probe failed", nil)

		res := startReadyWaiter(p, make(chan struct{}))
		expectStillWaiting(t, res)

		p.onProcessEnd(types.ProcessStateCompleted)
		expectWaitResult(t, res, waitFailed)
	})

	t.Run("aborts with the waiter", func(t *testing.T) {
		p := newProbedProcess()
		p.onReadinessCheckEnd(true, false, "", nil)
		p.onReadinessCheckEnd(false, false, "probe failed", nil)

		abort := make(chan struct{})
		res := startReadyWaiter(p, abort)
		expectStillWaiting(t, res)

		close(abort)
		expectWaitResult(t, res, waitAborted)
		if code := p.getExitCode(); code != 0 {
			t.Errorf("dependency exit code = %d, want 0 - it never exited", code)
		}
	})
}

// The #544 scenario end to end: a disabled dependent is started by hand while
// its dependency is running but temporarily not ready.
func TestLateHealthyDependentWaitsThroughTransientNotReady(t *testing.T) {
	shell := command.DefaultShellConfig()
	const (
		parent = "parent"
		child  = "child"
	)
	marker := filepath.Join(t.TempDir(), "child-started")
	gate := newReadinessGate(t)
	gate.release()

	runner, err := NewProjectRunner(&ProjectOpts{
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
					Args:        []string{shell.ShellArgument, fmt.Sprintf("echo x >> %s && ", marker) + getSleepCommand(120.0)},
					Disabled:    true,
					DependsOn: map[string]types.ProcessDependency{
						parent: {Condition: types.ProcessConditionHealthy},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	go func() { _ = runner.Run() }()
	t.Cleanup(func() { _ = runner.ShutDownProject() })

	waitForProcessHealth(t, runner, parent, types.ProcessHealthReady, 30*time.Second)
	// A nonfatal readiness loss: parent keeps running and can still recover.
	gate.open.Store(false)
	waitForProcessHealth(t, runner, parent, types.ProcessHealthNotReady, 30*time.Second)

	if err := runner.StartProcess(child); err != nil {
		t.Fatalf("start child: %v", err)
	}
	expectWaitingOnDependency(t, runner, child, 2*time.Second)
	expectExitCode(t, runner, parent, 0)

	gate.release()
	waitForProcessState(t, runner, child, types.ProcessStateRunning, 30*time.Second)
	if !waitForLines(marker, 1, 10*time.Second) {
		t.Errorf("child ran %d times, want exactly 1", countLines(t, marker))
	}
	expectExitCode(t, runner, parent, 0)
}

func waitForProcessHealth(t *testing.T, runner *ProjectRunner, name, want string, timeout time.Duration) {
	t.Helper()
	got := "unknown"
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if state, err := runner.GetProcessState(name); err == nil {
			if state.Health == want {
				return
			}
			got = state.Health
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("process %s: want health %q, got %q (after %v)", name, want, got, timeout)
}

// expectWaitingOnDependency fails as soon as the process stops waiting for its
// dependencies within d, whether it was skipped or started. Its status is no
// help here: a disabled process started by hand reports Disabled while it
// waits.
func expectWaitingOnDependency(t *testing.T, runner *ProjectRunner, name string, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		state, err := runner.GetProcessState(name)
		if err != nil {
			t.Fatal(err)
		}
		if proc := runner.getRunningProcess(name); proc == nil || proc.isDone() || state.IsRunning {
			t.Fatalf("process %s stopped waiting for its dependency: status %s, exit code %d", name, state.Status, state.ExitCode)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func expectExitCode(t *testing.T, runner *ProjectRunner, name string, want int) {
	t.Helper()
	state, err := runner.GetProcessState(name)
	if err != nil {
		t.Fatal(err)
	}
	if state.ExitCode != want {
		t.Errorf("process %s: exit code = %d, want %d", name, state.ExitCode, want)
	}
}
