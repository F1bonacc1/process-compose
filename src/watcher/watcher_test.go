package watcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
)

// mockController records restart batches. The copy-under-lock accessors are
// what keep these tests clean under -race, mirroring the scheduler's
// mockProcessStarter.
type mockController struct {
	mtx        sync.Mutex
	batches    [][]string
	dependents map[string][]string
	err        error
	restarted  chan struct{}
}

func newMockController() *mockController {
	return &mockController{
		dependents: make(map[string][]string),
		restarted:  make(chan struct{}, 128),
	}
}

func (m *mockController) RestartProcesses(names []string) error {
	m.mtx.Lock()
	m.batches = append(m.batches, slices.Clone(names))
	err := m.err
	m.mtx.Unlock()
	select {
	case m.restarted <- struct{}{}:
	default:
	}
	return err
}

func (m *mockController) TransitiveDependents(name string) []string {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return slices.Clone(m.dependents[name])
}

func (m *mockController) GetProcessState(name string) (*types.ProcessState, error) {
	return &types.ProcessState{Name: name}, nil
}

func (m *mockController) allBatches() [][]string {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	out := make([][]string, len(m.batches))
	for i, batch := range m.batches {
		out[i] = slices.Clone(batch)
	}
	return out
}

func (m *mockController) batchCount() int {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return len(m.batches)
}

// waitForRestart blocks until a restart batch is recorded or the deadline passes.
func (m *mockController) waitForRestart(t *testing.T, timeout time.Duration) bool {
	t.Helper()
	select {
	case <-m.restarted:
		return true
	case <-time.After(timeout):
		return false
	}
}

// testOptions runs everything in milliseconds. This package has no clock
// injection, so short real durations are the only lever.
func testOptions() Options {
	return Options{
		DefaultDebounce: 15 * time.Millisecond,
		RescanSettle:    5 * time.Millisecond,
		Quiesce:         0, // off by default; TestWatcher_Quiesce* opts in
		FlapWindow:      time.Minute,
		FlapThreshold:   1000,
	}
}

func newTestWatcher(t *testing.T, ctrl ProjectController, opts Options) *Watcher {
	t.Helper()
	w, err := New(ctrl, opts)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = w.Stop() })
	return w
}

func watchCfg(dir string, mutate func(*types.WatchConfig)) *types.WatchConfig {
	cfg := &types.WatchConfig{
		Debounce: "15ms",
		Paths:    []types.WatchPath{{Path: dir}},
	}
	if mutate != nil {
		mutate(cfg)
	}
	return cfg
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func TestWatcher_RestartsOnFileChange(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	w := newTestWatcher(t, ctrl, testOptions())

	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()

	writeFile(t, filepath.Join(dir, "main.go"), "package main")

	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("no restart after a watched file changed")
	}
	batches := ctrl.allBatches()
	if len(batches[0]) != 1 || batches[0][0] != "api" {
		t.Errorf("first batch = %v, want [api]", batches[0])
	}
}

// TestWatcher_CascadeOrder is the differentiator: restarting a dependency must
// restart its dependents, and must do so root-first. The reverse order would
// let a dependent resolve against its dependency's stale completed incarnation
// and rebuild against old output.
func TestWatcher_CascadeOrder(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	ctrl.dependents["assets"] = []string{"api", "worker"}
	w := newTestWatcher(t, ctrl, testOptions())

	if err := w.AddProcess("assets", watchCfg(dir, func(c *types.WatchConfig) {
		c.Cascade = true
	})); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()

	writeFile(t, filepath.Join(dir, "app.scss"), "body{}")

	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("no restart after a watched file changed")
	}
	want := []string{"assets", "api", "worker"}
	if got := ctrl.allBatches()[0]; !slices.Equal(got, want) {
		t.Errorf("cascade batch = %v, want %v (root must come first)", got, want)
	}
}

// TestWatcher_CascadeIncludesRecentlyRestartedDependent guards a dependent that
// its own watch restarted moments before the cascade fires.
//
// The causality rule used to be applied to dependents as well, comparing their
// last restart against the *file event* time. A dependent does not watch that
// file, so a save landing in both trees at once - an editor's "save all" - let
// the dependent's own restart win the race and then excluded it from the
// cascade, leaving it running against output that was about to be rebuilt.
func TestWatcher_CascadeIncludesRecentlyRestartedDependent(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	ctrl.dependents["assets"] = []string{"api"}
	w := newTestWatcher(t, ctrl, testOptions())

	if err := w.AddProcess("assets", watchCfg(dir, func(c *types.WatchConfig) {
		c.Cascade = true
	})); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}

	// The save that fired this trigger, and an "api" restart that landed just
	// after it - the ordering that used to drop api from the batch.
	eventAt := time.Now().Add(20 * time.Millisecond)
	restartedAt := map[string]time.Time{"api": eventAt.Add(10 * time.Millisecond)}

	w.handleTrigger(
		trigger{proc: "assets", path: filepath.Join(dir, "app.scss"), at: eventAt},
		restartedAt,
		newFlapDetector(time.Minute, 1000),
	)

	batches := ctrl.allBatches()
	if len(batches) != 1 {
		t.Fatalf("restart batches = %v, want exactly one", batches)
	}
	if want := []string{"assets", "api"}; !slices.Equal(batches[0], want) {
		t.Errorf("cascade batch = %v, want %v", batches[0], want)
	}
}

// TestWatcher_BufferSizeIsPerProcess pins that watch.buffer_size reaches the
// registration. It was previously read into a config accessor that nothing
// called, so the option was inert and the Windows advice to raise it when
// events are dropped did nothing.
func TestWatcher_BufferSizeIsPerProcess(t *testing.T) {
	dir := t.TempDir()
	w := newTestWatcher(t, newMockController(), testOptions())

	small := 8192
	large := 262144

	if err := w.AddProcess("small", watchCfg(dir, func(c *types.WatchConfig) {
		c.BufferSize = small
	})); err != nil {
		t.Fatalf("AddProcess(small) error = %v", err)
	}
	if got := w.bufSizeOf(dir); got != small {
		t.Errorf("buffer size = %v, want %v (the process's own setting)", got, small)
	}

	// A second process sharing the directory and needing more must upgrade it:
	// the buffer only ever guards against dropped events, so the larger wins.
	if err := w.AddProcess("large", watchCfg(dir, func(c *types.WatchConfig) {
		c.BufferSize = large
	})); err != nil {
		t.Fatalf("AddProcess(large) error = %v", err)
	}
	if got := w.bufSizeOf(dir); got != large {
		t.Errorf("buffer size = %v, want %v (the larger request must win)", got, large)
	}

	// A third asking for less must not shrink it back.
	if err := w.AddProcess("third", watchCfg(dir, func(c *types.WatchConfig) {
		c.BufferSize = small
	})); err != nil {
		t.Fatalf("AddProcess(third) error = %v", err)
	}
	if got := w.bufSizeOf(dir); got != large {
		t.Errorf("buffer size = %v after a smaller request, want %v", got, large)
	}

	// An unset buffer_size falls back to the documented default.
	other := t.TempDir()
	if err := w.AddProcess("default", watchCfg(other, nil)); err != nil {
		t.Fatalf("AddProcess(default) error = %v", err)
	}
	if got, want := w.bufSizeOf(other), types.DefaultWatchBufferSize; got != want {
		t.Errorf("default buffer size = %v, want %v", got, want)
	}
}

// TestWatcher_ResumeReusesRegistration separates "registered" from "armed".
//
// A paused watch is not armed - it must not restart anything or hold the
// project open - but it is still fully built, and resuming it must reuse that
// work. Asking IsWatched to decide is what a caller does wrong: it is false for
// exactly the paused case, sending every stop/start through a full re-walk.
func TestWatcher_ResumeReusesRegistration(t *testing.T) {
	dir := t.TempDir()
	w := newTestWatcher(t, newMockController(), testOptions())
	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	original := w.procs["api"]

	if err := w.PauseProcess("api"); err != nil {
		t.Fatalf("PauseProcess() error = %v", err)
	}
	if w.IsWatched("api") {
		t.Error("IsWatched() = true for a paused watch, want false")
	}
	if !w.IsRegistered("api") {
		t.Error("IsRegistered() = false for a paused watch, want true")
	}

	if err := w.ResumeProcess("api"); err != nil {
		t.Fatalf("ResumeProcess() error = %v", err)
	}
	if !w.IsWatched("api") {
		t.Error("IsWatched() = false after resume, want true")
	}
	if w.procs["api"] != original {
		t.Error("resume rebuilt the registration instead of reusing it")
	}

	// AddProcess, by contrast, deliberately rebuilds - the expensive path a
	// merely paused process must not be sent down.
	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	if w.procs["api"] == original {
		t.Error("AddProcess() kept a stale registration, want a rebuilt one")
	}
}

// TestWatcher_AdoptRespectsEntryCap covers directories created while the
// project runs. scanTree caps the initial walk, but adoption happens one
// directory at a time long afterwards - so without its own budget check a code
// generator or an unpacked dependency tree walks straight past max_entries and
// exhausts the inotify watch limit the cap exists to protect.
func TestWatcher_AdoptRespectsEntryCap(t *testing.T) {
	dir := t.TempDir()
	w := newTestWatcher(t, newMockController(), testOptions())

	// A cap of two: the root itself takes one, leaving room for exactly one
	// more directory.
	if err := w.AddProcess("api", watchCfg(dir, func(c *types.WatchConfig) {
		c.MaxEntries = 2
	})); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	if got := w.WatchedEntryCount(); got != 1 {
		t.Fatalf("WatchedEntryCount() = %v after registering the root, want 1", got)
	}
	w.Start()

	mkdir := func(name string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", path, err)
		}
		return path
	}
	waitForEntries := func(want int) bool {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if w.WatchedEntryCount() == want {
				return true
			}
			time.Sleep(20 * time.Millisecond)
		}
		return false
	}

	first := mkdir("first")
	if !waitForEntries(2) {
		t.Fatalf("WatchedEntryCount() = %v, want 2 (the first new directory should be adopted)",
			w.WatchedEntryCount())
	}

	mkdir("second")
	time.Sleep(300 * time.Millisecond)
	if got := w.WatchedEntryCount(); got != 2 {
		t.Errorf("WatchedEntryCount() = %v, want 2 - a directory was adopted past max_entries", got)
	}

	// Removing a watched directory must refund its budget, or a project that
	// churns directories would drift into a permanent false cap.
	if err := os.RemoveAll(first); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}
	if !waitForEntries(1) {
		t.Fatalf("WatchedEntryCount() = %v after removing a watched directory, want 1",
			w.WatchedEntryCount())
	}
	mkdir("third")
	if !waitForEntries(2) {
		t.Errorf("WatchedEntryCount() = %v, want 2 - the freed budget was not reused",
			w.WatchedEntryCount())
	}
}

func TestWatcher_NoCascadeWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	ctrl.dependents["assets"] = []string{"api"}
	w := newTestWatcher(t, ctrl, testOptions())

	if err := w.AddProcess("assets", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()

	writeFile(t, filepath.Join(dir, "app.scss"), "body{}")

	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("no restart after a watched file changed")
	}
	if got := ctrl.allBatches()[0]; !slices.Equal(got, []string{"assets"}) {
		t.Errorf("batch = %v, want [assets] with cascade off", got)
	}
}

func TestWatcher_ExcludedFileDoesNotRestart(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	w := newTestWatcher(t, ctrl, testOptions())

	cfg := watchCfg(dir, func(c *types.WatchConfig) {
		c.Paths[0].Exclude = []string{"*_test.go"}
	})
	if err := w.AddProcess("api", cfg); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()

	writeFile(t, filepath.Join(dir, "main_test.go"), "package main")
	time.Sleep(250 * time.Millisecond)
	if got := ctrl.batchCount(); got != 0 {
		t.Fatalf("restarted %d times on an excluded file, want 0", got)
	}

	// A non-excluded file in the same directory must still fire, proving the
	// watch itself is live.
	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("no restart for a non-excluded file")
	}
}

func TestWatcher_IncludeActsAsAllowlist(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	w := newTestWatcher(t, ctrl, testOptions())

	cfg := watchCfg(dir, func(c *types.WatchConfig) {
		c.Paths[0].Include = []string{"**/*.go"}
	})
	if err := w.AddProcess("api", cfg); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()

	writeFile(t, filepath.Join(dir, "README.md"), "docs")
	time.Sleep(250 * time.Millisecond)
	if got := ctrl.batchCount(); got != 0 {
		t.Fatalf("restarted %d times on a non-included file, want 0", got)
	}

	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("no restart for an included file")
	}
}

// TestWatcher_AdoptsNewDirectory covers the mkdir race window. fsnotify has no
// recursive watching, so a directory created after startup must be picked up by
// hand - and files created inside it before the watch lands are only found by
// the rescan.
func TestWatcher_AdoptsNewDirectory(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	w := newTestWatcher(t, ctrl, testOptions())

	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()

	// Create the directory and immediately fill it, the way a code generator or
	// a branch switch would.
	newPkg := filepath.Join(dir, "newpkg")
	if err := os.Mkdir(newPkg, 0o755); err != nil {
		t.Fatalf("Mkdir error = %v", err)
	}
	writeFile(t, filepath.Join(newPkg, "gen.go"), "package newpkg")

	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("no restart after a new directory was created and filled")
	}

	// Drain, then prove the new directory is genuinely watched going forward.
	time.Sleep(200 * time.Millisecond)
	before := ctrl.batchCount()
	writeFile(t, filepath.Join(newPkg, "later.go"), "package newpkg")
	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("new directory was adopted but not watched")
	}
	if ctrl.batchCount() <= before {
		t.Error("no additional restart for a change inside the adopted directory")
	}
}

func TestWatcher_PausedDoesNotRestart(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	w := newTestWatcher(t, ctrl, testOptions())

	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()
	if err := w.PauseProcess("api"); err != nil {
		t.Fatalf("PauseProcess() error = %v", err)
	}

	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	time.Sleep(250 * time.Millisecond)
	if got := ctrl.batchCount(); got != 0 {
		t.Fatalf("restarted %d times while paused, want 0", got)
	}
	if w.IsWatched("api") {
		t.Error("IsWatched() = true while paused; a paused watch cannot restart anything")
	}

	if err := w.ResumeProcess("api"); err != nil {
		t.Fatalf("ResumeProcess() error = %v", err)
	}
	if !w.IsWatched("api") {
		t.Error("IsWatched() = false after resume")
	}
	writeFile(t, filepath.Join(dir, "main.go"), "package main // v2")
	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("no restart after resume")
	}
}

func TestWatcher_RemoveProcessReleasesWatches(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	w := newTestWatcher(t, ctrl, testOptions())

	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	if got := w.WatchedEntryCount(); got == 0 {
		t.Fatal("WatchedEntryCount() = 0 after AddProcess")
	}
	w.Start()

	if err := w.RemoveProcess("api"); err != nil {
		t.Fatalf("RemoveProcess() error = %v", err)
	}
	if got := w.WatchedEntryCount(); got != 0 {
		t.Errorf("WatchedEntryCount() = %v after RemoveProcess, want 0", got)
	}
	if w.IsWatched("api") {
		t.Error("IsWatched() = true after RemoveProcess")
	}

	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	time.Sleep(250 * time.Millisecond)
	if got := ctrl.batchCount(); got != 0 {
		t.Errorf("restarted %d times after RemoveProcess, want 0", got)
	}
}

// TestWatcher_SharedDirectoryRefcount pins that two processes watching one tree
// share registrations - on kqueue every entry costs a descriptor - and that
// removing one does not blind the other.
func TestWatcher_SharedDirectoryRefcount(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	w := newTestWatcher(t, ctrl, testOptions())

	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess(api) error = %v", err)
	}
	afterFirst := w.WatchedEntryCount()

	if err := w.AddProcess("worker", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess(worker) error = %v", err)
	}
	if got := w.WatchedEntryCount(); got != afterFirst {
		t.Errorf("WatchedEntryCount() = %v after a second process on the same tree, want %v (registrations must be shared)", got, afterFirst)
	}

	w.Start()
	if err := w.RemoveProcess("api"); err != nil {
		t.Fatalf("RemoveProcess() error = %v", err)
	}
	if got := w.WatchedEntryCount(); got != afterFirst {
		t.Errorf("WatchedEntryCount() = %v after removing one of two sharers, want %v", got, afterFirst)
	}

	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("remaining process stopped receiving events when its co-watcher was removed")
	}
	if got := ctrl.allBatches()[0]; !slices.Equal(got, []string{"worker"}) {
		t.Errorf("batch = %v, want [worker]", got)
	}
}

func TestWatcher_GetWatchedProcesses(t *testing.T) {
	dir := t.TempDir()
	w := newTestWatcher(t, newMockController(), testOptions())

	if got := w.GetWatchedProcesses(); len(got) != 0 {
		t.Errorf("GetWatchedProcesses() = %v, want empty", got)
	}
	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	if err := w.AddProcess("worker", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}

	got := w.GetWatchedProcesses()
	slices.Sort(got)
	if !slices.Equal(got, []string{"api", "worker"}) {
		t.Errorf("GetWatchedProcesses() = %v, want [api worker]", got)
	}

	// A paused watch must not count: it cannot restart anything, so it must
	// not hold the project open either.
	if err := w.PauseProcess("api"); err != nil {
		t.Fatalf("PauseProcess() error = %v", err)
	}
	if got := w.GetWatchedProcesses(); !slices.Equal(got, []string{"worker"}) {
		t.Errorf("GetWatchedProcesses() = %v after pausing api, want [worker]", got)
	}
}

func TestWatcher_AddProcessIgnoresDisabledConfig(t *testing.T) {
	w := newTestWatcher(t, newMockController(), testOptions())

	if err := w.AddProcess("api", nil); err != nil {
		t.Errorf("AddProcess(nil) error = %v, want nil", err)
	}
	if err := w.AddProcess("api", &types.WatchConfig{}); err != nil {
		t.Errorf("AddProcess(empty) error = %v, want nil", err)
	}
	if w.IsWatched("api") {
		t.Error("IsWatched() = true for a config with no paths")
	}
}

func TestWatcher_AddProcessRejectsMissingPath(t *testing.T) {
	w := newTestWatcher(t, newMockController(), testOptions())
	cfg := &types.WatchConfig{Paths: []types.WatchPath{{Path: filepath.Join(t.TempDir(), "nope")}}}
	if err := w.AddProcess("api", cfg); err == nil {
		t.Error("AddProcess() error = nil, want a stat error for a missing path")
	}
}

func TestWatcher_AddProcessEnforcesEntryCap(t *testing.T) {
	root := mkTree(t, "a/x.go", "b/x.go", "c/x.go", "d/x.go")
	w := newTestWatcher(t, newMockController(), testOptions())

	cfg := &types.WatchConfig{
		MaxEntries: 2,
		Paths:      []types.WatchPath{{Path: root}},
	}
	err := w.AddProcess("api", cfg)
	if err == nil {
		t.Fatal("AddProcess() error = nil, want a TooManyEntriesError")
	}
	var tooMany *TooManyEntriesError
	if !errors.As(err, &tooMany) {
		t.Fatalf("AddProcess() error = %T, want *TooManyEntriesError", err)
	}
}

// TestWatcher_FlapSuspends covers the feedback loop a process creates by
// building into a directory it also watches. No causality rule can break it,
// so the watch must be suspended by rate instead of running away.
func TestWatcher_FlapSuspends(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	opts := testOptions()
	opts.FlapWindow = 10 * time.Second
	opts.FlapThreshold = 3
	w := newTestWatcher(t, ctrl, opts)

	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && w.IsWatched("api") {
		writeFile(t, filepath.Join(dir, "main.go"), time.Now().String())
		time.Sleep(40 * time.Millisecond)
	}

	if w.IsWatched("api") {
		t.Fatal("watch was never suspended despite a sustained restart loop")
	}
	if got := ctrl.batchCount(); got >= opts.FlapThreshold+3 {
		t.Errorf("restarted %d times before suspending, want the loop cut near the threshold of %d", got, opts.FlapThreshold)
	}

	// A suspended watch must stop restarting entirely.
	before := ctrl.batchCount()
	writeFile(t, filepath.Join(dir, "main.go"), "after suspension")
	time.Sleep(300 * time.Millisecond)
	if ctrl.batchCount() != before {
		t.Error("suspended watch still restarted the process")
	}
}

// TestWatcher_StopIsCleanAndIdempotent guards against goroutine leaks and
// against a panic on a timer that fires during shutdown.
func TestWatcher_StopIsCleanAndIdempotent(t *testing.T) {
	dir := t.TempDir()
	baseline := runtime.NumGoroutine()

	ctrl := newMockController()
	w, err := New(ctrl, testOptions())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()

	// Fire events right up to the moment of shutdown, so a debouncer timer is
	// very likely in flight while Stop runs. Directories are created alongside
	// the writes because each one spawns a deferred rescan, which Stop has to
	// account for as well.
	for i := range 20 {
		writeFile(t, filepath.Join(dir, "main.go"), time.Now().String())
		if err := os.MkdirAll(filepath.Join(dir, fmt.Sprintf("pkg%d", i)), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if i%5 == 0 {
			time.Sleep(5 * time.Millisecond)
		}
	}

	if err := w.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
	if err := w.Stop(); err != nil {
		t.Errorf("second Stop() error = %v, want nil (Stop must be idempotent)", err)
	}

	// Let any straggler unwind before counting.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= baseline+2 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("goroutines = %d after Stop, want <= %d (leak)", runtime.NumGoroutine(), baseline+2)
}

func TestWatcher_StartIsIdempotent(t *testing.T) {
	w := newTestWatcher(t, newMockController(), testOptions())
	w.Start()
	w.Start()
	w.Start()
	if err := w.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

// TestWatcher_RestartErrorDoesNotKillDispatcher - a failing restart must be
// logged and survived, not take the watcher down with it.
func TestWatcher_RestartErrorDoesNotKillDispatcher(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	ctrl.err = errors.New("boom")
	w := newTestWatcher(t, ctrl, testOptions())

	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()

	writeFile(t, filepath.Join(dir, "main.go"), "one")
	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("no first restart")
	}

	time.Sleep(100 * time.Millisecond)
	writeFile(t, filepath.Join(dir, "main.go"), "two")
	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("dispatcher stopped working after a restart error")
	}
}

// TestWatcher_QuiesceOffByDefault pins the deliberate default. Quiescing drops
// events rather than deferring them, so leaving it on would silently lose an
// edit saved just after a restart - worse than an extra restart.
func TestWatcher_QuiesceOffByDefault(t *testing.T) {
	if got := (Options{}).withDefaults().Quiesce; got != 0 {
		t.Errorf("default Quiesce = %v, want 0 (off)", got)
	}
	if got := (Options{Quiesce: -1}).withDefaults().Quiesce; got != 0 {
		t.Errorf("negative Quiesce = %v, want 0 (off)", got)
	}
	if got := (Options{Quiesce: time.Second}).withDefaults().Quiesce; got != time.Second {
		t.Errorf("explicit Quiesce = %v, want 1s", got)
	}
}

// TestWatcher_ConsecutiveEditsBothRestart is the regression guard for the bug
// the quiesce default hid: two saves in quick succession must both be acted on.
func TestWatcher_ConsecutiveEditsBothRestart(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	w := newTestWatcher(t, ctrl, testOptions())

	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()

	writeFile(t, filepath.Join(dir, "main.go"), "first edit")
	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("no restart after the first edit")
	}

	time.Sleep(100 * time.Millisecond)
	writeFile(t, filepath.Join(dir, "main.go"), "second edit")
	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("second edit was swallowed; a save must never be silently dropped")
	}
}

// TestWatcher_QuiesceAbsorbsBuildOutput pins the window that swallows the
// output a rebuild writes into its own watched tree, for users who opt in.
func TestWatcher_QuiesceAbsorbsBuildOutput(t *testing.T) {
	dir := t.TempDir()
	ctrl := newMockController()
	opts := testOptions()
	opts.Quiesce = 750 * time.Millisecond
	w := newTestWatcher(t, ctrl, opts)

	if err := w.AddProcess("api", watchCfg(dir, nil)); err != nil {
		t.Fatalf("AddProcess() error = %v", err)
	}
	w.Start()

	writeFile(t, filepath.Join(dir, "main.go"), "source edit")
	if !ctrl.waitForRestart(t, 5*time.Second) {
		t.Fatal("no restart after the source edit")
	}
	after := ctrl.batchCount()

	// Simulate the build writing its artifacts immediately afterwards.
	for range 5 {
		writeFile(t, filepath.Join(dir, "artifact.bin"), time.Now().String())
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)

	if got := ctrl.batchCount(); got != after {
		t.Errorf("restarted %d more times during the quiesce window, want 0", got-after)
	}
}
