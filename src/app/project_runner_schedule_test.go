package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
)

func newScheduledUpdateRunner(t *testing.T) (*ProjectRunner, types.ProcessConfig) {
	t.Helper()
	shell := command.DefaultShellConfig()
	keeper := types.ProcessConfig{Name: "keeper", ReplicaName: "keeper", Command: getSleepCommand(30)}
	keeper.AssignProcessExecutableAndArgs(shell, "")
	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{ShellConfig: shell, Processes: types.Processes{"keeper": keeper}},
		refRate: 100 * time.Millisecond,
		noWatch: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- runner.Run() }()
	t.Cleanup(func() {
		_ = runner.ShutDownProject()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("runner did not stop")
		}
	})
	waitForProcessLaunched(t, runner, "keeper", 5*time.Second)
	return runner, keeper
}

func updateScheduledProject(t *testing.T, runner *ProjectRunner, keeper types.ProcessConfig, proc *types.ProcessConfig) map[string]string {
	t.Helper()
	project := &types.Project{ShellConfig: runner.project.ShellConfig, Processes: types.Processes{"keeper": keeper}}
	if proc != nil {
		proc.AssignProcessExecutableAndArgs(project.ShellConfig, "")
		project.Processes[proc.Name] = *proc
	}
	status, err := runner.UpdateProject(project)
	if err != nil {
		t.Fatal(err)
	}
	return status
}

func scheduledRunCount(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return strings.Count(string(data), "\n")
}

func TestUpdateProjectRegistersScheduledProcess(t *testing.T) {
	runner, keeper := newScheduledUpdateRunner(t)
	shell := runner.project.ShellConfig
	proc := types.ProcessConfig{
		Name: "scheduled", ReplicaName: "scheduled", Command: "echo scheduled",
		Schedule: &types.ScheduleConfig{Interval: "1h"},
	}
	proc.AssignProcessExecutableAndArgs(shell, "")
	status, err := runner.UpdateProject(&types.Project{
		ShellConfig: shell, Processes: types.Processes{"keeper": keeper, proc.Name: proc},
	})
	if err != nil {
		t.Fatal(err)
	}
	if status[proc.Name] != types.ProcessUpdateAdded {
		t.Fatalf("update status = %v", status)
	}
	state, err := runner.GetProcessState(proc.Name)
	if err != nil {
		t.Fatal(err)
	}
	if state.NextRunTime == nil || state.Status != types.ProcessStateScheduled {
		t.Fatalf("hot-added process: status=%s next_run_time=%v; want Scheduled with a next run", state.Status, state.NextRunTime)
	}
}

func TestUpdateProjectScheduledProcessLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs")
	runner, keeper := newScheduledUpdateRunner(t)
	proc := types.ProcessConfig{
		Name: "scheduled", ReplicaName: "scheduled", Command: "echo run >> \"" + path + "\"",
		Schedule: &types.ScheduleConfig{Interval: "500ms", RunOnStart: true},
	}
	updateScheduledProject(t, runner, keeper, &proc)
	deadline := time.Now().Add(5 * time.Second)
	for scheduledRunCount(t, path) < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if n := scheduledRunCount(t, path); n < 2 {
		t.Fatalf("hot-added interval task ran %d times; want recurring runs", n)
	}

	// The command's output is written before its process goroutine finishes.
	// Wait for the run to complete before editing the pending schedule.
	deadline = time.Now().Add(5 * time.Second)
	for runner.getRunningProcess(proc.Name) != nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if runner.getRunningProcess(proc.Name) != nil {
		t.Fatal("scheduled run did not complete")
	}

	// A schedule-only edit must replace the old armed job.
	proc.Schedule = &types.ScheduleConfig{Interval: "2h"}
	if status := updateScheduledProject(t, runner, keeper, &proc); status[proc.Name] != types.ProcessUpdateUpdated {
		t.Fatalf("schedule edit status = %v", status)
	}
	sched := runner.processScheduler.Load()
	next := sched.GetNextRunTime(proc.Name)
	if next == nil || time.Until(*next) < time.Hour {
		t.Fatalf("edited schedule next run = %v", next)
	}
	oldRuns := scheduledRunCount(t, path)
	time.Sleep(700 * time.Millisecond)
	if n := scheduledRunCount(t, path); n != oldRuns {
		t.Fatalf("old interval still firing: runs changed from %d to %d", oldRuns, n)
	}

	if status := updateScheduledProject(t, runner, keeper, nil); status[proc.Name] != types.ProcessUpdateRemoved {
		t.Fatalf("remove status = %v", status)
	}
	if sched.IsScheduled(proc.Name) || sched.GetNextRunTime(proc.Name) != nil {
		t.Fatal("removed process still has a schedule")
	}

	// Re-adding the same name must be independently scheduled.
	proc.Schedule = &types.ScheduleConfig{Cron: "0 0 1 1 *", Timezone: "UTC"}
	updateScheduledProject(t, runner, keeper, &proc)
	if sched.GetNextRunTime(proc.Name) == nil {
		t.Fatal("re-added cron task has no next run")
	}

	// Removing scheduling turns it back into an ordinary one-shot process.
	proc.Schedule = nil
	updateScheduledProject(t, runner, keeper, &proc)
	if sched.IsScheduled(proc.Name) {
		t.Fatal("one-shot process kept its old schedule")
	}
}

func TestUpdateProjectDisabledSchedule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unexpected-run")
	runner, keeper := newScheduledUpdateRunner(t)
	proc := types.ProcessConfig{
		Name: "disabled", ReplicaName: "disabled", Disabled: true,
		Command:  "echo unexpected >> \"" + path + "\"",
		Schedule: &types.ScheduleConfig{Interval: "20ms", RunOnStart: true},
	}
	proc.AssignProcessExecutableAndArgs(runner.project.ShellConfig, "")
	project := &types.Project{ShellConfig: runner.project.ShellConfig, Processes: types.Processes{"keeper": keeper, proc.Name: proc}}
	if _, err := runner.UpdateProject(project); err != nil {
		t.Fatal(err)
	}
	sched := runner.processScheduler.Load()
	if !sched.IsScheduled(proc.Name) || sched.GetNextRunTime(proc.Name) != nil {
		t.Fatal("disabled schedule was not registered paused")
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("disabled task ran: stat error = %v", err)
	}
	if err := runner.StartProcess(proc.Name); err != nil {
		t.Fatal(err)
	}
	if sched.GetNextRunTime(proc.Name) == nil {
		t.Fatal("manual start did not resume the disabled schedule")
	}
}

func TestUpdateProjectRemovesRunningScheduledProcess(t *testing.T) {
	runner, keeper := newScheduledUpdateRunner(t)
	proc := types.ProcessConfig{
		Name: "scheduled", ReplicaName: "scheduled", Command: getSleepCommand(30),
		Schedule: &types.ScheduleConfig{Interval: "1h", RunOnStart: true},
	}
	proc.AssignProcessExecutableAndArgs(runner.project.ShellConfig, "")
	if _, err := runner.UpdateProject(&types.Project{ShellConfig: runner.project.ShellConfig, Processes: types.Processes{"keeper": keeper, proc.Name: proc}}); err != nil {
		t.Fatal(err)
	}
	waitForProcessLaunched(t, runner, proc.Name, 5*time.Second)
	waitForProcessState(t, runner, proc.Name, types.ProcessStateRunning, 5*time.Second)
	running := runner.getRunningProcess(proc.Name)
	if running == nil {
		t.Fatal("scheduled task did not start")
	}
	if _, err := runner.UpdateProject(&types.Project{ShellConfig: runner.project.ShellConfig, Processes: types.Processes{"keeper": keeper}}); err != nil {
		t.Fatal(err)
	}
	if !running.waitUntilTerminated(5 * time.Second) {
		t.Fatal("removed task did not terminate")
	}
	sched := runner.processScheduler.Load()
	if sched.IsScheduled(proc.Name) || runner.getRunningProcess(proc.Name) != nil {
		t.Fatal("deleted task still running or scheduled")
	}
}
