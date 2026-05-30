package app

import (
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
)

func TestUpdateProcessReadyLogLineHealthyDependentFailsWithoutHanging(t *testing.T) {
	proc1 := "proc1"
	proc2 := "proc2"
	shell := command.DefaultShellConfig()
	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes: map[string]types.ProcessConfig{
				proc1: {
					Name:         proc1,
					ReplicaName:  proc1,
					Executable:   shell.ShellCommand,
					Args:         []string{shell.ShellArgument, "echo old-ready"},
					ReadyLogLine: "old-ready",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := runner.Run(); err != nil {
		t.Fatalf("initial run failed: %v", err)
	}
	waitForProcessState(t, runner, proc1, types.ProcessStateCompleted, 2*time.Second)
	if t.Failed() {
		return
	}

	err = runner.UpdateProcess(&types.ProcessConfig{
		Name:         proc1,
		ReplicaName:  proc1,
		Executable:   shell.ShellCommand,
		Args:         []string{shell.ShellArgument, "echo new-ready && " + getSleepCommand(2.0)},
		ReadyLogLine: "new-ready",
	})
	if err != nil {
		t.Fatalf("update proc1: %v", err)
	}
	runningProc := runner.getRunningProcess(proc1)
	if runningProc == nil {
		t.Fatalf("expected %s to be running after update", proc1)
	}
	if proc := runner.getDoneOrRunningProcess(proc1); proc != runningProc {
		t.Fatalf("expected running %s process, got stale done process", proc1)
	}
	t.Cleanup(func() {
		if proc := runner.getRunningProcess(proc1); proc != nil {
			_ = runner.StopProcess(proc1)
		}
	})

	runner.addProcessAndRun(types.ProcessConfig{
		Name:        proc2,
		ReplicaName: proc2,
		Executable:  shell.ShellCommand,
		Args:        []string{shell.ShellArgument, getSleepCommand(1.0)},
		DependsOn: map[string]types.ProcessDependency{
			proc1: {Condition: types.ProcessConditionHealthy},
		},
		RestartPolicy: types.RestartPolicyConfig{ExitOnSkipped: true},
	})

	waitForProcessState(t, runner, proc2, types.ProcessStateSkipped, 1500*time.Millisecond)
}
