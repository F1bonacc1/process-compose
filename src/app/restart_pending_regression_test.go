package app

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/types"
)

// readinessGate is an HTTP endpoint a process can be probed against, which the
// test opens when it wants the dependency to become ready. Gating on an
// explicit signal instead of a timer keeps these tests deterministic on a
// loaded machine.
type readinessGate struct {
	server *httptest.Server
	open   atomic.Bool
}

func newReadinessGate(t *testing.T) *readinessGate {
	t.Helper()
	gate := &readinessGate{}
	gate.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if gate.open.Load() {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(gate.server.Close)
	return gate
}

func (g *readinessGate) probe(t *testing.T) *health.Probe {
	t.Helper()
	addr, ok := g.server.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener address type %T", g.server.Listener.Addr())
	}
	return &health.Probe{
		HttpGet: &health.HttpProbe{
			Host: "127.0.0.1",
			Path: "/ready",
			Port: strconv.Itoa(addr.Port),
		},
		PeriodSeconds:    1,
		SuccessThreshold: 1,
		// Never let the failing probe be considered fatal - the dependency has
		// to stay alive and simply not be ready until the gate opens.
		FailureThreshold: 1 << 20,
	}
}

func (g *readinessGate) release() {
	g.open.Store(true)
}

// Regression test for https://github.com/F1bonacc1/process-compose/issues/530
//
// Restarting a process while it is still Pending (blocked in waitIfNeeded on a
// dependency) must not wedge it in Terminating, must not leave the previous,
// still-blocked instance alive, and must never produce two concurrent
// incarnations of the same process.
func TestRestartWhilePendingOnDependency(t *testing.T) {
	shell := command.DefaultShellConfig()
	const (
		procA = "a"
		procB = "b"
	)
	marker := filepath.Join(t.TempDir(), "b-started")
	gate := newReadinessGate(t)

	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes: map[string]types.ProcessConfig{
				procA: {
					Name:           procA,
					ReplicaName:    procA,
					Executable:     shell.ShellCommand,
					Args:           []string{shell.ShellArgument, getSleepCommand(120.0)},
					ReadinessProbe: gate.probe(t),
				},
				procB: {
					Name:        procB,
					ReplicaName: procB,
					Executable:  shell.ShellCommand,
					// Append on every start so concurrent incarnations are countable.
					Args: []string{shell.ShellArgument, fmt.Sprintf("echo x >> %s && ", marker) + getSleepCommand(120.0)},
					DependsOn: map[string]types.ProcessDependency{
						procA: {Condition: types.ProcessConditionHealthy},
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

	waitForProcessLaunched(t, runner, procB, 10*time.Second)
	waitForProcessState(t, runner, procB, types.ProcessStatePending, 10*time.Second)

	// Restart b while it is still waiting for a to become ready.
	if err := runner.RestartProcess(procB); err != nil {
		t.Fatalf("restart b: %v", err)
	}
	// The replacement must be waiting again, not wedged in Terminating.
	waitForProcessState(t, runner, procB, types.ProcessStatePending, 10*time.Second)
	if t.Failed() {
		return
	}

	gate.release()
	waitForProcessState(t, runner, procB, types.ProcessStateRunning, 30*time.Second)
	if t.Failed() {
		return
	}

	if proc := runner.getRunningProcess(procB); proc == nil {
		t.Errorf("b is not tracked in runningProcesses after dependency became ready (issue #530)")
	}
	// The marker is written by b's shell, which is only spawned once the state
	// reads Running - so wait for the write rather than assuming it has already
	// landed. What matters is that the count settles at one and stays there.
	if !waitForLines(marker, 1, 10*time.Second) {
		t.Errorf("b was started %d times, want exactly 1 (issue #530)", countLines(t, marker))
	}

	// A subsequent start must be rejected, not silently duplicate the process.
	if err := runner.StartProcess(procB); err == nil {
		t.Errorf("StartProcess(b) succeeded while b is running - duplicate spawned (issue #530)")
	}
	time.Sleep(500 * time.Millisecond)
	if n := countLines(t, marker); n != 1 {
		t.Errorf("after StartProcess b ran %d times, want exactly 1 (issue #530)", n)
	}
}

// A stop that lands while a restart is in flight must win. Between tearing the
// old incarnation down and launching its replacement the process is registered
// nowhere, so the stop finds nothing to stop and used to be silently overtaken
// by the relaunch - the process came back up, and the caller was told either
// that the stop had succeeded or that the process was not running.
//
// The window is sub-millisecond on Linux, which is why this test holds it open
// with a restart backoff instead of racing for it. On Windows it is wide enough
// to be hit by accident, by a watch-triggered restart of a process the user
// stops at the wrong moment.
func TestStopDuringRestartCancelsTheRestart(t *testing.T) {
	shell := command.DefaultShellConfig()
	const (
		proc = "a"
		// A second process keeps the project alive while a is down, so that the
		// runner is still around to be asked about a's state.
		keepAlive = "keepalive"
	)
	marker := filepath.Join(t.TempDir(), "a-started")

	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes: map[string]types.ProcessConfig{
				proc: {
					Name:        proc,
					ReplicaName: proc,
					Executable:  shell.ShellCommand,
					// Append on every start, so a relaunch is countable even if
					// it is stopped again right after.
					Args: []string{shell.ShellArgument, fmt.Sprintf("echo x >> %s && ", marker) + getSleepCommand(120.0)},
					// Widens the gap between the teardown and the relaunch to
					// the whole backoff, which is where the stop has to land.
					RestartPolicy: types.RestartPolicyConfig{BackoffSeconds: 2},
				},
				keepAlive: {
					Name:        keepAlive,
					ReplicaName: keepAlive,
					Executable:  shell.ShellCommand,
					Args:        []string{shell.ShellArgument, getSleepCommand(120.0)},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	go func() { _ = runner.Run() }()
	t.Cleanup(func() { _ = runner.ShutDownProject() })

	waitForProcessState(t, runner, proc, types.ProcessStateRunning, 30*time.Second)
	if t.Failed() {
		return
	}
	if !waitForLines(marker, 1, 10*time.Second) {
		t.Fatalf("a started %d times before the restart, want exactly 1", countLines(t, marker))
	}

	restarted := make(chan error, 1)
	go func() { restarted <- runner.RestartProcess(proc) }()

	// The restart is in its backoff once the old incarnation is gone. Stopping
	// here means stopping a process that is, for the moment, running nowhere.
	if !waitForProcessDeregistered(runner, proc, 30*time.Second) {
		t.Fatal("the restarted process never left the running set")
	}
	if err := runner.StopProcess(proc); err != nil {
		t.Errorf("StopProcess() during a restart error = %v, want nil", err)
	}

	select {
	case err := <-restarted:
		if err != nil {
			t.Errorf("RestartProcess() error = %v, want nil", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the restart never returned")
	}

	// The stop is the later intent, so the replacement must never have started.
	time.Sleep(time.Second)
	if n := countLines(t, marker); n != 1 {
		t.Errorf("a ran %d times, want exactly 1 - the stop was undone by the restart", n)
	}
	if p := runner.getRunningProcess(proc); p != nil {
		t.Errorf("a is running again after being stopped mid-restart")
	}
	if state, err := runner.GetProcessState(proc); err == nil && state.IsRunning {
		t.Errorf("a reports IsRunning after being stopped mid-restart, status %s", state.Status)
	}
}

// waitForProcessDeregistered reports whether the process left the running
// processes registry within timeout.
func waitForProcessDeregistered(runner *ProjectRunner, name string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if runner.getRunningProcess(name) == nil {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// waitForLines polls a marker file until it holds exactly want lines. The
// marker is appended by the process's own shell, which is spawned after the
// runner reports the process as running - on a slow spawn (Windows `cmd /C`,
// or a loaded machine) reading it right away sees nothing.
func waitForLines(path string, want int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil && strings.Count(string(data), "\n") == want {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return strings.Count(string(data), "\n")
}

// The dependency wait must be abortable for every condition, not just the
// log-ready one, so that stopping a Pending process releases it instead of
// leaving its goroutine parked on a dependency that resolves much later.
func TestStopWhilePendingReleasesTheProcess(t *testing.T) {
	shell := command.DefaultShellConfig()
	const (
		procA = "a"
		procB = "b"
	)
	tests := []struct {
		name string
		a    types.ProcessConfig
		dep  types.ProcessDependency
	}{
		{
			name: "process_completed",
			a: types.ProcessConfig{
				Args: []string{shell.ShellArgument, getSleepCommand(120.0)},
			},
			dep: types.ProcessDependency{Condition: types.ProcessConditionCompleted},
		},
		{
			name: "process_completed_successfully",
			a: types.ProcessConfig{
				Args: []string{shell.ShellArgument, getSleepCommand(120.0)},
			},
			dep: types.ProcessDependency{Condition: types.ProcessConditionCompletedSuccessfully},
		},
		{
			name: "process_log_ready",
			a: types.ProcessConfig{
				Args:         []string{shell.ShellArgument, getSleepCommand(120.0)},
				ReadyLogLine: "never-printed",
			},
			dep: types.ProcessDependency{Condition: types.ProcessConditionLogReady},
		},
		{
			name: "process_healthy",
			a: types.ProcessConfig{
				Args: []string{shell.ShellArgument, getSleepCommand(120.0)},
				ReadinessProbe: &health.Probe{
					Exec:             &health.ExecProbe{Command: "cat /unexisting"},
					FailureThreshold: 100,
					PeriodSeconds:    1,
					InitialDelay:     1,
				},
			},
			dep: types.ProcessDependency{Condition: types.ProcessConditionHealthy},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			confA := tc.a
			confA.Name = procA
			confA.ReplicaName = procA
			confA.Executable = shell.ShellCommand
			runner, err := NewProjectRunner(&ProjectOpts{
				project: &types.Project{
					ShellConfig: shell,
					Processes: map[string]types.ProcessConfig{
						procA: confA,
						procB: {
							Name:        procB,
							ReplicaName: procB,
							Executable:  shell.ShellCommand,
							Args:        []string{shell.ShellArgument, getSleepCommand(120.0)},
							DependsOn:   map[string]types.ProcessDependency{procA: tc.dep},
						},
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}

			go func() { _ = runner.Run() }()
			t.Cleanup(func() { _ = runner.ShutDownProject() })

			waitForProcessLaunched(t, runner, procB, 10*time.Second)
			waitForProcessState(t, runner, procB, types.ProcessStatePending, 10*time.Second)

			if err := runner.StopProcess(procB); err != nil {
				t.Fatalf("stop b: %v", err)
			}
			// The waiting goroutine must exit promptly - and deregister the
			// process - instead of staying parked until a resolves.
			if !waitForProcessDeregistered(runner, procB, 10*time.Second) {
				t.Fatal("stopped b is still registered - its goroutine is parked waiting for its dependency (issue #530)")
			}
			// And it must be startable again.
			if err := runner.StartProcess(procB); err != nil {
				t.Errorf("start b after stop: %v", err)
			}
		})
	}
}

// Same root cause, harsher outcome: with process_completed the stale waiter's
// exit drives the running-process count to 0 and terminates the whole project
// while the live incarnation of b is still starting, orphaning it.
func TestRestartWhilePendingPrematureProjectShutdown(t *testing.T) {
	shell := command.DefaultShellConfig()
	const (
		procA = "a"
		procB = "b"
	)

	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes: map[string]types.ProcessConfig{
				procA: {
					Name:        procA,
					ReplicaName: procA,
					Executable:  shell.ShellCommand,
					Args:        []string{shell.ShellArgument, getSleepCommand(120.0)},
				},
				procB: {
					Name:        procB,
					ReplicaName: procB,
					Executable:  shell.ShellCommand,
					Args:        []string{shell.ShellArgument, getSleepCommand(120.0)},
					DependsOn: map[string]types.ProcessDependency{
						procA: {Condition: types.ProcessConditionCompleted},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	runDone := make(chan struct{})
	go func() { _ = runner.Run(); close(runDone) }()
	t.Cleanup(func() { _ = runner.ShutDownProject() })

	waitForProcessLaunched(t, runner, procB, 10*time.Second)
	waitForProcessState(t, runner, procB, types.ProcessStatePending, 10*time.Second)

	if err := runner.RestartProcess(procB); err != nil {
		t.Fatalf("restart b: %v", err)
	}

	// Completing the dependency releases b. Stopping a is the deterministic way
	// to do that - no reliance on a process finishing within some interval.
	if err := runner.StopProcess(procA); err != nil {
		t.Fatalf("stop a: %v", err)
	}

	waitForProcessState(t, runner, procB, types.ProcessStateRunning, 30*time.Second)
	select {
	case <-runDone:
		t.Errorf("project terminated while b should still be running (issue #530)")
	default:
	}
	if proc := runner.getRunningProcess(procB); proc == nil {
		t.Errorf("b not tracked in runningProcesses")
	}
}
