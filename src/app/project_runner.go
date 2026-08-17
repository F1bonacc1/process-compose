package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/f1bonacc1/process-compose/src/admitter"
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/scheduler"
	"github.com/f1bonacc1/process-compose/src/templater"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/f1bonacc1/process-compose/src/watcher"

	"github.com/rs/zerolog/log"
)

// pendingTerminationTimeout bounds how long a restart waits for a process that
// never started (one still waiting on its dependencies) to release its
// goroutine before the replacement instance is launched.
const pendingTerminationTimeout = 5 * time.Second

type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("project non-zero exit code: %d", e.Code)
}

type ProjectRunner struct {
	procConfMutex sync.Mutex
	project       *types.Project
	logsMutex     sync.Mutex
	processLogs   map[string]*pclog.ProcessLogBuffer
	statesMutex   sync.Mutex
	processStates map[string]*types.ProcessState
	// pendingRestartReset marks processes whose next incarnation should start
	// with a zeroed restart counter. Guarded by statesMutex.
	pendingRestartReset map[string]bool
	runProcMutex        sync.Mutex
	runningProcesses    map[string]*Process
	doneProcMutex       sync.Mutex
	doneProcesses       map[string]*Process
	restartMutex        sync.Mutex
	restartCalls        map[string]*RestartCall
	logger              pclog.PcLogger
	//waitGroup            sync.WaitGroup
	//waitGroup            sync.WaitGroup
	exitCodeMutex        sync.Mutex
	exitCode             int
	projectState         *types.ProjectState
	processesToRun       []string
	noDeps               bool
	mainProcess          string
	mainProcessArgs      []string
	isTuiOn              bool
	isOrderedShutdown    bool
	ctxApp               context.Context
	cancelAppFn          context.CancelFunc
	disableDotenv        bool
	truncateLogs         bool
	refRate              time.Duration
	withRecursiveMetrics bool
	procCompleteChannel  chan int
	updatesInFlight      atomic.Int32
	processTree          *ProcessTree
	noWatch              bool
	processScheduler     atomic.Pointer[scheduler.Scheduler]
	processWatcher       atomic.Pointer[watcher.Watcher]
	stateBroadcaster     *ProcessStateBroadcaster
	admitters            []admitter.Admitter
}

// RestartCall represents an in-flight restart operation
type RestartCall struct {
	wg  sync.WaitGroup
	err error
}

func (p *ProjectRunner) GetLexicographicProcessNames() ([]string, error) {
	return p.project.GetLexicographicProcessNames()
}

func (p *ProjectRunner) init() {
	p.initProcessStates()
	p.initProcessLogs()
	p.initRestartCoalescing()
	p.processTree = NewProcessTree(p.refRate)
	p.stateBroadcaster = NewProcessStateBroadcaster(p.snapshotProcessStates)
}

// snapshotProcessStates returns the current state of every process. Used by
// the state broadcaster to deliver an initial snapshot to new subscribers.
// Errors from per-process lookups are logged and the offending process is
// skipped so the snapshot remains best-effort consistent.
func (p *ProjectRunner) snapshotProcessStates() []types.ProcessState {
	states, err := p.GetProcessesState()
	if err != nil || states == nil {
		log.Err(err).Msg("Failed to snapshot process states for broadcaster")
		return nil
	}
	return states.States
}

// RegisterStateObserver registers an observer that receives an initial
// snapshot of every process followed by every state change.
func (p *ProjectRunner) RegisterStateObserver(o types.StateObserver) {
	if p.stateBroadcaster == nil {
		return
	}
	p.stateBroadcaster.SubscribeWithSnapshot(o)
}

// UnregisterStateObserver stops delivery to the given observer.
func (p *ProjectRunner) UnregisterStateObserver(o types.StateObserver) {
	if p.stateBroadcaster == nil {
		return
	}
	p.stateBroadcaster.Unsubscribe(o)
}

// publishProcessState is the publish callback injected into each Process. It
// is safe to call before the broadcaster has been wired up (during early
// initialization) — events are simply dropped.
func (p *ProjectRunner) publishProcessState(ev types.ProcessStateEvent) {
	if p.stateBroadcaster == nil {
		return
	}
	p.stateBroadcaster.Publish(ev)
}

func (p *ProjectRunner) Run() error {
	p.runProcMutex.Lock()
	p.runningProcesses = make(map[string]*Process)
	p.runProcMutex.Unlock()
	p.doneProcMutex.Lock()
	p.doneProcesses = make(map[string]*Process)
	p.doneProcMutex.Unlock()
	runOrder := []types.ProcessConfig{}
	err := p.project.WithProcesses([]string{}, func(process types.ProcessConfig) error {
		if process.IsDeferred() {
			return nil
		}
		runOrder = append(runOrder, process)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to build project run order: %w", err)
	}
	var nameOrder []string
	for _, v := range runOrder {
		nameOrder = append(nameOrder, v.ReplicaName)
	}
	p.logger = pclog.NewNilLogger()
	if isStringDefined(p.project.LogLocation) {
		p.logger = pclog.NewLogger()
		p.logger.Open(p.project.LogLocation, p.project.LoggerConfig)
		defer p.logger.Close()
	}
	p.prepareEnvCmds()
	//zerolog.SetGlobalLevel(zerolog.PanicLevel)
	log.Debug().Msgf("Spinning up %d processes. Order: %q", len(runOrder), nameOrder)

	// Initialize and start scheduler for scheduled processes
	sched, err := scheduler.New(p)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create scheduler")
	} else {
		p.processScheduler.Store(sched)
		for name, proc := range p.project.Processes {
			if proc.Schedule != nil && proc.Schedule.IsScheduled() {
				if err := sched.AddProcess(name, proc.Schedule); err != nil {
					log.Error().Err(err).Msgf("Failed to schedule process %s", name)
				} else if proc.Disabled {
					if err := sched.PauseProcess(name); err != nil {
						log.Error().Err(err).Msgf("Failed to pause schedule for disabled process %s", name)
					}
				}
			}
		}
		sched.Start()
		defer func() {
			if err := sched.Stop(); err != nil {
				log.Error().Err(err).Msg("Failed to stop scheduler gracefully")
			}
		}()
	}

	for _, proc := range runOrder {
		if proc.Schedule != nil && proc.Schedule.IsScheduled() {
			continue
		}
		newConf := proc
		p.runProcess(&newConf)
	}

	// Started after the run order, deliberately: a watch registered earlier
	// could fire on start-up churn and restart a process whose first launch is
	// still in flight, leaving two incarnations.
	p.startWatcher()
	defer p.stopWatcher()

	for {
		select {
		case <-p.ctxApp.Done():
			p.exitCodeMutex.Lock()
			exitCode := p.exitCode
			p.exitCodeMutex.Unlock()
			if exitCode != 0 {
				return &ExitError{exitCode}
			}
			return err
		case runProcCount := <-p.procCompleteChannel:
			log.Debug().Msgf("Remaining processes: %d", runProcCount)
			if runProcCount == 0 {
				p.restartMutex.Lock()
				pendingRestarts := len(p.restartCalls)
				p.restartMutex.Unlock()
				if pendingRestarts > 0 {
					log.Debug().Msgf("Skipping project completion: %d restart(s) in progress", pendingRestarts)
					continue
				}
				if updates := p.updatesInFlight.Load(); updates > 0 {
					log.Debug().Msgf("Skipping project completion: %d update(s) in progress", updates)
					continue
				}
				// The count carried by the message is a snapshot taken when a
				// process exited. By the time it is read here a replacement may
				// already be running (restart, update, scale), so re-read the
				// live count - a stale zero must not complete a project that
				// still has running processes.
				p.runProcMutex.Lock()
				running := len(p.runningProcesses)
				p.runProcMutex.Unlock()
				if running > 0 {
					log.Debug().Msgf("Skipping project completion: %d process(es) still running", running)
					continue
				}
				// Scheduled processes and armed watchers can both still start
				// something, so an empty running set is not the end of the
				// project while either is live.
				if !p.hasBackgroundTriggers() {
					log.Info().Msg("Project completed")
					p.exitCodeMutex.Lock()
					exitCode := p.exitCode
					p.exitCodeMutex.Unlock()
					if exitCode != 0 {
						err = &ExitError{exitCode}
					}
					return err
				}
			}
		}
	}
}

func (p *ProjectRunner) runProcess(config *types.ProcessConfig) {
	procLogger := p.logger
	if isStringDefined(config.LogLocation) {
		procLogger = pclog.NewLogger()
	}
	procLog, err := p.getProcessLog(config.ReplicaName)
	if err != nil {
		// we shouldn't get here
		log.Error().Msgf("Error: Can't get log: %s using empty buffer", err.Error())
		procLog = pclog.NewLogBuffer(0)
	}
	procState := p.newIncarnationState(config)
	isMain := config.Name == p.mainProcess
	hasMain := p.mainProcess != ""
	printLogs := !hasMain && !p.isTuiOn && !p.project.MCPServer.IsStdio()
	extraArgs := []string{}
	if isMain {
		extraArgs = p.mainProcessArgs
		config.RestartPolicy.ExitOnEnd = true
	}
	process := NewProcess(
		withTuiOn(p.isTuiOn),
		withGlobalEnv(p.project.Environment),
		withDotEnv(p.project.DotEnvVars),
		withLogger(procLogger),
		withProcConf(config),
		withProcState(procState),
		withProcLog(procLog),
		withShellConfig(*p.project.ShellConfig),
		withPrintLogs(printLogs),
		withIsMain(isMain),
		withExtraArgs(extraArgs),
		withLogsTruncate(p.truncateLogs || config.TruncateLog),
		withRefRate(p.refRate),
		withRecursiveMetrics(p.withRecursiveMetrics),
		withProcessTree(p.processTree),
		withStatePublisher(p.publishProcessState),
	)
	p.addRunningProcess(process)
	go func(proc *Process) {
		defer proc.onTerminated()
		if waitErr := p.waitIfNeeded(proc); waitErr != nil {
			if errors.Is(waitErr, errWaitAborted) {
				log.Debug().Msgf("Process %s was stopped while waiting for its dependencies", proc.getName())
				// The stop path marks a Pending process as done; make sure of it
				// so that anything waiting on this instance is released.
				if !proc.isDone() {
					proc.onProcessEnd(types.ProcessStateTerminating)
				}
			} else {
				log.Error().Msgf("Error: %s", waitErr.Error())
				log.Error().Msgf("Error: process %s won't run", proc.getName())
				proc.wontRun()
				p.onProcessSkipped(proc.procConf)
			}
		} else {
			exitCode := proc.run()
			p.addDoneProcess(proc)
			// An exit caused by a restart is not the process ending: a
			// SIGTERM'd process exits non-zero, so letting it reach
			// onProcessEnd would shut the whole project down under
			// exit_on_end or exit_on_failure every time a file is saved.
			if proc.isBeingRestarted() {
				log.Debug().Msgf("Process %s exited for a restart; not ending the project", proc.getName())
			} else {
				p.onProcessEnd(exitCode, proc.procConf)
			}
		}
		// Only the instance that still owns the process name may deregister it
		// and report the remaining process count. A superseded instance must
		// not evict its replacement.
		if count, removed := p.removeRunningProcess(proc); removed {
			p.procCompleteChannel <- count
		}
	}(process)
}

// errWaitAborted is returned by waitIfNeeded when the waiting process was shut
// down (stopped, restarted or removed) before its dependencies were satisfied.
// It is not a failure of the process itself, so it must not be reported as a
// skipped process.
var errWaitAborted = errors.New("process was stopped while waiting for its dependencies")

func (p *ProjectRunner) waitIfNeeded(waiter *Process) error {
	process := waiter.procConf
	// Cancelled when the waiting process is shut down, which releases every
	// wait below instead of leaving this goroutine parked on a dependency that
	// may only resolve minutes later (or never).
	abort := waiter.procRunCtx.Done()
	for k := range process.DependsOn {
		if proc := p.getDoneOrRunningProcess(k); proc != nil {
			switch process.DependsOn[k].Condition {
			case types.ProcessConditionCompleted:
				if _, res := proc.waitForCompletionOrAbort(abort); res == waitAborted {
					return errWaitAborted
				}
			case types.ProcessConditionCompletedSuccessfully:
				log.Info().Msgf("%s is waiting for %s to complete successfully", process.ReplicaName, k)
				exitCode, res := proc.waitForCompletionOrAbort(abort)
				if res == waitAborted {
					return errWaitAborted
				}
				if !proc.procConf.IsExitCodeSuccess(exitCode) {
					return fmt.Errorf("process %s depended on %s to complete successfully, but it exited with status %d",
						process.ReplicaName, k, exitCode)
				}
			case types.ProcessConditionHealthy:
				if proc.procConf.ReadinessProbe == nil && proc.procConf.LivenessProbe == nil {
					return fmt.Errorf("health dependency defined in '%s' but no health check exists in '%s'", process.ReplicaName, k)
				}
				log.Info().Msgf("%s is waiting for %s to be healthy", process.ReplicaName, k)
				switch proc.waitUntilReady(abort) {
				case waitAborted:
					return errWaitAborted
				case waitFailed:
					return fmt.Errorf("process %s depended on %s to become ready, but it was terminated", process.ReplicaName, k)
				}
			case types.ProcessConditionLogReady:
				log.Info().Msgf("%s is waiting for %s log line %s", process.ReplicaName, k, proc.procConf.ReadyLogLine)
				switch proc.waitUntilLogReady(abort) {
				case waitAborted:
					return errWaitAborted
				case waitFailed:
					return fmt.Errorf("process %s depended on %s to become ready, but it was terminated", process.ReplicaName, k)
				}
			case types.ProcessConditionStarted:
				log.Info().Msgf("%s is waiting for %s to start", process.ReplicaName, k)
				proc.waitForStarted(abort)
			}
		} else {
			log.Error().Msgf("Error: process %s depends on %s, but it isn't running or completed", process.ReplicaName, k)
		}

	}
	// A stop that arrived while the last dependency was already satisfied must
	// still keep the process from starting.
	select {
	case <-abort:
		return errWaitAborted
	default:
	}
	return nil
}

func (p *ProjectRunner) onProcessEnd(exitCode int, procConf *types.ProcessConfig) {
	success := procConf.IsExitCodeSuccess(exitCode)
	if (!success && procConf.RestartPolicy.Restart == types.RestartPolicyExitOnFailure) ||
		procConf.RestartPolicy.ExitOnEnd {
		// A success_exit_codes match is equivalent to a clean exit, so the
		// project should not inherit the raw non-zero code (e.g. on exit_on_end).
		code := exitCode
		if success {
			code = 0
		}
		p.exitCodeMutex.Lock()
		p.exitCode = code
		p.exitCodeMutex.Unlock()
		log.Info().Msgf("Process %s exited with code %d. Shutting down project...", procConf.Name, exitCode)
		_ = p.ShutDownProject()
	}
}

func (p *ProjectRunner) onProcessSkipped(procConf *types.ProcessConfig) {
	if procConf.RestartPolicy.ExitOnSkipped {
		p.exitCodeMutex.Lock()
		p.exitCode = 1
		p.exitCodeMutex.Unlock()
		log.Info().Msgf("Process %s skipped. Shutting down project...", procConf.Name)
		_ = p.ShutDownProject()
	}
}

// newIncarnationState builds the state object for a fresh incarnation of a
// process. Previously the state was cloned from the outgoing instance, which
// made a new instance inherit a terminal status ("Terminating") and a stale
// process_end_time. Only the cumulative restart counter is carried over. The
// state is also registered as the canonical one for that name, so lookups
// performed while no instance is running observe the latest incarnation.
func (p *ProjectRunner) newIncarnationState(config *types.ProcessConfig) *types.ProcessState {
	state := types.NewProcessState(config)
	p.statesMutex.Lock()
	defer p.statesMutex.Unlock()
	if prev, ok := p.processStates[config.ReplicaName]; ok && prev != nil {
		state.Restarts = prev.Restarts
	}
	if p.pendingRestartReset[config.ReplicaName] {
		state.Restarts = 0
		delete(p.pendingRestartReset, config.ReplicaName)
	}
	p.processStates[config.ReplicaName] = state
	return state
}

// resetRestartCount arranges for the next incarnation of name to start with a
// zeroed restart counter, which newIncarnationState would otherwise carry
// forward. A restart the user asked for is a fresh start, not a continuation of
// a crash loop - without this, a process that had already exhausted
// max_restarts would come back with its budget still spent.
//
// The intent is recorded rather than applied: the stored ProcessState is the
// same object the outgoing Process is still updating under its own mutex, so
// writing to it here would race. newIncarnationState consumes the flag while
// building the new state, which nothing else can see yet.
func (p *ProjectRunner) resetRestartCount(name string) {
	p.statesMutex.Lock()
	defer p.statesMutex.Unlock()
	if p.pendingRestartReset == nil {
		p.pendingRestartReset = make(map[string]bool)
	}
	p.pendingRestartReset[name] = true
}

func (p *ProjectRunner) initProcessStates() {
	p.statesMutex.Lock()
	defer p.statesMutex.Unlock()
	p.processStates = make(map[string]*types.ProcessState)
	for name, proc := range p.project.Processes {
		p.processStates[name] = types.NewProcessState(&proc)
	}
}

func (p *ProjectRunner) initProcessLogs() {
	p.processLogs = make(map[string]*pclog.ProcessLogBuffer)
	for _, proc := range p.project.Processes {
		p.initProcessLog(proc.ReplicaName)
	}
}

func (p *ProjectRunner) initRestartCoalescing() {
	p.restartCalls = make(map[string]*RestartCall)
}

func (p *ProjectRunner) initProcessLog(name string) {
	p.processLogs[name] = pclog.NewLogBuffer(p.project.LogLength)
}

func (p *ProjectRunner) GetProcessState(name string) (*types.ProcessState, error) {
	var state *types.ProcessState
	proc := p.getRunningProcess(name)
	if proc != nil {
		state = proc.getState()
	} else {
		p.statesMutex.Lock()
		defer p.statesMutex.Unlock()
		var ok bool
		state, ok = p.processStates[name]
		if !ok {
			log.Error().Msgf("Error: process %s doesn't exist", name)
			return nil, fmt.Errorf("can't get state of process %s: no such process", name)
		}
	}
	// Add last activity time from log buffer
	if procLog, err := p.getProcessLog(name); err == nil {
		if t := procLog.GetLastWriteTime(); !t.IsZero() {
			state.LastActivityTime = &t
		}
	}

	// Add next run time for scheduled processes
	if s := p.processScheduler.Load(); s != nil {
		nextRun := s.GetNextRunTime(name)
		state.NextRunTime = nextRun
		if nextRun != nil {
			if !state.IsRunning {
				state.Status = types.ProcessStateScheduled
			}
		} else if state.Status == types.ProcessStateScheduled {
			// Restore to Completed if it was marked as Scheduled but no longer has a next run
			state.Status = types.ProcessStateCompleted
		}
	}

	// A process that exited cleanly but still has an armed watch is not
	// finished - it is waiting for a file to change. Reporting Watching also
	// explains why the project is still running, which "Completed" would leave
	// mysterious. IsWatchIdle is the same predicate DisplayProcessStatus uses,
	// so the API and the TUI cannot disagree; in particular a failed process
	// keeps its failure rather than being masked as Watching.
	state.IsWatched = p.isProcessWatched(name)
	state.WatchTriggerPath, state.WatchTriggerTime = p.lastWatchTrigger(name)
	if state.IsWatched && state.IsWatchIdle() {
		state.Status = types.ProcessStateWatching
	} else if !state.IsWatched && state.Status == types.ProcessStateWatching {
		state.Status = types.ProcessStateCompleted
	}
	return state, nil
}

func (p *ProjectRunner) getProcessStateData(name string, filter filterFn) error {
	proc := p.getRunningProcess(name)
	if proc != nil {
		proc.getStateData(filter)
	} else {
		p.statesMutex.Lock()
		defer p.statesMutex.Unlock()
		state, ok := p.processStates[name]
		if !ok {
			log.Error().Msgf("Error: process %s doesn't exist", name)
			return fmt.Errorf("can't get state of process %s: no such process", name)
		}
		filter(state)
		return nil
	}
	return nil
}

func (p *ProjectRunner) GetProcessesState() (*types.ProcessesState, error) {
	if p.withRecursiveMetrics {
		_ = p.processTree.Update()
	}
	states := &types.ProcessesState{
		States: make([]types.ProcessState, 0),
	}
	for name := range p.project.Processes {
		state, err := p.GetProcessState(name)
		if err != nil {
			return nil, err
		}
		states.States = append(states.States, *state)

	}
	return states, nil
}

func (p *ProjectRunner) getProcessesStateData(filter filterFn) error {
	for name := range p.project.Processes {
		err := p.getProcessStateData(name, filter)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *ProjectRunner) addRunningProcess(process *Process) {
	p.runProcMutex.Lock()
	p.runningProcesses[process.getName()] = process
	p.runProcMutex.Unlock()
}

// addDoneProcess records process as the last completed incarnation of its name.
// A stale instance that finishes after it was superseded is dropped, so that
// dependents resolving through getDoneOrRunningProcess never see an older,
// already cancelled object.
func (p *ProjectRunner) addDoneProcess(process *Process) {
	name := process.getName()
	if current := p.getRunningProcess(name); current != nil && current != process {
		log.Debug().Msgf("Not recording superseded instance of %s as done", name)
		return
	}
	p.doneProcMutex.Lock()
	p.doneProcesses[name] = process
	p.doneProcMutex.Unlock()
}

func (p *ProjectRunner) getRunningProcess(name string) *Process {
	p.runProcMutex.Lock()
	defer p.runProcMutex.Unlock()
	if runningProc, ok := p.runningProcesses[name]; ok {
		return runningProc
	}
	return nil
}

func (p *ProjectRunner) getDoneProcess(name string) *Process {
	p.doneProcMutex.Lock()
	defer p.doneProcMutex.Unlock()
	if doneProc, ok := p.doneProcesses[name]; ok {
		return doneProc
	}
	return nil
}

func (p *ProjectRunner) getDoneOrRunningProcess(name string) *Process {
	// Prefer the currently running process over a stale done entry.
	// After UpdateProcess / Restart replaces a process, the old object lingers
	// in doneProcesses with an already-cancelled procReadyCtx; returning it
	// would cause downstream waitUntilReady to spuriously fail ("aborted").
	if runningProc := p.getRunningProcess(name); runningProc != nil {
		return runningProc
	}
	return p.getDoneProcess(name)
}

// removeRunningProcess deregisters process, but only if it is still the
// instance registered under its name. A process can be superseded by a newer
// incarnation (restart, update, scale), and a stale instance finishing later
// must not evict the live one. It returns the number of remaining running
// processes and whether the removal took place.
func (p *ProjectRunner) removeRunningProcess(process *Process) (int, bool) {
	p.runProcMutex.Lock()
	defer p.runProcMutex.Unlock()
	name := process.getName()
	if current, ok := p.runningProcesses[name]; ok && current != process {
		log.Debug().Msgf("Not removing superseded instance of %s", name)
		return len(p.runningProcesses), false
	}
	delete(p.runningProcesses, name)
	return len(p.runningProcesses), true
}

func (p *ProjectRunner) StartProcess(name string) error {
	proc := p.getRunningProcess(name)
	if proc != nil {
		log.Error().Msgf("Process %s is already running", name)
		return fmt.Errorf("process %s is already running", name)
	}
	if processConfig, ok := p.project.Processes[name]; ok {
		p.runProcess(&processConfig)
		// Resume schedule if it was paused (e.g. initially disabled)
		if s := p.processScheduler.Load(); s != nil && s.IsScheduled(name) {
			if err := s.ResumeProcess(name); err != nil {
				log.Error().Err(err).Msgf("Failed to resume schedule for process %s", name)
			}
		}
		// Resume an existing registration - a stopped process's watch is paused,
		// not removed, so resuming it avoids re-walking the whole tree. A
		// process with no registration becomes watchable only now: it may have
		// been deferred (disabled, or excluded from an `up <process>...`
		// selection) when the watcher was first populated.
		if p.isProcessWatchRegistered(name) {
			p.watchResume(name)
		} else {
			p.watchAdd(&processConfig)
		}
	} else {
		return fmt.Errorf("no such process: %s", name)
	}

	return nil
}

func (p *ProjectRunner) StopProcess(name string) error {
	log.Info().Msgf("Stopping %s", name)
	proc := p.getRunningProcess(name)

	var err error
	if proc != nil {
		err = proc.shutDownNoRestart()
		if err != nil {
			log.Err(err).Msgf("failed to stop process %s", name)
		}
	} else {
		// A process that is not running may still be armed to start again on
		// its own - by a schedule, or by a file watch. Stopping it then means
		// disarming it, which is a perfectly sensible request and not an error:
		// a process reading Scheduled or Watching is exactly what the user is
		// looking at when they ask for it to stop.
		sched := p.processScheduler.Load()
		isScheduled := sched != nil && sched.IsScheduled(name)
		if !isScheduled && !p.isProcessWatched(name) {
			if _, ok := p.project.Processes[name]; !ok {
				log.Error().Msgf("Process %s does not exist", name)
				return fmt.Errorf("process %s does not exist", name)
			}
			log.Error().Msgf("Process %s is not running", name)
			return fmt.Errorf("process %s is not running", name)
		}
	}

	// A deliberately stopped process must not be brought back by a file change.
	p.watchPause(name)

	// Pause schedule if it was running or scheduled
	if sched := p.processScheduler.Load(); sched != nil && sched.IsScheduled(name) {
		if pauseErr := sched.PauseProcess(name); pauseErr != nil {
			log.Error().Err(pauseErr).Msgf("Failed to pause schedule for process %s", name)
			if err == nil {
				err = pauseErr
			}
		}
	}

	return err
}

func (p *ProjectRunner) SendSignal(name string, sig int) error {
	log.Info().Msgf("Sending signal %d to %s", sig, name)
	proc := p.getRunningProcess(name)
	if proc == nil {
		if _, ok := p.project.Processes[name]; !ok {
			log.Error().Msgf("Process %s does not exist", name)
			return fmt.Errorf("process %s does not exist", name)
		}
		log.Error().Msgf("Process %s is not running", name)
		return fmt.Errorf("process %s is not running", name)
	}
	return proc.sendSignal(sig)
}

func (p *ProjectRunner) StopProcesses(names []string) (map[string]string, error) {
	stopped := make(map[string]string)
	successes := 0
	for _, name := range names {
		if err := p.StopProcess(name); err == nil {
			stopped[name] = "ok"
			successes++
		} else {
			stopped[name] = err.Error()
		}
	}

	if successes != len(names) {
		if successes == 0 {
			return stopped, fmt.Errorf("no such processes or not running: %v", names)
		}
		return stopped, errors.New("failed to stop some processes")
	}
	return stopped, nil
}

// getNamespaceProcesses returns the namespace's processes in dependency order.
// Processes that are disabled - by `disabled: true` or by exclusion from an
// `up <process>...` selection - are included: a namespace operation is an
// explicit request, just like `process start <name>`, which also ignores the
// flag. Foreground processes are skipped, they can only run in the TUI.
// Only the namespace's own members are returned - a dependency belonging to
// another namespace is never started or stopped as a side effect.
func (p *ProjectRunner) getNamespaceProcesses(namespace string) ([]string, error) {
	if namespace == "" {
		namespace = types.DefaultNamespace
	}
	members := []string{}
	foreground := 0
	for name, proc := range p.project.Processes {
		if !proc.Namespace.Contains(namespace) {
			continue
		}
		if proc.IsForeground {
			foreground++
			continue
		}
		members = append(members, name)
	}
	if len(members) == 0 {
		if foreground > 0 {
			return nil, fmt.Errorf("namespace %s has only foreground processes, which are excluded from namespace operations", namespace)
		}
		return nil, fmt.Errorf("namespace %s not found (no processes assigned)", namespace)
	}
	// Map iteration order is random - sort to keep the dependency walk deterministic
	slices.Sort(members)
	isMember := make(map[string]bool, len(members))
	for _, name := range members {
		isMember[name] = true
	}

	// Walk the dependency closure to get the correct order for bulk operations,
	// then keep only the members - a dependency in another namespace can still
	// order two members relative to each other.
	nsProcs := []string{}
	err := p.project.WithProcesses(members, func(proc types.ProcessConfig) error {
		if isMember[proc.ReplicaName] {
			nsProcs = append(nsProcs, proc.ReplicaName)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nsProcs, nil
}

func (p *ProjectRunner) GetNamespaces() ([]string, error) {
	states, err := p.GetProcessesState()
	if err != nil {
		return nil, err
	}

	nsMap := make(map[string]struct{})
	for _, state := range states.States {
		for _, ns := range state.Namespace.OrDefault() {
			nsMap[ns] = struct{}{}
		}
	}

	namespaces := make([]string, 0, len(nsMap))
	for ns := range nsMap {
		namespaces = append(namespaces, ns)
	}
	slices.Sort(namespaces)

	return namespaces, nil
}

func (p *ProjectRunner) StartNamespace(namespace string) error {
	names, err := p.getNamespaceProcesses(namespace)
	if err != nil {
		return err
	}

	log.Info().Msgf("Starting namespace: %s", namespace)
	var errs []error
	for _, name := range names {
		// Check if already running to avoid error logs
		if p.getRunningProcess(name) == nil {
			if err := p.StartProcess(name); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to start namespace %s: %v", namespace, errs)
	}
	return nil
}

func (p *ProjectRunner) StopNamespace(namespace string) error {
	names, err := p.getNamespaceProcesses(namespace)
	if err != nil {
		return err
	}
	// Reverse order for stop
	slices.Reverse(names)

	log.Info().Msgf("Stopping namespace: %s", namespace)

	var errs []error
	for _, name := range names {
		if p.getRunningProcess(name) != nil {
			if err := p.StopProcess(name); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to stop namespace %s: %v", namespace, errs)
	}
	return nil
}

func (p *ProjectRunner) RestartNamespace(namespace string) error {
	// Determine processes first
	names, err := p.getNamespaceProcesses(namespace)
	if err != nil {
		return err
	}

	// Stop them (Reverse)
	slices.Reverse(names)
	stopErrs := []error{}
	for _, name := range names {
		if p.getRunningProcess(name) != nil {
			if err := p.StopProcess(name); err != nil {
				stopErrs = append(stopErrs, err)
			}
		}
	}

	// Wait for them to stop
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

waitingForStop:
	for {
		select {
		case <-timeout:
			log.Warn().Msgf("Timeout waiting for namespace %s to stop during restart", namespace)
			break waitingForStop
		case <-ticker.C:
			allStopped := true
			for _, name := range names {
				if p.getRunningProcess(name) != nil {
					allStopped = false
					break
				}
			}
			if allStopped {
				break waitingForStop
			}
		}
	}

	// Start them (Forward)
	slices.Reverse(names) // Restore order
	startErrs := []error{}
	for _, name := range names {
		if p.getRunningProcess(name) == nil {
			if err := p.StartProcess(name); err != nil {
				startErrs = append(startErrs, err)
			}
		}
	}

	if len(stopErrs) > 0 || len(startErrs) > 0 {
		return fmt.Errorf("errors during restart namespace %s. Stop: %v, Start: %v", namespace, stopErrs, startErrs)
	}
	return nil
}

// restartOpts tunes an individual restart. The zero value reproduces the
// historical behavior of RestartProcess exactly.
type restartOpts struct {
	// skipBackoff omits the inter-incarnation sleep. Backoff exists to damp a
	// crash loop; a restart the user asked for - directly or by saving a file -
	// is not a crash, and the one-second floor would otherwise make every
	// watch-triggered restart feel sluggish.
	skipBackoff bool
	// resetRestarts zeroes the cumulative restart counter. Without it a process
	// that had already exhausted max_restarts would come back with its
	// crash-loop budget still spent, so the user's fix-and-save would start it
	// once and then refuse to restart it again.
	resetRestarts bool
}

func (p *ProjectRunner) RestartProcess(name string) error {
	return p.restartProcessWithOpts(name, restartOpts{})
}

func (p *ProjectRunner) restartProcessWithOpts(name string, opts restartOpts) error {
	p.restartMutex.Lock()

	// Check if restart is already in progress
	if call, exists := p.restartCalls[name]; exists {
		// Join the existing restart operation
		p.restartMutex.Unlock()
		call.wg.Wait()
		return call.err
	}

	// Create new restart operation
	call := &RestartCall{}
	call.wg.Add(1)
	p.restartCalls[name] = call
	p.restartMutex.Unlock()

	// Perform the restart
	err := p.doRestart(name, opts)

	// Complete the operation and notify waiters
	call.err = err
	call.wg.Done()

	// Clean up
	p.restartMutex.Lock()
	delete(p.restartCalls, name)
	p.restartMutex.Unlock()

	return err
}

func (p *ProjectRunner) doRestart(name string, opts restartOpts) error {
	log.Debug().Msgf("Restarting %s", name)
	proc := p.getRunningProcess(name)
	if proc != nil {
		// Mark before shutting down: the exit is a consequence of this
		// restart, and onProcessEnd must not read it as the process ending.
		proc.markRestarting()
		err := proc.shutDownNoRestart()
		if err != nil {
			log.Err(err).Msgf("failed to stop process %s", name)
			return err
		}
		// A process that never started is still parked in waitIfNeeded. The
		// shutdown above releases it; wait for its goroutine to actually exit so
		// the replacement is the only incarnation in flight. Already started
		// processes are not waited for here - their shutdown honours the
		// configured shutdown parameters and can take arbitrarily long.
		if !proc.hasStarted() && !proc.waitUntilTerminated(pendingTerminationTimeout) {
			log.Warn().Msgf("process %s did not stop within %v, starting a new instance anyway", name, pendingTerminationTimeout)
		}
		if !opts.skipBackoff {
			time.Sleep(proc.getBackoff())
		}
	}

	if opts.resetRestarts {
		p.resetRestartCount(name)
	}

	if processConfig, ok := p.project.Processes[name]; ok {
		p.runProcess(&processConfig)
	} else {
		return fmt.Errorf("no such process: %s", name)
	}
	return nil
}

func (p *ProjectRunner) GetProcessInfo(name string) (*types.ProcessConfig, error) {
	p.runProcMutex.Lock()
	defer p.runProcMutex.Unlock()
	if processConfig, ok := p.project.Processes[name]; ok {
		return &processConfig, nil
	} else {
		return nil, fmt.Errorf("no such process: %s", name)
	}
}

func (p *ProjectRunner) SetProcessInfo(config *types.ProcessConfig) error {
	p.runProcMutex.Lock()
	defer p.runProcMutex.Unlock()
	if config.Name == "" {
		return fmt.Errorf("process name is required")
	}
	config.Namespace = config.Namespace.Normalized()
	p.project.Processes[config.Name] = *config
	return nil
}

func (p *ProjectRunner) GetProcessPorts(name string) (*types.ProcessPorts, error) {
	proc := p.getRunningProcess(name)
	if proc == nil {
		return nil, fmt.Errorf("can't get ports: process %s is not running", name)
	}

	ports := &types.ProcessPorts{
		Name:     name,
		TcpPorts: make([]uint16, 0),
		UdpPorts: make([]uint16, 0),
	}
	err := proc.getOpenPorts(ports)
	if err != nil {
		return nil, err
	}
	return ports, nil
}

func (p *ProjectRunner) SetProcessPassword(name, pass string) error {
	p.runProcMutex.Lock()
	var elevatedProcs []*Process
	for _, process := range p.runningProcesses {
		if process.procConf.IsElevated && !process.passProvided {
			elevatedProcs = append(elevatedProcs, process)
		}
	}
	p.runProcMutex.Unlock()

	var wg sync.WaitGroup
	for _, process := range elevatedProcs {
		wg.Add(1)
		go func(process *Process) {
			defer wg.Done()
			err := process.setPassword(pass)
			if err != nil {
				log.Err(err).Msgf("failed to set password for elevated process %s", process.getName())
			}
		}(process)
	}
	wg.Wait()

	for _, process := range elevatedProcs {
		if process.passProvided {
			return nil
		}
	}
	return errors.New("password not accepted")
}

func (p *ProjectRunner) runningProcessesReverseDependencies() map[string]map[string]*Process {
	reverseDependencies := make(map[string]map[string]*Process)

	p.runProcMutex.Lock()
	defer p.runProcMutex.Unlock()
	for _, process := range p.runningProcesses {
		for k := range process.procConf.DependsOn {
			if runningProc, ok := p.runningProcesses[k]; ok {
				if _, ok := reverseDependencies[runningProc.getName()]; !ok {
					dep := make(map[string]*Process)
					reverseDependencies[runningProc.getName()] = dep
				}
				reverseDependencies[runningProc.getName()][process.getName()] = process
			} else {
				continue
			}
		}
	}

	return reverseDependencies
}

func (p *ProjectRunner) shutDownInOrder(wg *sync.WaitGroup, shutdownOrder []*Process) {
	reverseDependencies := p.runningProcessesReverseDependencies()
	for _, process := range shutdownOrder {
		wg.Add(1)
		go func(proc *Process) {
			defer wg.Done()
			waitForDepsWg := sync.WaitGroup{}
			if revDeps, ok := reverseDependencies[proc.getName()]; ok {
				for _, runningProc := range revDeps {
					waitForDepsWg.Add(1)
					go func(pr *Process) {
						pr.waitForCompletion()
						waitForDepsWg.Done()
					}(runningProc)
				}
			}
			waitForDepsWg.Wait()
			log.Debug().Msgf("[%s]: waited for all dependencies to shut down", proc.getName())

			err := proc.shutDown()
			if err != nil {
				log.Err(err).Msgf("failed to shutdown %s", proc.getName())
				return
			}
			proc.waitForCompletion()
		}(process)
	}
}

func (p *ProjectRunner) shutDownAndWait(shutdownOrder []*Process) {
	wg := sync.WaitGroup{}
	if p.isOrderedShutdown {
		p.shutDownInOrder(&wg, shutdownOrder)
	} else {
		for _, proc := range shutdownOrder {
			err := proc.shutDown()
			if err != nil {
				log.Err(err).Msgf("failed to shutdown %s", proc.getName())
				continue
			}
			wg.Add(1)
			go func(pr *Process) {
				pr.waitForCompletion()
				wg.Done()
			}(proc)
		}
	}

	wg.Wait()
}

func (p *ProjectRunner) ShutDownProject() error {
	// Stop watching before anything is torn down: a file touched by a
	// shutdown.command must not trigger a restart of a process that is already
	// on its way out.
	p.stopWatcher()

	p.runProcMutex.Lock()
	shutdownOrder := []*Process{}
	if p.isOrderedShutdown {
		err := p.project.WithProcesses([]string{}, func(process types.ProcessConfig) error {
			if runningProc, ok := p.runningProcesses[process.ReplicaName]; ok {
				shutdownOrder = append(shutdownOrder, runningProc)
			}
			return nil
		})
		if err != nil {
			log.Error().Msgf("Failed to build project run order: %s", err.Error())
		}
		slices.Reverse(shutdownOrder)
	} else {
		for _, proc := range p.runningProcesses {
			shutdownOrder = append(shutdownOrder, proc)
		}
	}
	p.runProcMutex.Unlock()

	var nameOrder []string
	for _, v := range shutdownOrder {
		nameOrder = append(nameOrder, v.getName())
	}
	log.Debug().Msgf("Shutting down %d processes. Order: %q", len(shutdownOrder), nameOrder)
	for _, proc := range shutdownOrder {
		proc.prepareForShutDown()
	}

	p.shutDownAndWait(shutdownOrder)
	p.cancelAppFn()
	return nil
}

func (p *ProjectRunner) WaitForProjectShutdown() {
	if p.ctxApp != nil {
		if !p.isTuiOn {
			fmt.Println("Project Completed. Press Ctrl+C to quit")
		}
		<-p.ctxApp.Done()
	}
}

func (p *ProjectRunner) IsRemote() bool {
	return false
}

func (p *ProjectRunner) ErrorForSecs() int {
	return 0
}

func (p *ProjectRunner) GetProjectName() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Err(err).Msg("Failed get hostname")
		hostname = "unknown"
	}

	name := p.project.Name
	if name == "" {
		name, err = os.Getwd()
		if err != nil {
			log.Err(err).Msg("Failed get CWD")
			name = "unknown"
		}
	}

	return fmt.Sprintf("%s/%s", hostname, path.Base(name)), nil
}

func (p *ProjectRunner) getProcessLog(name string) (*pclog.ProcessLogBuffer, error) {
	if procLogs, ok := p.processLogs[name]; ok {
		return procLogs, nil
	}
	log.Error().Msgf("process %s doesn't exist", name)
	return nil, fmt.Errorf("process %s doesn't exist", name)
}

func (p *ProjectRunner) GetProcessLog(name string, offsetFromEnd, limit int) ([]string, error) {
	logs, err := p.getProcessLog(name)
	if err != nil {
		return nil, err
	}
	return logs.GetLogRange(offsetFromEnd, limit), nil
}

func (p *ProjectRunner) GetProcessLogLength(name string) int {
	logs, err := p.getProcessLog(name)
	if err != nil {
		return 0
	}
	return logs.GetLogLength()
}

func (p *ProjectRunner) GetLogsAndSubscribe(name string, observer pclog.LogObserver) error {
	logs, err := p.getProcessLog(name)
	if err != nil {
		log.Err(err).Msgf("can't subscribe to process %s", name)
		return err
	}
	logs.GetLogsAndSubscribe(observer)
	return nil
}

func (p *ProjectRunner) UnSubscribeLogger(name string, observer pclog.LogObserver) error {
	logs, err := p.getProcessLog(name)
	if err != nil {
		return err
	}
	logs.UnSubscribe(observer)
	return nil
}

func (p *ProjectRunner) TruncateProcessLogs(name string) error {
	logs, err := p.getProcessLog(name)
	if err != nil {
		return err
	}
	logs.Truncate()
	return nil
}

func (p *ProjectRunner) ScaleProcess(name string, scale int) error {
	if scale < 1 {
		err := fmt.Errorf("cannot scale process %s to a negative or zero value %d", name, scale)
		log.Err(err).Msg("scale failed")
		return err
	}
	if processConfig, ok := p.project.Processes[name]; ok {
		origScale := p.getCurrentReplicaCount(processConfig.Name)
		scaleDelta := scale - origScale
		if scaleDelta < 0 {
			log.Info().Msgf("scaling down %s by %d", name, -scaleDelta)
			p.scaleDownProcess(processConfig.Name, scale)
		} else if scaleDelta > 0 {
			log.Info().Msgf("scaling up %s by %d", name, scaleDelta)
			p.scaleUpProcess(processConfig, scaleDelta, scale, origScale)
		} else {
			log.Info().Msgf("no change in scale of %s", name)
			return nil
		}
		p.updateReplicaCount(processConfig.Name, scale)
	} else {
		return fmt.Errorf("no such process: %s", name)
	}
	return nil
}

func (p *ProjectRunner) getCurrentReplicaCount(name string) int {
	counter := 0
	for _, proc := range p.project.Processes {
		if proc.Name == name {
			counter++
		}
	}
	return counter
}

func (p *ProjectRunner) scaleUpProcess(proc types.ProcessConfig, toAdd, scale, origScale int) {
	for i := range toAdd {
		var procFromConf types.ProcessConfig
		err := json.Unmarshal([]byte(proc.OriginalConfig), &procFromConf)
		if err != nil {
			log.Err(err).Msgf("failed to unmarshal config for %s", proc.Name)
			return
		}
		procFromConf.ReplicaNum = origScale + i
		procFromConf.Replicas = scale
		procFromConf.ReplicaName = procFromConf.CalculateReplicaName()
		tpl := templater.New(p.project.Vars)
		tpl.RenderProcess(&procFromConf)
		procFromConf.AssignProcessExecutableAndArgs(p.project.ShellConfig, p.project.GetElevatedShellArg())
		p.addProcessAndRun(procFromConf)
	}
}

func (p *ProjectRunner) scaleDownProcess(name string, scale int) {
	toRemove := []string{}
	p.procConfMutex.Lock()
	for _, proc := range p.project.Processes {
		if proc.Name == name {
			if proc.ReplicaNum >= scale {
				toRemove = append(toRemove, proc.ReplicaName)
			} else {
				proc.Replicas = scale
				p.project.Processes[proc.ReplicaName] = proc
			}
		}
	}
	p.procConfMutex.Unlock()

	wg := sync.WaitGroup{}
	for _, name := range toRemove {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := p.removeProcess(name); err != nil {
				log.Err(err).Msgf("failed to scale down process %s", name)
			}
		}(name)
	}
	wg.Wait()
}

func (p *ProjectRunner) updateReplicaCount(name string, scale int) {
	for _, proc := range p.project.Processes {
		if proc.Name == name {
			proc.Replicas = scale
			p.project.Processes[proc.ReplicaName] = proc
			if proc.ReplicaName != proc.CalculateReplicaName() {
				p.renameProcess(proc.ReplicaName, proc.CalculateReplicaName())
			}
		}
	}
}

func (p *ProjectRunner) renameProcess(name string, newName string) {
	process := p.getRunningProcess(name)
	if process != nil {
		_, _ = p.removeRunningProcess(process)
		process.setName(newName)
		p.addRunningProcess(process)
	}
	logs := p.removeProcessLogs(name)
	if logs != nil {
		p.processLogs[newName] = logs
	}
	state, err := p.GetProcessState(name)
	if err == nil {
		p.statesMutex.Lock()
		defer p.statesMutex.Unlock()
		delete(p.processStates, name)
		state.Name = newName
		p.processStates[newName] = state
	}
	procConf, ok := p.project.Processes[name]
	if ok {
		delete(p.project.Processes, name)
		procConf.ReplicaName = newName
		p.project.Processes[newName] = procConf
	}
	// The watcher is keyed by replica name, and scaling down to 1 renames
	// e.g. api-0 to api. Re-key it here, alongside the logs and state above,
	// or the watch would be stranded under a name nothing looks up.
	p.watchRemove(name)
	if ok {
		p.watchAdd(&procConf)
	}
}
func (p *ProjectRunner) removeProcessLogs(name string) *pclog.ProcessLogBuffer {
	p.logsMutex.Lock()
	defer p.logsMutex.Unlock()
	logs, ok := p.processLogs[name]
	if ok {
		logs.Close()
		delete(p.processLogs, name)
	}
	return logs
}

func (p *ProjectRunner) removeProcess(name string) error {
	p.watchRemove(name)
	p.removeProcessLogs(name)
	p.procConfMutex.Lock()
	delete(p.project.Processes, name)
	p.procConfMutex.Unlock()
	running := p.getRunningProcess(name)
	if running != nil {
		err := running.shutDownNoRestart()
		if err != nil {
			log.Err(err).Msgf("failed to remove process %s", name)
			return err
		} else {
			running.waitForCompletion()
		}
	}
	return nil
}

func (p *ProjectRunner) addProcessAndRun(proc types.ProcessConfig) {
	p.statesMutex.Lock()
	p.processStates[proc.ReplicaName] = types.NewProcessState(&proc)
	p.statesMutex.Unlock()
	p.procConfMutex.Lock()
	p.project.Processes[proc.ReplicaName] = proc
	p.procConfMutex.Unlock()
	// Drop any stale done entry left by a previous incarnation; otherwise
	// new dependents would still see the old, cancelled Process object.
	p.doneProcMutex.Lock()
	delete(p.doneProcesses, proc.ReplicaName)
	p.doneProcMutex.Unlock()
	p.initProcessLog(proc.ReplicaName)
	if !proc.IsDeferred() {
		p.runProcess(&proc)
	}
	// UpdateProject routes every add, update and scale-up through here, so this
	// single hook keeps the watcher reconciled across reloads.
	p.watchAdd(&proc)
}

func selectRunningProcesses(project *types.Project, procList []string) error {
	if len(procList) == 0 {
		return nil
	}
	newProcMap := types.Processes{}
	err := project.WithProcesses(procList, func(process types.ProcessConfig) error {
		if process.IsForeground {
			return nil
		}
		newProcMap[process.ReplicaName] = process
		return nil
	})
	if err != nil {
		log.Err(err).Msgf("Failed select processes")
		return err
	}
	for name, proc := range project.Processes {
		if _, ok := newProcMap[name]; !ok {
			proc.Disabled = true
		} else {
			proc.Disabled = false
		}
		project.Processes[name] = proc
	}
	return nil
}

func selectRunningProcessesNoDeps(project *types.Project, procList []string) error {
	if len(procList) == 0 {
		return nil
	}
	for name, proc := range project.Processes {
		found := slices.Contains(procList, proc.Name)
		if !found {
			proc.Disabled = true
		} else {
			proc.DependsOn = types.DependsOnConfig{}
			proc.Disabled = false
		}
		project.Processes[name] = proc
	}

	return nil
}

// applySelection narrows project to the `up <process>...` selection by
// disabling everything that wasn't selected.
func (p *ProjectRunner) applySelection(project *types.Project) error {
	if p.noDeps {
		return selectRunningProcessesNoDeps(project, p.processesToRun)
	}
	return selectRunningProcesses(project, p.processesToRun)
}

// reapplySelection re-applies the `up <process>...` selection to a freshly
// loaded project, so that a reload or an update doesn't resurrect - and start -
// the processes the user left out. Unlike the initial selection, a name that
// the new configuration no longer defines is dropped instead of failing the
// whole update: that process is about to be removed from the project anyway.
func (p *ProjectRunner) reapplySelection(project *types.Project) error {
	if len(p.processesToRun) == 0 {
		return nil
	}
	selected := make([]string, 0, len(p.processesToRun))
	for _, name := range p.processesToRun {
		if _, err := project.GetProcesses(name); err != nil {
			log.Info().Msgf("Selected process %s is no longer defined, dropping it from the selection", name)
			continue
		}
		selected = append(selected, name)
	}
	if len(selected) == 0 {
		// Nothing of the original selection survived. Keep the remaining
		// processes deferred rather than starting the entire project.
		for name, proc := range project.Processes {
			proc.Disabled = true
			project.Processes[name] = proc
		}
		return nil
	}
	if p.noDeps {
		return selectRunningProcessesNoDeps(project, selected)
	}
	return selectRunningProcesses(project, selected)
}

func (p *ProjectRunner) GetLogLength() int {
	return p.project.LogLength
}

// GetDependenciesOrderNames used for testing
func (p *ProjectRunner) GetDependenciesOrderNames() ([]string, error) {
	return p.project.GetDependenciesOrderNames()
}

func (p *ProjectRunner) GetProjectState(checkMem bool) (*types.ProjectState, error) {
	runningProcesses := 0
	for name := range p.project.Processes {
		state, err := p.GetProcessState(name)
		if err != nil {
			return nil, err
		}
		if state.IsRunning {
			runningProcesses++
		}
	}
	p.projectState.RunningProcessNum = runningProcesses
	p.projectState.UpTime = time.Since(p.projectState.StartTime)
	if checkMem {
		p.projectState.MemoryState = getMemoryUsage()
	}
	return p.projectState, nil
}

func getMemoryUsage() *types.MemoryState {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	return &types.MemoryState{
		Allocated:      bToMb(m.Alloc),
		TotalAllocated: bToMb(m.TotalAlloc),
		SystemMemory:   bToMb(m.Sys),
		GcCycles:       m.NumGC,
	}
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func NewProjectRunner(opts *ProjectOpts) (*ProjectRunner, error) {
	current, err := user.Current()
	username := "unknown-user"
	if err != nil {
		log.Err(err).Msg("Failed get user")
	} else {
		username = current.Username
	}
	runner := &ProjectRunner{
		project:              opts.project,
		admitters:            opts.admitters,
		processesToRun:       opts.processesToRun,
		noDeps:               opts.noDeps,
		mainProcess:          opts.mainProcess,
		mainProcessArgs:      opts.mainProcessArgs,
		isTuiOn:              opts.isTuiOn,
		isOrderedShutdown:    opts.isOrderedShutdown,
		disableDotenv:        opts.disableDotenv,
		truncateLogs:         opts.truncateLogs,
		refRate:              opts.refRate,
		withRecursiveMetrics: opts.withRecursiveMetrics,
		noWatch:              opts.noWatch,
		projectState: &types.ProjectState{
			FileNames: opts.project.FileNames,
			StartTime: time.Now(),
			UserName:  username,
			Version:   config.Version,
		},
		procCompleteChannel: make(chan int, 128),
	}

	name, err := runner.GetProjectName()
	if err != nil {
		log.Err(err).Msg("Failed get project name")
	} else {
		runner.projectState.ProjectName = name
	}

	if err = runner.applySelection(runner.project); err != nil {
		return nil, err
	}
	runner.projectState.ProcessNum = len(runner.project.Processes)
	runner.init()
	runner.ctxApp, runner.cancelAppFn = context.WithCancel(context.Background())
	return runner, nil
}

// beginUpdate marks a project or process update as in flight. An update stops
// and restarts processes, so while one is running the project must not mistake
// a momentarily empty running set for "all processes are done". Returns the
// function that ends the update.
func (p *ProjectRunner) beginUpdate() func() {
	p.updatesInFlight.Add(1)
	return func() {
		if p.updatesInFlight.Add(-1) != 0 {
			return
		}
		// The update may have removed the last running process for good -
		// re-trigger the completion check that was suppressed while it ran.
		p.runProcMutex.Lock()
		running := len(p.runningProcesses)
		p.runProcMutex.Unlock()
		if running == 0 {
			select {
			case p.procCompleteChannel <- 0:
			default:
			}
		}
	}
}

func (p *ProjectRunner) UpdateProject(project *types.Project) (map[string]string, error) {
	defer p.beginUpdate()()
	// Re-apply the load-time admission policies (e.g. --namespace) and the
	// `up <process>...` selection so that excluded processes don't get
	// resurrected - and started - by a project reload or update.
	admitter.ApplyToProject(project, p.admitters)
	if err := p.reapplySelection(project); err != nil {
		return nil, err
	}
	newProcs := make(map[string]types.ProcessConfig)
	delProcs := make(map[string]types.ProcessConfig)
	updatedProcs := make(map[string]types.ProcessConfig)
	for name, newProc := range project.Processes {
		if currentProc, ok := p.project.Processes[name]; ok {
			equal := currentProc.Compare(&newProc)
			if equal {
				log.Debug().Msgf("Process %s is up to date", name)
				continue
			}
			log.Debug().Msgf("Process %s is updated", name)
			updatedProcs[name] = newProc
		} else {
			log.Debug().Msgf("Process %s is new", name)
			newProcs[name] = newProc
		}
	}
	for name, currentProc := range p.project.Processes {
		if _, ok := project.Processes[name]; !ok {
			log.Debug().Msgf("Process %s is deleted", name)
			delProcs[name] = currentProc
		}
	}
	status := make(map[string]string)
	errs := make([]error, 0)
	//Delete removed processes
	for name := range delProcs {
		err := p.removeProcess(name)
		if err != nil {
			log.Err(err).Msgf("Failed to remove process %s", name)
			errs = append(errs, err)
			status[name] = types.ProcessUpdateError
			continue
		}
		status[name] = types.ProcessUpdateRemoved
	}
	//Add new processes
	for name, proc := range newProcs {
		p.addProcessAndRun(proc)
		status[name] = types.ProcessUpdateAdded
	}
	//Update processes
	for name, proc := range updatedProcs {
		err := p.UpdateProcess(&proc)
		if err != nil {
			log.Err(err).Msgf("Failed to update process %s", name)
			errs = append(errs, err)
			status[name] = types.ProcessUpdateError
			continue
		}
		status[name] = types.ProcessUpdateUpdated
	}
	return status, errors.Join(errs...)
}

func (p *ProjectRunner) ReloadProject() (map[string]string, error) {
	opts := &loader.LoaderOptions{
		FileNames:        p.project.FileNames,
		EnvFileNames:     p.project.EnvFileNames,
		IsInternalLoader: true,
	}
	opts.WithTuiDisabled(p.disableDotenv)
	opts.WithTuiDisabled(p.isTuiOn)
	project, err := loader.Load(opts)
	if err != nil {
		log.Err(err).Msg("Failed to load project")
		return nil, err
	}
	status, err := p.UpdateProject(project)
	if err != nil {
		log.Err(err).Msg("Failed to update project")
		return nil, err
	}
	return status, nil
}
func (p *ProjectRunner) UpdateProcess(updated *types.ProcessConfig) error {
	defer p.beginUpdate()()
	isScaleChanged := false
	validateProbes(updated.LivenessProbe)
	validateProbes(updated.ReadinessProbe)
	updated.AssignProcessExecutableAndArgs(p.project.ShellConfig, p.project.ShellConfig.ElevatedShellArg)
	if currentProc, ok := p.project.Processes[updated.ReplicaName]; ok {
		equal := currentProc.Compare(updated)
		if equal {
			log.Debug().Msgf("Process %s is up to date", updated.Name)
			return nil
		}
		log.Debug().Msgf("Process %s is updated", updated.Name)
		if currentProc.Replicas != updated.Replicas {
			isScaleChanged = true
		}
	} else {
		err := fmt.Errorf("no such process: %s", updated.ReplicaName)
		log.Err(err).Msgf("Failed to update process %s", updated.ReplicaName)
		return err
	}

	err := p.removeProcess(updated.ReplicaName)
	if err != nil {
		log.Err(err).Msgf("Failed to remove process %s", updated.ReplicaName)
		return err
	}
	p.addProcessAndRun(*updated)

	if isScaleChanged {
		err = p.ScaleProcess(updated.ReplicaName, updated.Replicas)
		if err != nil {
			log.Err(err).Msgf("Failed to scale process %s", updated.Name)
			return err
		}
	}
	return nil
}

func (p *ProjectRunner) prepareEnvCmds() {
	for env, cmd := range p.project.EnvCommands {
		output, err := runCmd(cmd)
		if err != nil {
			log.Err(err).Msgf("Failed to run Env command %s for %s variable", cmd, env)
			continue
		}
		if p.project.Environment == nil {
			p.project.Environment = make(types.Environment, 0)
		}
		p.project.Environment = append(p.project.Environment, fmt.Sprintf("%s=%s", env, output))
		log.Debug().Msgf("Env variable %s set to %s", env, output)
	}
}

func runCmd(envCmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := command.BuildCommandContext(ctx, envCmd)
	out, err := cmd.Output()
	if err != nil {
		log.Err(err).Msgf("Failed to run Env command %s", envCmd)
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func validateProbes(probe *health.Probe) {
	if probe != nil {
		probe.ValidateAndSetDefaults()
	}
}
func (p *ProjectRunner) GetProcessPty(name string) *os.File {
	proc := p.getRunningProcess(name)
	if proc == nil {
		return nil
	}
	return proc.GetPty()
}

// SendProcessKeys writes the given keys to a running interactive process's stdin.
func (p *ProjectRunner) SendProcessKeys(name string, keys string) error {
	proc := p.getRunningProcess(name)
	if proc == nil {
		if _, ok := p.project.Processes[name]; !ok {
			return fmt.Errorf("process %s does not exist", name)
		}
		return fmt.Errorf("process %s is not running", name)
	}
	return proc.sendKeys(keys)
}

func (p *ProjectRunner) GetFullProcessEnvironment(proc *types.ProcessConfig) []string {
	var dotEnvVars map[string]string
	if !p.disableDotenv {
		dotEnvVars = p.project.DotEnvVars
	}
	return buildProcessEnvironment(proc, p.project.Environment, dotEnvVars)
}

// GetDependencyGraph builds and returns the process dependency graph with current status
func (p *ProjectRunner) GetDependencyGraph() (*types.DependencyGraph, error) {
	graph := types.BuildDependencyGraph(p.project.Processes)

	// Enrich with runtime status
	for name, node := range graph.AllNodes {
		if state, err := p.GetProcessState(name); err == nil {
			node.Status = state.Status
			node.IsReady = state.Health
		}
	}
	return graph, nil
}
