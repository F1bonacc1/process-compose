package app

import (
	"fmt"
	"maps"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
)

// A project update that replaces a process while that process is restarting
// must cancel the restart. Otherwise both start an instance, and the one that
// is not in runningProcesses keeps running where nothing can stop it.
func TestProjectUpdateDuringRestart(t *testing.T) {
	shell := command.DefaultShellConfig()
	tests := []struct {
		name string
		// script must append a line to marker on every start.
		script     func(marker string) string
		shutdown   types.ShutDownParams
		unixOnly   bool
		waitWindow func(t *testing.T, runner *ProjectRunner, name string)
	}{
		{
			name: "update during restart backoff",
			script: func(marker string) string {
				return fmt.Sprintf("echo x >> %s && ", marker) + getSleepCommand(120.0)
			},
			waitWindow: func(t *testing.T, runner *ProjectRunner, name string) {
				if !waitForProcessDeregistered(runner, name, 30*time.Second) {
					t.Fatal("svc never stopped")
				}
			},
		},
		{
			name:     "update while restart waits for shutdown",
			unixOnly: true,
			script: func(marker string) string {
				return fmt.Sprintf("trap 'sleep 1; exit 0' TERM; echo x >> %s; %s & wait", marker, getSleepCommand(120.0))
			},
			shutdown: types.ShutDownParams{Signal: 15, ShutDownTimeout: 5},
			waitWindow: func(t *testing.T, runner *ProjectRunner, name string) {
				waitForProcessState(t, runner, name, types.ProcessStateTerminating, 10*time.Second)
				if t.Failed() {
					t.FailNow()
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.unixOnly && runtime.GOOS == "windows" {
				t.Skip("uses a POSIX shell trap")
			}
			const (
				proc = "svc"
				// Keeps the project running while svc is down during the restart.
				keepAlive = "keepalive"
			)
			marker := filepath.Join(t.TempDir(), "svc-started")
			svc := types.ProcessConfig{
				Name:           proc,
				ReplicaName:    proc,
				Executable:     shell.ShellCommand,
				Args:           []string{shell.ShellArgument, tc.script(marker)},
				ShutDownParams: tc.shutdown,
				RestartPolicy:  types.RestartPolicyConfig{BackoffSeconds: 2},
			}
			processes := map[string]types.ProcessConfig{
				proc: svc,
				keepAlive: {
					Name:        keepAlive,
					ReplicaName: keepAlive,
					Executable:  shell.ShellCommand,
					Args:        []string{shell.ShellArgument, getSleepCommand(120.0)},
				},
			}
			runner, err := NewProjectRunner(&ProjectOpts{
				project: &types.Project{
					ShellConfig: shell,
					Processes:   maps.Clone(processes),
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
				t.Fatalf("svc started %d times before the restart, want exactly 1", countLines(t, marker))
			}

			restarted := make(chan error, 1)
			go func() { restarted <- runner.RestartProcess(proc) }()
			tc.waitWindow(t, runner, proc)

			// Same as editing the config and calling POST /project/configuration.
			updated := maps.Clone(processes)
			changed := svc
			changed.RestartPolicy.BackoffSeconds = 3
			updated[proc] = changed
			if _, err := runner.UpdateProject(&types.Project{ShellConfig: shell, Processes: updated}); err != nil {
				t.Fatalf("UpdateProject() error = %v", err)
			}
			instance := runner.getRunningProcess(proc)
			if instance == nil {
				t.Fatal("svc is not running after the update")
			}

			select {
			case err := <-restarted:
				if err != nil {
					t.Errorf("RestartProcess() error = %v, want nil", err)
				}
			case <-time.After(30 * time.Second):
				t.Fatal("the restart never returned")
			}

			// One start before the restart and one by the update.
			if !waitForLines(marker, 2, 10*time.Second) {
				t.Fatalf("svc started %d times, want 2", countLines(t, marker))
			}
			time.Sleep(time.Second)
			if n := countLines(t, marker); n != 2 {
				t.Errorf("svc started %d times, want 2", n)
			}
			if current := runner.getRunningProcess(proc); current != instance {
				t.Error("svc started by the update is no longer in runningProcesses")
			}
		})
	}
}
