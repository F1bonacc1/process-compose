package app

import (
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/types"
)

func raceProcs(shell *command.ShellConfig, initialDelay int, gen int) (types.ProcessConfig, types.ProcessConfig) {
	a := types.ProcessConfig{
		Name:        "dep-a",
		ReplicaName: "dep-a",
		Executable:  shell.ShellCommand,
		Args:        []string{shell.ShellArgument, getSleepCommand(float64(100 + gen))},
		ReadinessProbe: &health.Probe{
			Exec:             &health.ExecProbe{Command: "true"},
			FailureThreshold: 3,
			PeriodSeconds:    1,
			InitialDelay:     initialDelay,
		},
	}
	b := types.ProcessConfig{
		Name:        "dep-b",
		ReplicaName: "dep-b",
		Executable:  shell.ShellCommand,
		Args:        []string{shell.ShellArgument, getSleepCommand(float64(200 + gen))},
		DependsOn: map[string]types.ProcessDependency{
			"dep-a": {Condition: types.ProcessConditionHealthy},
		},
	}
	return a, b
}

func newRaceRunner(t *testing.T, shell *command.ShellConfig, a, b types.ProcessConfig) *ProjectRunner {
	t.Helper()
	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes:   map[string]types.ProcessConfig{a.Name: a, b.Name: b},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = runner.Run() }()
	t.Cleanup(func() {
		for _, n := range []string{"dep-a", "dep-b"} {
			if runner.getRunningProcess(n) != nil {
				_ = runner.StopProcess(n)
			}
		}
	})
	return runner
}

// TestUpdateDependentBeforeDependencyKeepsDependentRunning checks that a dependent
// re-added while its dependency is replaced waits for the new dependency instance
// instead of being skipped because the old one was terminated.
func TestUpdateDependentBeforeDependencyKeepsDependentRunning(t *testing.T) {
	shell := command.DefaultShellConfig()
	a, b := raceProcs(shell, 2, 0)
	runner := newRaceRunner(t, shell, a, b)
	waitForProcessLaunched(t, runner, "dep-a", 2*time.Second)

	// Dependent first, while the old dependency is still not ready.
	a1, b1 := raceProcs(shell, 1, 1)
	if err := runner.UpdateProcess(&b1); err != nil {
		t.Fatalf("update dep-b: %v", err)
	}
	if err := runner.UpdateProcess(&a1); err != nil {
		t.Fatalf("update dep-a: %v", err)
	}
	waitForProcessState(t, runner, "dep-a", types.ProcessStateRunning, 3*time.Second)
	waitForProcessState(t, runner, "dep-b", types.ProcessStateRunning, 5*time.Second)
}

// TestUpdateProjectBothChangedDependentNotSkipped covers the random map order of UpdateProject.
func TestUpdateProjectBothChangedDependentNotSkipped(t *testing.T) {
	shell := command.DefaultShellConfig()
	a, b := raceProcs(shell, 0, 0)
	runner := newRaceRunner(t, shell, a, b)
	waitForProcessLaunched(t, runner, "dep-a", 2*time.Second)
	waitForProcessLaunched(t, runner, "dep-b", 5*time.Second)

	for gen := 1; gen <= 6; gen++ {
		a, b = raceProcs(shell, 0, gen)
		a.Args = append([]string(nil), a.Args...)
		if _, err := runner.UpdateProject(&types.Project{
			ShellConfig: shell,
			Processes:   map[string]types.ProcessConfig{a.Name: a, b.Name: b},
		}); err != nil {
			t.Fatalf("gen %d: %v", gen, err)
		}
		waitForProcessState(t, runner, "dep-b", types.ProcessStateRunning, 5*time.Second)
		if t.Failed() {
			return
		}
	}
}
