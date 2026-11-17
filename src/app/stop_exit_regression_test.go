package app

import (
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
)

// Regression test for https://github.com/F1bonacc1/process-compose/issues/589
//
// A process can exit on the stop signal before the shutdown timeout starts
// waiting for it. The wait must still see that exit, instead of running out the
// timeout and then failing to SIGKILL the already-reaped PID.
func TestStopExitBeforeShutdownWait(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on Unix signals")
	}
	const name = "worker"
	shell := command.DefaultShellConfig()

	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes: map[string]types.ProcessConfig{
				name: {
					Name:        name,
					ReplicaName: name,
					Executable:  shell.ShellCommand,
					Args:        []string{shell.ShellArgument, getSleepCommand(60.0)},
					ShutDownParams: types.ShutDownParams{
						Signal:          int(syscall.SIGTERM),
						ShutDownTimeout: 1,
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

	waitForProcessState(t, runner, name, types.ProcessStateRunning, 10*time.Second)
	proc := runner.getRunningProcess(name)
	if proc == nil {
		t.Fatalf("process %s is not running", name)
	}

	// Replay the losing interleaving: the stop signal is sent and the process
	// exits and is reaped before the shutdown wait is set up.
	proc.setState(types.ProcessStateTerminating)
	if err := proc.command.Stop(int(syscall.SIGTERM), false); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}
	select {
	case <-proc.procDoneChan:
	case <-time.After(10 * time.Second):
		t.Fatalf("process %s did not exit on SIGTERM", name)
	}

	start := time.Now()
	if err := proc.forceKillOnTimeout(); err != nil {
		t.Errorf("forceKillOnTimeout after exit: %v", err)
	}
	if elapsed := time.Since(start); elapsed >= time.Second {
		t.Errorf("forceKillOnTimeout waited %v for a process that had already exited", elapsed)
	}
}
