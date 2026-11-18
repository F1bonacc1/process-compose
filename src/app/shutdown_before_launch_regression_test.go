package app

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
)

// A shutdown that lands before Run launches the processes, as a SIGTERM early
// in `process-compose up` does, must keep them from launching. ShutDownProject
// stops only the processes it finds running, so any launched after it would
// outlive process-compose.
func TestShutDownBeforeLaunchKeepsProcessesFromRunning(t *testing.T) {
	shell := command.DefaultShellConfig()
	marker := filepath.Join(t.TempDir(), "started")
	processes := map[string]types.ProcessConfig{}
	for i := range 4 {
		name := fmt.Sprintf("p%d", i)
		processes[name] = types.ProcessConfig{
			Name:        name,
			ReplicaName: name,
			Executable:  shell.ShellCommand,
			Args:        []string{shell.ShellArgument, fmt.Sprintf("echo x >> %s && ", marker) + getSleepCommand(5)},
		}
	}
	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{ShellConfig: shell, Processes: processes},
	})
	if err != nil {
		t.Fatal(err)
	}

	_ = runner.ShutDownProject()
	done := make(chan struct{})
	go func() {
		_ = runner.Run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Run() did not return after the project was shut down")
	}

	// A launched process writes its line as it starts.
	time.Sleep(500 * time.Millisecond)
	if n := countLines(t, marker); n != 0 {
		t.Errorf("%d processes launched after the shutdown, want 0", n)
	}
}
