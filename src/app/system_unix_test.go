//go:build !windows

package app

import (
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
	"syscall"
	"testing"
	"time"
)

func assertProcessStatus(t *testing.T, runner *ProjectRunner, procName string, wantStatus string) {
	t.Helper()
	state, err := runner.GetProcessState(procName)
	if err != nil {
		t.Fatalf("%s", err)
	}
	if state.Status != wantStatus {
		t.Fatalf("process %s status want %s got %s", procName, wantStatus, state.Status)
	}
}

func TestSystem_TestProcShutDownWithConfiguredTimeOut(t *testing.T) {
	ignoresSigTerm := "IgnoresSIGTERM"
	shell := command.DefaultShellConfig()
	timeout := 5

	project := &types.Project{
		Processes: map[string]types.ProcessConfig{
			ignoresSigTerm: {
				Name:        ignoresSigTerm,
				ReplicaName: ignoresSigTerm,
				Executable:  shell.ShellCommand,
				Args:        []string{shell.ShellArgument, "trap '' SIGTERM && sleep 60"},
				ShutDownParams: types.ShutDownParams{
					ShutDownTimeout: timeout,
					Signal:          int(syscall.SIGTERM),
				},
			},
		},
		ShellConfig: shell,
	}
	runner, err := NewProjectRunner(&ProjectOpts{project: project})
	if err != nil {
		t.Fatalf("%s", err)
	}
	go runner.Run()

	time.Sleep(100 * time.Millisecond)
	assertProcessStatus(t, runner, ignoresSigTerm, types.ProcessStateRunning)

	// If the test fails, cleanup after ourselves
	proc := runner.getRunningProcess(ignoresSigTerm)
	defer proc.command.Stop(int(syscall.SIGKILL), true)

	err = runner.StopProcess(ignoresSigTerm)
	if err != nil {
		t.Fatalf("%s", err)
	}

	for i := 0; i < timeout-1; i++ {
		time.Sleep(time.Second)
		assertProcessStatus(t, runner, ignoresSigTerm, types.ProcessStateTerminating)
	}

	time.Sleep(2 * time.Second)
	assertProcessStatus(t, runner, ignoresSigTerm, types.ProcessStateCompleted)
}
