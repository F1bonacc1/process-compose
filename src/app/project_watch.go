package app

import (
	"errors"
	"time"

	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/f1bonacc1/process-compose/src/watcher"
	"github.com/rs/zerolog/log"
)

// startWatcher creates and starts the file watcher, registering every eligible
// process. Failing to build it is logged and survived rather than fatal, the
// same way a scheduler that fails to construct does not stop the project.
//
// Called after the initial run order has been launched: registering earlier
// would let a startup-time filesystem event restart a process whose first
// runProcess is still in flight, producing two incarnations.
func (p *ProjectRunner) startWatcher() {
	// Snapshot once, under the lock. Registering straight off p.project.Processes
	// would iterate a map that UpdateProject can rewrite from an API goroutine,
	// and a concurrent map read/write aborts the process outright.
	eligible := p.watchEligibleProcesses()
	if p.noWatch {
		if len(eligible) > 0 {
			log.Info().Msg("File watching is disabled (--no-watch); 'watch' blocks are ignored")
		}
		return
	}
	if len(eligible) == 0 {
		return
	}

	w, err := watcher.New(p, watcher.Options{
		AlwaysExclude: p.watchAlwaysExclude(),
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create file watcher")
		return
	}
	p.processWatcher.Store(w)

	for i := range eligible {
		p.watchAdd(&eligible[i])
	}
	if watched := w.GetWatchedProcesses(); len(watched) > 0 {
		log.Info().Msgf("Watching files for %d process(es); the project stays up while any watch is armed", len(watched))
	}
	w.Start()
}

// watchAlwaysExclude collects process-compose's own log files so a watch on the
// project directory does not retrigger on output the project itself produces.
// These are never user content, so they are dropped even when a process sets
// disable_default_excludes.
func (p *ProjectRunner) watchAlwaysExclude() []string {
	paths := []string{config.GetLogFilePath()}

	p.procConfMutex.Lock()
	if isStringDefined(p.project.LogLocation) {
		paths = append(paths, p.project.LogLocation)
	}
	for _, proc := range p.project.Processes {
		if isStringDefined(proc.LogLocation) {
			paths = append(paths, proc.LogLocation)
		}
	}
	p.procConfMutex.Unlock()

	patterns := make([]string, 0, len(paths)*3)
	for _, path := range paths {
		patterns = append(patterns, watcher.LogExclusionPatterns(path)...)
	}
	return patterns
}

// watchEligibleProcesses returns a copy of every process that should carry a
// watch. Copying under the lock lets the caller register them without holding
// it, which matters because registration walks the filesystem.
func (p *ProjectRunner) watchEligibleProcesses() []types.ProcessConfig {
	p.procConfMutex.Lock()
	defer p.procConfMutex.Unlock()
	eligible := make([]types.ProcessConfig, 0, len(p.project.Processes))
	for _, proc := range p.project.Processes {
		if isWatchEligible(&proc) {
			eligible = append(eligible, proc)
		}
	}
	return eligible
}

// isWatchEligible reports whether a process should carry a watch.
//
// IsDeferred covers disabled: true, is_disabled, PC_DISABLE_PROCESSES,
// is_foreground and - importantly - the `up <process>...` selection, which
// marks unselected processes disabled rather than removing them. Without this
// test a file change would start a process the user explicitly deselected,
// because neither StartProcess nor doRestart checks Disabled.
//
// MCP processes are excluded because they are started per tool invocation.
func isWatchEligible(proc *types.ProcessConfig) bool {
	return proc.Watch.IsEnabled() && !proc.IsDeferred() && !proc.IsMCP()
}

func (p *ProjectRunner) watchAdd(proc *types.ProcessConfig) {
	w := p.processWatcher.Load()
	if w == nil || !isWatchEligible(proc) {
		return
	}
	if err := w.AddProcess(proc.ReplicaName, proc.Watch); err != nil {
		log.Error().Err(err).Msgf("Failed to watch process %s", proc.ReplicaName)
	}
}

func (p *ProjectRunner) watchRemove(name string) {
	if w := p.processWatcher.Load(); w != nil {
		_ = w.RemoveProcess(name)
	}
}

func (p *ProjectRunner) watchPause(name string) {
	if w := p.processWatcher.Load(); w != nil {
		_ = w.PauseProcess(name)
	}
}

func (p *ProjectRunner) watchResume(name string) {
	if w := p.processWatcher.Load(); w != nil {
		_ = w.ResumeProcess(name)
	}
}

// stopWatcher shuts the watcher down. Called first thing in ShutDownProject, so
// that a file touched by a shutdown.command cannot trigger a restart of a
// process that is already being torn down.
func (p *ProjectRunner) stopWatcher() {
	if w := p.processWatcher.Swap(nil); w != nil {
		if err := w.Stop(); err != nil {
			log.Error().Err(err).Msg("Failed to stop the file watcher")
		}
	}
}

// isProcessWatched reports whether an active watch is armed for name. Used to
// derive the Watching state.
func (p *ProjectRunner) isProcessWatched(name string) bool {
	w := p.processWatcher.Load()
	return w != nil && w.IsWatched(name)
}

// isProcessWatchRegistered reports whether name has a watch registration at
// all, armed or paused. StartProcess needs this rather than isProcessWatched:
// a stopped process's watch is paused, so asking whether it is armed would
// always say no and send every restart down the re-register path.
func (p *ProjectRunner) isProcessWatchRegistered(name string) bool {
	w := p.processWatcher.Load()
	return w != nil && w.IsRegistered(name)
}

// lastWatchTrigger returns the change that last restarted name, so the state -
// and through it a local or attached TUI - can report why it restarted.
func (p *ProjectRunner) lastWatchTrigger(name string) (string, *time.Time) {
	w := p.processWatcher.Load()
	if w == nil {
		return "", nil
	}
	info, ok := w.LastTrigger(name)
	if !ok {
		return "", nil
	}
	at := info.At
	return info.Path, &at
}

// hasBackgroundTriggers reports whether anything outside the running process
// set can still start a process. While it is true, an empty running set must
// not complete the project - otherwise a project of one-shot builders would
// exit the moment they finished, before a watch could ever fire.
func (p *ProjectRunner) hasBackgroundTriggers() bool {
	if s := p.processScheduler.Load(); s != nil && len(s.GetScheduledProcesses()) > 0 {
		return true
	}
	if w := p.processWatcher.Load(); w != nil && len(w.GetWatchedProcesses()) > 0 {
		return true
	}
	return false
}

// RestartProcesses implements watcher.ProjectController.
//
// The whole batch is wrapped in beginUpdate because runningProcesses can
// legitimately hit zero in the gaps between the individual restarts of a
// cascade, and the completion loop would otherwise read that as the project
// having finished.
//
// Backoff is skipped and the restart counter reset: this is a restart the user
// asked for by saving a file, not a crash loop being damped.
func (p *ProjectRunner) RestartProcesses(names []string) error {
	defer p.beginUpdate()()
	opts := restartOpts{skipBackoff: true, resetRestarts: true}
	errs := make([]error, 0, len(names))
	for _, name := range names {
		if err := p.restartProcessWithOpts(name, opts); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// TransitiveDependents implements watcher.ProjectController.
func (p *ProjectRunner) TransitiveDependents(name string) []string {
	p.procConfMutex.Lock()
	defer p.procConfMutex.Unlock()
	return types.TransitiveDependents(p.project.Processes, name)
}
