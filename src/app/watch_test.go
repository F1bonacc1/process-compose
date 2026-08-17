package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
)

// countLinesIn returns the number of non-empty lines in a marker file,
// tolerating a file that does not exist yet.
func countLinesIn(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	count := 0
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// waitForLineCount polls a marker file until it reaches want lines.
func waitForLineCount(t *testing.T, path string, want int, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if countLinesIn(t, path) >= want {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func touch(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

// newWatchRunner builds a project with a watched long-running process.
func newWatchRunner(t *testing.T, watchDir string, mutate func(types.Processes)) *ProjectRunner {
	t.Helper()
	shell := command.DefaultShellConfig()
	procs := types.Processes{
		"api": {
			Name:        "api",
			ReplicaName: "api",
			Replicas:    1,
			Executable:  shell.ShellCommand,
			Args:        []string{shell.ShellArgument, getSleepCommand(120.0)},
			Watch: &types.WatchConfig{
				Debounce: "30ms",
				Paths:    []types.WatchPath{{Path: watchDir}},
			},
		},
	}
	if mutate != nil {
		mutate(procs)
	}
	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes:   procs,
		},
		processesToRun:  []string{},
		mainProcessArgs: []string{},
	})
	if err != nil {
		t.Fatalf("NewProjectRunner() error = %v", err)
	}
	return runner
}

func startRunner(t *testing.T, runner *ProjectRunner) {
	t.Helper()
	go func() { _ = runner.Run() }()
	t.Cleanup(func() { _ = runner.ShutDownProject() })
}

func TestWatch_RestartsProcessOnFileChange(t *testing.T) {
	watchDir := t.TempDir()
	runner := newWatchRunner(t, watchDir, nil)
	startRunner(t, runner)
	waitForProcessLaunched(t, runner, "api", 10*time.Second)
	waitForProcessState(t, runner, "api", types.ProcessStateRunning, 30*time.Second)
	if t.Failed() {
		return
	}

	before, err := runner.GetProcessState("api")
	if err != nil {
		t.Fatalf("GetProcessState() error = %v", err)
	}
	pidBefore := before.Pid

	touch(t, filepath.Join(watchDir, "main.go"), "package main")

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		state, err := runner.GetProcessState("api")
		if err == nil && state.IsRunning && state.Pid != pidBefore && state.Pid != 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("process was not restarted after a watched file changed (pid stayed %d)", pidBefore)
}

// TestWatch_CascadeRestartsDependentsInOrder is the headline behavior: a change
// re-runs the one-shot builder and then restarts its dependent, in that order.
// The order matters - a dependent restarted first would resolve against the
// builder's stale completed run and use its old output.
func TestWatch_CascadeRestartsDependentsInOrder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows: the marker commands use POSIX shell redirection")
	}
	watchDir := t.TempDir()
	markerDir := t.TempDir()
	assetsMarker := filepath.Join(markerDir, "assets.log")
	apiMarker := filepath.Join(markerDir, "api.log")

	shell := command.DefaultShellConfig()
	procs := types.Processes{
		"assets": {
			Name:        "assets",
			ReplicaName: "assets",
			Replicas:    1,
			Executable:  shell.ShellCommand,
			Args:        []string{shell.ShellArgument, fmt.Sprintf("echo built >> %s", assetsMarker)},
			RestartPolicy: types.RestartPolicyConfig{
				Restart: types.RestartPolicyNo,
			},
			Watch: &types.WatchConfig{
				Debounce: "30ms",
				Cascade:  true,
				Paths:    []types.WatchPath{{Path: watchDir}},
			},
		},
		"api": {
			Name:        "api",
			ReplicaName: "api",
			Replicas:    1,
			Executable:  shell.ShellCommand,
			Args: []string{shell.ShellArgument,
				fmt.Sprintf("echo started >> %s && %s", apiMarker, getSleepCommand(120.0))},
			DependsOn: types.DependsOnConfig{
				"assets": {Condition: types.ProcessConditionCompletedSuccessfully},
			},
		},
	}

	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes:   procs,
		},
		processesToRun:  []string{},
		mainProcessArgs: []string{},
	})
	if err != nil {
		t.Fatalf("NewProjectRunner() error = %v", err)
	}
	startRunner(t, runner)

	// Baseline: assets runs once, then api starts.
	if !waitForLineCount(t, assetsMarker, 1, 20*time.Second) {
		t.Fatal("assets did not run at startup")
	}
	if !waitForLineCount(t, apiMarker, 1, 20*time.Second) {
		t.Fatal("api did not start after assets completed")
	}

	// A change under the watched tree must re-run assets and restart api.
	touch(t, filepath.Join(watchDir, "app.scss"), "body{}")

	if !waitForLineCount(t, assetsMarker, 2, 20*time.Second) {
		t.Fatalf("assets did not re-run after the file change (ran %d times)", countLinesIn(t, assetsMarker))
	}
	if !waitForLineCount(t, apiMarker, 2, 20*time.Second) {
		t.Fatalf("api was not cascaded after assets re-ran (started %d times)", countLinesIn(t, apiMarker))
	}
}

// TestWatch_NoCascadeLeavesDependentAlone pins the direction rule: a change to
// a dependent must not re-run the dependency it relies on.
func TestWatch_NoCascadeLeavesDependentAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows: the marker commands use POSIX shell redirection")
	}
	watchDir := t.TempDir()
	markerDir := t.TempDir()
	assetsMarker := filepath.Join(markerDir, "assets.log")

	shell := command.DefaultShellConfig()
	procs := types.Processes{
		"assets": {
			Name:        "assets",
			ReplicaName: "assets",
			Replicas:    1,
			Executable:  shell.ShellCommand,
			Args:        []string{shell.ShellArgument, fmt.Sprintf("echo built >> %s", assetsMarker)},
			RestartPolicy: types.RestartPolicyConfig{
				Restart: types.RestartPolicyNo,
			},
		},
		"api": {
			Name:        "api",
			ReplicaName: "api",
			Replicas:    1,
			Executable:  shell.ShellCommand,
			Args:        []string{shell.ShellArgument, getSleepCommand(120.0)},
			DependsOn: types.DependsOnConfig{
				"assets": {Condition: types.ProcessConditionCompletedSuccessfully},
			},
			Watch: &types.WatchConfig{
				Debounce: "30ms",
				Paths:    []types.WatchPath{{Path: watchDir}},
			},
		},
	}

	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes:   procs,
		},
		processesToRun:  []string{},
		mainProcessArgs: []string{},
	})
	if err != nil {
		t.Fatalf("NewProjectRunner() error = %v", err)
	}
	startRunner(t, runner)

	if !waitForLineCount(t, assetsMarker, 1, 20*time.Second) {
		t.Fatal("assets did not run at startup")
	}
	waitForProcessState(t, runner, "api", types.ProcessStateRunning, 30*time.Second)
	if t.Failed() {
		return
	}

	touch(t, filepath.Join(watchDir, "main.go"), "package main")
	time.Sleep(3 * time.Second)

	if got := countLinesIn(t, assetsMarker); got != 1 {
		t.Errorf("assets ran %d times, want 1 (a restart must not propagate to a dependency)", got)
	}
}

// TestWatch_ExitOnEndSurvivesRestart is the regression guard for the worst
// failure mode: every exit routes through onProcessEnd, so without the
// intentional-restart marker a watch-triggered restart of an exit_on_end
// process would tear the whole project down on the first file save.
func TestWatch_ExitOnEndSurvivesRestart(t *testing.T) {
	watchDir := t.TempDir()
	runner := newWatchRunner(t, watchDir, func(procs types.Processes) {
		proc := procs["api"]
		proc.RestartPolicy = types.RestartPolicyConfig{ExitOnEnd: true}
		procs["api"] = proc
	})

	runDone := make(chan struct{})
	go func() { _ = runner.Run(); close(runDone) }()
	t.Cleanup(func() { _ = runner.ShutDownProject() })

	waitForProcessLaunched(t, runner, "api", 10*time.Second)
	waitForProcessState(t, runner, "api", types.ProcessStateRunning, 30*time.Second)
	if t.Failed() {
		return
	}

	touch(t, filepath.Join(watchDir, "main.go"), "package main")

	// The process must come back, and the project must still be running.
	waitForProcessState(t, runner, "api", types.ProcessStateRunning, 30*time.Second)
	if t.Failed() {
		return
	}
	select {
	case <-runDone:
		t.Fatal("project shut down after a watch-triggered restart of an exit_on_end process")
	default:
	}
}

// TestWatch_ProcessStateIsWatching pins the derived state that tells the user
// an exited one-shot is still armed - and explains why the project is up.
func TestWatch_ProcessStateIsWatching(t *testing.T) {
	watchDir := t.TempDir()
	shell := command.DefaultShellConfig()
	procs := types.Processes{
		"builder": {
			Name:        "builder",
			ReplicaName: "builder",
			Replicas:    1,
			Executable:  shell.ShellCommand,
			Args:        []string{shell.ShellArgument, "exit 0"},
			RestartPolicy: types.RestartPolicyConfig{
				Restart: types.RestartPolicyNo,
			},
			Watch: &types.WatchConfig{
				Debounce: "30ms",
				Paths:    []types.WatchPath{{Path: watchDir}},
			},
		},
	}
	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes:   procs,
		},
		processesToRun:  []string{},
		mainProcessArgs: []string{},
	})
	if err != nil {
		t.Fatalf("NewProjectRunner() error = %v", err)
	}

	runDone := make(chan struct{})
	go func() { _ = runner.Run(); close(runDone) }()
	t.Cleanup(func() { _ = runner.ShutDownProject() })

	waitForProcessState(t, runner, "builder", types.ProcessStateWatching, 30*time.Second)
	if t.Failed() {
		return
	}

	state, err := runner.GetProcessState("builder")
	if err != nil {
		t.Fatalf("GetProcessState() error = %v", err)
	}
	if !state.IsWatched {
		t.Error("IsWatched = false, want true; the remote client relies on this field")
	}

	// The armed watch must also hold the project open, or the watch could never
	// fire for a project made only of one-shot builders.
	select {
	case <-runDone:
		t.Fatal("project completed while a watch was still armed")
	default:
	}
}

// TestWatch_NoWatchFlagDisablesEverything - CI and scripted runs need `up` to
// behave exactly as it did before watching existed.
func TestWatch_NoWatchFlagDisablesEverything(t *testing.T) {
	watchDir := t.TempDir()
	shell := command.DefaultShellConfig()
	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes: types.Processes{
				"api": {
					Name:        "api",
					ReplicaName: "api",
					Replicas:    1,
					Executable:  shell.ShellCommand,
					Args:        []string{shell.ShellArgument, getSleepCommand(120.0)},
					Watch: &types.WatchConfig{
						Debounce: "30ms",
						Paths:    []types.WatchPath{{Path: watchDir}},
					},
				},
			},
		},
		processesToRun:  []string{},
		mainProcessArgs: []string{},
		noWatch:         true,
	})
	if err != nil {
		t.Fatalf("NewProjectRunner() error = %v", err)
	}
	startRunner(t, runner)
	waitForProcessLaunched(t, runner, "api", 10*time.Second)
	waitForProcessState(t, runner, "api", types.ProcessStateRunning, 30*time.Second)
	if t.Failed() {
		return
	}

	before, err := runner.GetProcessState("api")
	if err != nil {
		t.Fatalf("GetProcessState() error = %v", err)
	}

	touch(t, filepath.Join(watchDir, "main.go"), "package main")
	time.Sleep(2 * time.Second)

	after, err := runner.GetProcessState("api")
	if err != nil {
		t.Fatalf("GetProcessState() error = %v", err)
	}
	if after.Pid != before.Pid {
		t.Errorf("process restarted with --no-watch (pid %d -> %d)", before.Pid, after.Pid)
	}
	if after.IsWatched {
		t.Error("IsWatched = true with --no-watch")
	}
}

// TestWatch_StoppedProcessIsNotRestarted - a deliberately stopped process must
// stay stopped, however much its files change.
func TestWatch_StoppedProcessIsNotRestarted(t *testing.T) {
	watchDir := t.TempDir()
	runner := newWatchRunner(t, watchDir, nil)
	startRunner(t, runner)
	waitForProcessLaunched(t, runner, "api", 10*time.Second)
	waitForProcessState(t, runner, "api", types.ProcessStateRunning, 30*time.Second)
	if t.Failed() {
		return
	}

	if err := runner.StopProcess("api"); err != nil {
		t.Fatalf("StopProcess() error = %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	touch(t, filepath.Join(watchDir, "main.go"), "package main")
	time.Sleep(2 * time.Second)

	state, err := runner.GetProcessState("api")
	if err != nil {
		t.Fatalf("GetProcessState() error = %v", err)
	}
	if state.IsRunning {
		t.Error("a stopped process was restarted by a file change")
	}
}

// TestWatch_StopDisarmsWatchingProcess covers stopping a process that is not
// running but is armed. A one-shot builder sits in Watching once it completes,
// and asking it to stop used to be refused with "process is not running" - the
// process the user is looking at is precisely the one that has already exited,
// so the refusal made the state impossible to leave.
func TestWatch_StopDisarmsWatchingProcess(t *testing.T) {
	watchDir := t.TempDir()
	shell := command.DefaultShellConfig()
	procs := types.Processes{
		"builder": {
			Name:          "builder",
			ReplicaName:   "builder",
			Replicas:      1,
			Executable:    shell.ShellCommand,
			Args:          []string{shell.ShellArgument, "exit 0"},
			RestartPolicy: types.RestartPolicyConfig{Restart: types.RestartPolicyNo},
			Watch: &types.WatchConfig{
				Debounce: "30ms",
				Paths:    []types.WatchPath{{Path: watchDir}},
			},
		},
		// Keeps the project alive once the builder's watch is disarmed.
		"keepalive": {
			Name:        "keepalive",
			ReplicaName: "keepalive",
			Replicas:    1,
			Executable:  shell.ShellCommand,
			Args:        []string{shell.ShellArgument, getSleepCommand(120.0)},
		},
	}
	runner, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes:   procs,
		},
		processesToRun:  []string{},
		mainProcessArgs: []string{},
	})
	if err != nil {
		t.Fatalf("NewProjectRunner() error = %v", err)
	}
	startRunner(t, runner)

	waitForProcessState(t, runner, "builder", types.ProcessStateWatching, 30*time.Second)
	if t.Failed() {
		return
	}

	if err := runner.StopProcess("builder"); err != nil {
		t.Fatalf("StopProcess() on a Watching process error = %v, want nil", err)
	}

	state, err := runner.GetProcessState("builder")
	if err != nil {
		t.Fatalf("GetProcessState() error = %v", err)
	}
	if state.IsWatched {
		t.Error("IsWatched = true after stop, want false - the watch was not disarmed")
	}
	if state.Status == types.ProcessStateWatching {
		t.Error("status is still Watching after stop")
	}

	// And a change no longer brings it back.
	touch(t, filepath.Join(watchDir, "main.go"), "package main")
	time.Sleep(time.Second)
	if state, err := runner.GetProcessState("builder"); err == nil && state.IsRunning {
		t.Error("a stopped process was restarted by a file change")
	}

	// Stopping it a second time is now genuinely a no-op, and says so.
	if err := runner.StopProcess("builder"); err == nil {
		t.Error("StopProcess() on an already disarmed process error = nil, want 'not running'")
	}
}

// TestWatch_StartResumesPausedWatch covers the other half of the pause: a
// process started again after being stopped must be watchable once more, and
// must get there by resuming its existing registration rather than building a
// second one. StartProcess used to ask whether the watch was *armed*, which is
// false for exactly the case it was trying to detect, so it re-registered and
// re-walked the whole tree on every stop/start cycle.
func TestWatch_StartResumesPausedWatch(t *testing.T) {
	watchDir := t.TempDir()
	shell := command.DefaultShellConfig()
	// A second, unwatched process keeps the project alive. Without it, stopping
	// the only watched process empties the running set with no armed watch left
	// to hold the project open, so it completes and takes the watcher with it.
	runner := newWatchRunner(t, watchDir, func(procs types.Processes) {
		procs["keepalive"] = types.ProcessConfig{
			Name:        "keepalive",
			ReplicaName: "keepalive",
			Replicas:    1,
			Executable:  shell.ShellCommand,
			Args:        []string{shell.ShellArgument, getSleepCommand(120.0)},
		}
	})
	startRunner(t, runner)
	waitForProcessState(t, runner, "api", types.ProcessStateRunning, 30*time.Second)
	if t.Failed() {
		return
	}

	w := runner.processWatcher.Load()
	if w == nil {
		t.Fatal("no watcher was started")
	}
	entriesBefore := w.WatchedEntryCount()

	// Record a trigger before stopping. It lives on the registration, so it
	// survives a resume and is wiped by a rebuild - which is how this test tells
	// the two apart from outside the watcher package.
	triggered := runner.getRunningProcess("api")
	touch(t, filepath.Join(watchDir, "main.go"), "package main")
	if !waitFor(30*time.Second, func() bool {
		state, err := runner.GetProcessState("api")
		return err == nil && state.WatchTriggerPath != ""
	}) {
		t.Fatal("the watch never fired, so there is no trigger to preserve")
	}
	// The trigger is recorded as the restart starts, so stopping on it alone
	// lands in the gap between the teardown and the relaunch. That a stop wins
	// there is TestStopDuringRestartCancelsTheRestart's subject; here it would
	// just leave this test stopping a process that was already down. Wait for
	// the replacement instead.
	if !waitFor(30*time.Second, func() bool {
		restarted := runner.getRunningProcess("api")
		return restarted != nil && restarted != triggered && restarted.isRunning()
	}) {
		t.Fatal("the watch-triggered restart never produced a running process")
	}

	if err := runner.StopProcess("api"); err != nil {
		t.Fatalf("StopProcess() error = %v", err)
	}
	// The watch is paused, and IsRunning cleared, before the incarnation has
	// actually been deregistered - so wait on the same condition StartProcess
	// itself tests, or it refuses with "already running".
	if !waitFor(30*time.Second, func() bool { return runner.getRunningProcess("api") == nil }) {
		t.Fatal("the process did not stop")
	}
	if w.IsWatched("api") {
		t.Error("a stopped process's watch is still armed")
	}
	if !w.IsRegistered("api") {
		t.Error("a stopped process's watch registration was dropped instead of paused")
	}

	if err := runner.StartProcess("api"); err != nil {
		t.Fatalf("StartProcess() error = %v", err)
	}
	waitForProcessState(t, runner, "api", types.ProcessStateRunning, 30*time.Second)
	if t.Failed() {
		return
	}
	if !w.IsWatched("api") {
		t.Error("a restarted process's watch was not re-armed")
	}
	if got := w.WatchedEntryCount(); got != entriesBefore {
		t.Errorf("WatchedEntryCount() = %v after stop/start, want %v", got, entriesBefore)
	}
	state, err := runner.GetProcessState("api")
	if err != nil {
		t.Fatalf("GetProcessState() error = %v", err)
	}
	if state.WatchTriggerPath == "" {
		t.Error("start rebuilt the watch registration instead of resuming it: " +
			"the recorded trigger was lost")
	}

	// And the re-armed watch really fires.
	touch(t, filepath.Join(watchDir, "other.go"), "package main")
	if !waitFor(30*time.Second, func() bool {
		state, err := runner.GetProcessState("api")
		return err == nil && strings.HasSuffix(state.WatchTriggerPath, "other.go")
	}) {
		t.Error("a file change after start did not trigger the re-armed watch")
	}
}

// waitFor polls cond until it holds or the timeout elapses.
func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
