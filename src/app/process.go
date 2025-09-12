package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cakturk/go-netstat/netstat"
	"github.com/f1bonacc1/process-compose/src/types"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/pclog"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	puproc "github.com/shirou/gopsutil/v4/process"
)

const (
	UndefinedShutdownTimeoutSec = 0
	DefaultShutdownTimeoutSec   = 10
	EnvReplicaNum               = "PC_REPLICA_NUM"
	LogReplicaNum               = "{" + EnvReplicaNum + "}"
)

type Process struct {
	sync.Mutex
	globalEnv            []string
	confMtx              sync.Mutex
	procConf             *types.ProcessConfig
	procState            *types.ProcessState
	stateMtx             sync.Mutex
	procCond             sync.Cond
	procStartedChan      chan struct{}
	procStateChan        chan string
	procReadyCtx         context.Context
	readyCancelFn        context.CancelFunc
	procLogReadyCtx      context.Context
	readyLogCancelFn     context.CancelCauseFunc
	procRunCtx           context.Context
	runCancelFn          context.CancelFunc
	waitForPassCtx       context.Context
	waitForPassCancelFn  context.CancelFunc
	mtxStopFn            sync.Mutex
	waitForStoppedCtx    context.Context
	waitForStoppedFn     context.CancelFunc
	procColor            func(a ...interface{}) string
	noColor              func(a ...interface{}) string
	redColor             func(a ...interface{}) string
	logBuffer            *pclog.ProcessLogBuffer
	logger               pclog.PcLogger
	command              command.Commander
	started              bool
	done                 bool
	timeMutex            sync.Mutex
	startTime            time.Time
	liveProber           *health.Prober
	readyProber          *health.Prober
	shellConfig          command.ShellConfig
	printLogs            bool
	isMain               bool
	extraArgs            []string
	isStopped            atomic.Bool
	stdin                io.WriteCloser
	passProvided         bool
	isTuiEnabled         bool
	stdOutDone           chan struct{}
	stdErrDone           chan struct{}
	dotEnvVars           map[string]string
	truncateLogs         bool
	metricsProc          *puproc.Process
	lastStatusPoll       time.Time
	refRate              time.Duration
	withRecursiveMetrics bool
}

func NewProcess(opts ...ProcOpts) *Process {
	proc := &Process{
		redColor:        color.New(color.FgHiRed).SprintFunc(),
		noColor:         color.New(color.Reset).SprintFunc(),
		started:         false,
		done:            false,
		procStateChan:   make(chan string, 1),
		procStartedChan: make(chan struct{}, 1),
	}

	for _, opt := range opts {
		opt(proc)
	}
	proc.procColor = pclog.Name2Color(proc.getName())

	proc.procReadyCtx, proc.readyCancelFn = context.WithCancel(context.Background())
	proc.procLogReadyCtx, proc.readyLogCancelFn = context.WithCancelCause(context.Background())
	proc.procRunCtx, proc.runCancelFn = context.WithCancel(context.Background())
	proc.setUpProbes()
	proc.procCond = *sync.NewCond(proc)
	return proc
}

func (p *Process) run() int {
	if p.isState(types.ProcessStateTerminating) {
		return 0
	}

	if err := p.validateProcess(); err != nil {
		log.Error().Err(err).Msgf(`Failed to run command ["%v"] for process %s`, strings.Join(p.getCommand(), `" "`), p.getName())
		p.onProcessEnd(types.ProcessStateError)
		return 1
	}

	p.onProcessStart()
loop:
	for {
		err := p.setStateAndRun(p.getStartingStateName(), p.getProcessStarter())
		if err != nil {
			log.Error().Err(err).Msgf(`Failed to run command ["%v"] for process %s`, strings.Join(p.getCommand(), `" "`), p.getName())
			p.logBuffer.Write(err.Error())
			p.onProcessEnd(types.ProcessStateError)
			return 1
		}

		p.setStartTime(time.Now())
		p.stateMtx.Lock()
		p.procState.Pid = p.command.Pid()
		p.metricsProc, err = puproc.NewProcess(int32(p.procState.Pid))
		if err != nil {
			log.Err(err).Msgf("Could not find pid %d with name %s", p.procState.Pid, p.getName())
		}
		p.stateMtx.Unlock()
		log.Info().
			Str("process", p.getName()).
			Strs("command", p.getCommand()).
			Msg("Started")

		p.startProbes()

		p.waitForStdOutErr()
		_ = p.command.Wait()
		p.Lock()
		p.setExitCode(p.command.ExitCode())
		p.Unlock()
		log.Info().
			Str("process", p.getName()).
			Int("exit_code", p.getExitCode()).
			Msg("Exited")

		if p.isDaemonLaunched() {
			p.setState(types.ProcessStateLaunched)
			p.waitForDaemonCompletion()
		}

		if !p.isRestartable() {
			break
		}
		p.setState(types.ProcessStateRestarting)
		p.procState.Restarts += 1
		log.Info().Msgf("Restarting %s in %v second(s)... Restarts: %d",
			p.getName(), p.getBackoff().Seconds(), p.procState.Restarts)

		select {
		case <-p.procRunCtx.Done():
			log.Debug().Str("process", p.getName()).Msg("process stopped while waiting to restart")
			break loop
		case <-time.After(p.getBackoff()):
			p.handleInfo("\n")
			continue
		}
	}
	p.onProcessEnd(types.ProcessStateCompleted)
	return p.getExitCode()
}

func (p *Process) waitForStdOutErr() {
	ctx, cancel := context.WithCancel(context.Background())
	if p.procConf.IsDaemon {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(p.procConf.LaunchTimeout)*time.Second)
	}
	defer cancel()
	if p.stdOutDone != nil {
		select {
		case <-ctx.Done():
			log.Debug().Msgf("%s stdout done with timeout", p.getName())
			return
		case <-p.stdOutDone:
			log.Debug().Msgf("%s stdout done", p.getName())
		}
		p.stdOutDone = nil
	}
	if p.stdErrDone != nil {
		select {
		case <-ctx.Done():
			log.Debug().Msgf("%s stderr done with timeout", p.getName())
			return
		case <-p.stdErrDone:
			log.Debug().Msgf("%s stderr done", p.getName())
		}
		p.stdErrDone = nil
	}
}

func (p *Process) getProcessStarter() func() error {
	return func() error {
		p.command = p.getCommander()
		p.command.SetEnv(p.getProcessEnvironment())
		p.command.SetDir(p.procConf.WorkingDir)

		if p.isMain || (p.procConf.IsElevated && !p.isTuiEnabled) {
			p.command.AttachIo()
		} else {
			p.command.SetCmdArgs()
			if p.truncateLogs {
				p.logBuffer.Truncate()
			}
			stdout, _ := p.command.StdoutPipe()
			p.stdOutDone = make(chan struct{})
			go p.handleOutput(stdout, "stdout", p.handleInfo, p.stdOutDone)
			if !p.procConf.IsTty {
				stderr, _ := p.command.StderrPipe()
				p.stdErrDone = make(chan struct{})
				go p.handleOutput(stderr, "stderr", p.handleError, p.stdErrDone)
			}
		}

		if p.procConf.IsElevated && p.isTuiEnabled {
			stdin, err := p.command.StdinPipe()
			if err != nil {
				log.Error().Err(err).Msg("Failed to get stdin pipe")
			}
			p.stdin = stdin
		}

		return p.command.Start()
	}
}

func (p *Process) getCommander() command.Commander {
	if p.procConf.IsTty && !p.isMain {
		return command.BuildPtyCommand(
			p.procConf.Executable,
			p.mergeExtraArgs(),
		)
	} else {
		return command.BuildCommand(
			p.procConf.Executable,
			p.mergeExtraArgs(),
		)
	}

}

func (p *Process) mergeExtraArgs() []string {
	if len(p.extraArgs) == 0 {
		return p.procConf.Args
	}
	tmp := make([]string, len(p.procConf.Args))
	copy(tmp, p.procConf.Args)
	if isStringDefined(p.procConf.Command) {
		lastArg := p.procConf.Args[len(p.procConf.Args)-1]
		lastArg += " " + strings.Join(p.extraArgs, " ")
		return append(tmp[:len(tmp)-1], lastArg)
	} else if len(p.procConf.Entrypoint) > 0 {
		return append(tmp, p.extraArgs...)
	}
	return p.procConf.Args
}

func (p *Process) getBackoff() time.Duration {
	backoff := 1
	if p.procConf.RestartPolicy.BackoffSeconds > backoff {
		backoff = p.procConf.RestartPolicy.BackoffSeconds
	}
	return time.Duration(backoff) * time.Second
}

func (p *Process) getProcessEnvironment() []string {
	env := []string{
		"PC_PROC_NAME=" + p.procConf.Name,
		EnvReplicaNum + "=" + strconv.Itoa(p.procConf.ReplicaNum),
	}
	env = append(env, os.Environ()...)
	env = append(env, p.globalEnv...)
	env = append(env, p.procConf.Environment...)
	if p.dotEnvVars != nil && !p.procConf.DisableDotEnv {
		for k, v := range p.dotEnvVars {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func (p *Process) isRestartable() bool {
	p.Lock()
	exitCode := p.getExitCode()
	p.Unlock()
	if p.isStopped.Swap(false) {
		return false
	}
	if p.procConf.RestartPolicy.Restart == types.RestartPolicyNo {
		return false
	}

	if exitCode != 0 && p.procConf.RestartPolicy.Restart == types.RestartPolicyExitOnFailure {
		return false
	}

	if exitCode != 0 && p.procConf.RestartPolicy.Restart == types.RestartPolicyOnFailure {
		if p.procConf.RestartPolicy.MaxRestarts == 0 {
			return true
		}
		return p.procState.Restarts < p.procConf.RestartPolicy.MaxRestarts
	}

	// TODO consider if forking daemon should disable RestartPolicyAlways
	if p.procConf.RestartPolicy.Restart == types.RestartPolicyAlways {
		if p.procConf.RestartPolicy.MaxRestarts == 0 {
			return true
		}
		return p.procState.Restarts < p.procConf.RestartPolicy.MaxRestarts
	}

	return false
}

func (p *Process) waitForStarted() {
	select {
	case <-p.procStartedChan:
	case <-p.procRunCtx.Done():
	}
}

func (p *Process) waitForCompletion() int {
	p.Lock()
	defer p.Unlock()

	for !p.done {
		p.procCond.Wait()
	}
	return p.getExitCode()
}

func (p *Process) waitUntilReady() bool {
	<-p.procReadyCtx.Done()
	if p.procState.Health == types.ProcessHealthReady {
		return true
	}
	log.Error().Msgf("Process %s was aborted and won't become ready", p.getName())
	p.setExitCode(1)
	return false

}

func (p *Process) waitUntilLogReady() bool {
	<-p.procLogReadyCtx.Done()
	err := context.Cause(p.procLogReadyCtx)
	if errors.Is(err, context.Canceled) {
		return true
	}
	log.Error().Err(err).Msgf("Process %s was aborted and won't become log ready", p.getName())
	return false

}

func (p *Process) wontRun() {
	p.onProcessEnd(types.ProcessStateSkipped)
}

// perform graceful process shutdown if defined in configuration
func (p *Process) shutDownNoRestart() error {
	p.prepareForShutDown()
	return p.shutDown()
}

// perform graceful process shutdown if defined in configuration
func (p *Process) shutDown() error {
	return p.stopProcess(true)
}

// internal stop for graceful shutdown in case of readiness probe failure
func (p *Process) internalStop() error {
	return p.stopProcess(false)
}

func (p *Process) stopProcess(cancelReadinessFuncs bool) error {
	p.runCancelFn()
	if !p.isRunning() {
		log.Debug().Msgf("process %s is in state %s not shutting down", p.getName(), p.getStatusName())
		// prevent pending process from running
		if p.isOneOfStates(types.ProcessStatePending) {
			p.onProcessEnd(types.ProcessStateTerminating)
		}
		if cancelReadinessFuncs {
			p.cancelReadyLogFunc(fmt.Errorf("process %s completed, cannot produce log lines", p.getName()))
		}
		return nil
	}
	p.setState(types.ProcessStateTerminating)
	p.stopProbes()
	if cancelReadinessFuncs {
		if p.readyProber != nil {
			p.readyCancelFn()
		}
		p.cancelReadyLogFunc(fmt.Errorf("process %s was shut down", p.getName()))
	}
	if isStringDefined(p.procConf.ShutDownParams.ShutDownCommand) {
		return p.doConfiguredStop(p.procConf.ShutDownParams)
	}
	err := p.command.Stop(p.procConf.ShutDownParams.Signal, p.procConf.ShutDownParams.ParentOnly)
	if err != nil {
		log.Error().Err(err).Msgf("terminating %s failed", p.getName())
	}
	if p.procConf.ShutDownParams.ShutDownTimeout != UndefinedShutdownTimeoutSec {
		return p.forceKillOnTimeout()
	}
	return err
}

func (p *Process) forceKillOnTimeout() error {
	p.mtxStopFn.Lock()
	p.waitForStoppedCtx, p.waitForStoppedFn = context.WithTimeout(context.Background(), time.Duration(p.procConf.ShutDownParams.ShutDownTimeout)*time.Second)
	p.mtxStopFn.Unlock()
	<-p.waitForStoppedCtx.Done()
	err := p.waitForStoppedCtx.Err()
	switch {
	case errors.Is(err, context.Canceled):
		return nil
	case errors.Is(err, context.DeadlineExceeded):
		log.Debug().Msgf("process failed to shut down within %d seconds, sending %d", p.procConf.ShutDownParams.ShutDownTimeout, syscall.SIGKILL)
		return p.command.Stop(int(syscall.SIGKILL), false)
	default:
		log.Error().Err(err).Msgf("terminating %s with timeout %d failed", p.getName(), p.procConf.ShutDownParams.ShutDownTimeout)
		return err
	}
}

func (p *Process) doConfiguredStop(params types.ShutDownParams) error {
	timeout := params.ShutDownTimeout
	if timeout == UndefinedShutdownTimeoutSec {
		timeout = DefaultShutdownTimeoutSec
	}
	log.Debug().Msgf("terminating %s with timeout %d ...", p.getName(), timeout)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	defer p.notifyDaemonStopped()

	cmd := command.BuildCommandShellArgContext(ctx, p.shellConfig, params.ShutDownCommand)
	cmd.SetEnv(p.getProcessEnvironment())
	cmd.SetDir(p.procConf.WorkingDir)

	if err := cmd.Run(); err != nil {
		// the process termination timedout and it will be killed
		log.Error().Msgf("terminating %s with timeout %d failed - %s", p.getName(), timeout, err.Error())
		return p.command.Stop(int(syscall.SIGKILL), false)
	}
	return nil
}

func (p *Process) isRunning() bool {
	return p.isOneOfStates(types.ProcessStateRunning, types.ProcessStateLaunched, types.ProcessStateLaunching)
}

func (p *Process) prepareForShutDown() {
	// prevent restart during global shutdown or scale down
	//p.procConf.RestartPolicy.Restart = types.RestartPolicyNo
	p.isStopped.Store(true)

}

func (p *Process) onProcessStart() {
	if isStringDefined(p.procConf.LogLocation) {
		p.logger.Open(p.getLogPath(), p.procConf.LoggerConfig)
	}

	p.Lock()
	p.started = true
	p.Unlock()
	close(p.procStartedChan)
}

func (p *Process) onProcessEnd(state string) {
	if isStringDefined(p.procConf.LogLocation) {
		p.logger.Close()
	}
	p.mtxStopFn.Lock()
	if p.waitForStoppedFn != nil {
		p.waitForStoppedFn()
		p.waitForStoppedFn = nil
	}
	p.mtxStopFn.Unlock()
	p.cancelReadyLogFunc(fmt.Errorf("process %s completed", p.getName()))
	p.stopProbes()
	if p.readyProber != nil {
		p.readyCancelFn()
	}
	p.setState(state)
	p.updateProcState()

	p.Lock()
	p.done = true
	p.Unlock()
	p.procCond.Broadcast()
}

func (p *Process) getLogPath() string {
	logLocation := p.procConf.LogLocation

	if strings.Contains(logLocation, LogReplicaNum) {
		replicaStr := strconv.Itoa(p.procConf.ReplicaNum)
		logLocation = strings.Replace(logLocation, LogReplicaNum, replicaStr, -1)
	} else if p.procConf.Replicas > 1 {
		logLocation = fmt.Sprintf("%s.%d", logLocation, p.procConf.ReplicaNum)
	}

	return logLocation
}

func (p *Process) getName() string {
	return p.procConf.ReplicaName
}

func (p *Process) setName(replicaName string) {
	p.procConf.ReplicaName = replicaName
}

func (p *Process) getNameWithSmartReplica() string {
	if p.procConf.Replicas > 1 {
		return p.getName()
	}
	return p.procConf.Name
}

func (p *Process) getCommand() []string {
	return append(
		[]string{(*p.procConf).Executable},
		p.mergeExtraArgs()...,
	)
}

func (p *Process) updateProcState() {
	isRunning := p.isRunning()
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	if isRunning {
		dur := time.Since(p.getStartTime())
		p.procState.SystemTime = HumanDuration(dur)
		p.procState.Age = dur
		p.procState.Name = p.getName()
		if time.Since(p.lastStatusPoll) > p.refRate {
			p.procState.Mem, p.procState.CPU = p.getResourceUsage()
			p.lastStatusPoll = time.Now()
		}
	}
	p.procState.IsRunning = isRunning
	p.procState.IsElevated = p.procConf.IsElevated
	p.procState.PasswordProvided = p.passProvided

}

func (p *Process) setStartTime(startTime time.Time) {
	p.timeMutex.Lock()
	defer p.timeMutex.Unlock()
	p.startTime = startTime
}

func (p *Process) getStartTime() time.Time {
	p.timeMutex.Lock()
	defer p.timeMutex.Unlock()
	return p.startTime
}

func (p *Process) getResourceUsage() (int64, float64) {
	if p.procConf.IsDaemon {
		return -1, -1
	}
	if p.metricsProc == nil {
		return -1, -1
	}
	if p.withRecursiveMetrics {
		return p.getProcResourcesRecursive(p.metricsProc)
	}

	return p.getProcResources(p.metricsProc)
}

// recursively get the memory and cpu usage of the process and its children
func (p *Process) getProcResourcesRecursive(proc *puproc.Process) (int64, float64) {
	totalMem, totalCpu := p.getProcResources(proc)
	childrenProcs, err := proc.Children()
	if err != nil {
		log.Err(err).
			Str("process", p.getName()).
			Int("pid", p.procState.Pid).
			Msg("Error retrieving children")
		return totalMem, totalCpu
	}
	for _, childProc := range childrenProcs {
		childMem, childCpu := p.getProcResourcesRecursive(childProc)
		if childMem >= 0 {
			totalMem += childMem
		}
		if childCpu >= 0 {
			totalCpu += childCpu
		}
	}
	return totalMem, totalCpu
}

// getResourceUsage returns the memory and cpu usage of the process
// if the process is not running, returns -1 for both values
func (p *Process) getProcResources(proc *puproc.Process) (int64, float64) {
	memoryInfo, err := proc.MemoryInfo()
	if err != nil {
		//log.Err(err).
		//	Str("process", p.getName()).
		//	Int("pid", p.procState.Pid).
		//	Msg("Error retrieving memory stats")
		return -1, -1
	}
	cpuPercent, err := proc.CPUPercentWithContext(context.Background())
	if err != nil {
		log.Err(err).
			Str("process", p.getName()).
			Int("pid", p.procState.Pid).
			Msg("Error retrieving cpu stats")
		return int64(memoryInfo.RSS), -1
	}
	return int64(memoryInfo.RSS), cpuPercent
}

func (p *Process) handleInput(pipe io.WriteCloser) {
	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Err(err).
				Str("process", p.getName()).
				Msg("error reading from stdin")
			continue
		}
		_, _ = pipe.Write([]byte(input))
	}
}

func (p *Process) handleOutput(pipe io.ReadCloser, output string, handler func(message string), done chan struct{}) {
	reader := bufio.NewReader(pipe)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			var pathErr *os.PathError
			ok := errors.As(err, &pathErr)
			if ok && pathErr.Path == "/dev/ptmx" {
				break
			}
			log.Err(err).
				Str("process", p.getName()).
				Msgf("error reading from %s", output)
			break
		}
		if p.procConf.ReadyLogLine != "" && p.procState.Health == types.ProcessHealthUnknown && strings.Contains(line, p.procConf.ReadyLogLine) {
			p.procState.Health = types.ProcessHealthReady
			p.cancelReadyLogFunc(nil)
		}
		p.checkElevatedProcOutput(line)
		handler(strings.TrimSuffix(line, "\n"))
	}
	close(done)
}

func (p *Process) checkElevatedProcOutput(line string) {
	if p.procConf.IsElevated &&
		!p.passProvided {
		if p.waitForPassCancelFn != nil {
			if isWrongPasswordEntered(line) {
				log.Warn().
					Str("process", p.getName()).
					Msgf("Password rejected %s", line)
			} else {
				log.Info().
					Str("process", p.getName()).
					Msg("Password accepted")
				p.passProvided = true
			}
			p.waitForPassCancelFn()
			p.waitForPassCancelFn = nil
		} else {
			p.passProvided = true
		}
	}
}

func (p *Process) handleInfo(message string) {
	p.logger.Info(message, p.getName(), p.procConf.ReplicaNum)
	if p.printLogs {
		fmt.Printf("[%s\t] %s\n", p.procColor(p.getName()), message)
	}
	p.logBuffer.Write(message)
}

func (p *Process) handleError(message string) {
	p.logger.Error(message, p.getName(), p.procConf.ReplicaNum)
	if p.printLogs {
		fmt.Printf("[%s\t] %s\n", p.procColor(p.getName()), p.redColor(message))
	}
	p.logBuffer.Write(message)
}

func (p *Process) isState(state string) bool {
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	return p.procState.Status == state
}

func (p *Process) isOneOfStates(states ...string) bool {
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	for _, state := range states {
		if p.procState.Status == state {
			return true
		}
	}
	return false
}

func (p *Process) setState(state string) {
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	p.procState.Status = state
	p.onStateChange(state)
}

func (p *Process) getState() *types.ProcessState {
	p.updateProcState()
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	return p.procState
}

type filterFn func(*types.ProcessState)

func (p *Process) getStateData(filter filterFn) {
	p.updateProcState()
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	if filter != nil {
		filter(p.procState)
	}
}

func (p *Process) getStatusName() string {
	p.updateProcState()
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	return p.procState.Status
}

func (p *Process) setStateAndRun(state string, runnable func() error) error {
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	p.procState.Status = state
	p.onStateChange(state)
	return runnable()
}

func (p *Process) onStateChange(state string) {
	switch state {
	case types.ProcessStateSkipped:
		p.setExitCode(1)
	case types.ProcessStateRestarting:
		fallthrough
	case types.ProcessStateLaunching:
		fallthrough
	case types.ProcessStateTerminating:
		p.procState.Health = types.ProcessHealthUnknown
	}
}

func (p *Process) getStartingStateName() string {
	if p.procConf.IsDaemon {
		return types.ProcessStateLaunching
	}
	return types.ProcessStateRunning
}

func (p *Process) setUpProbes() {
	var err error
	if p.procConf.LivenessProbe != nil {
		p.liveProber, err = health.New(
			p.getName()+"_live_probe",
			*p.procConf.LivenessProbe,
			p.getProcessEnvironment(),
			p.onLivenessCheckEnd,
		)
		if err != nil {
			log.Error().Msgf("failed to setup liveness probe for %s - %s", p.getName(), err.Error())
			p.logBuffer.Write("Error: " + err.Error())
		}
	}

	if p.procConf.ReadinessProbe != nil {
		p.readyProber, err = health.New(
			p.getName()+"_ready_probe",
			*p.procConf.ReadinessProbe,
			p.getProcessEnvironment(),
			p.onReadinessCheckEnd,
		)
		if err != nil {
			log.Error().Msgf("failed to setup readiness probe for %s - %s", p.getName(), err.Error())
			p.logBuffer.Write("Error: " + err.Error())
		}
	}
}

func (p *Process) startProbes() {
	if p.liveProber != nil {
		p.liveProber.Start()
	}

	if p.readyProber != nil {
		p.readyProber.Start()
	}
}

func (p *Process) stopProbes() {
	if p.liveProber != nil {
		p.liveProber.Stop()
	}

	if p.readyProber != nil {
		p.readyProber.Stop()
	}
}

func (p *Process) onLivenessCheckEnd(_, isFatal bool, err string, details interface{}) {
	if isFatal {
		p.logBuffer.Write("Error: liveness check fail - " + err)
		p.notifyDaemonStopped()
		rcMap, ok := details.(map[string]string)
		if ok {
			p.printDetails(rcMap, err, "liveness")
		}
	}
}

func (p *Process) printDetails(details map[string]string, err, source string) {
	exitCode, _ := strconv.Atoi(details["exit_code"])
	output := details["output"]
	log.Warn().
		Str("error", err).
		Int("exit_code", exitCode).
		Msgf("%s %s probe failed", p.getName(), source)
	if output != "" {
		log.Debug().Msgf("%s %s failed with output: %s", p.getName(), source, output)
	}
}

func (p *Process) onReadinessCheckEnd(isOk, isFatal bool, err string, details interface{}) {
	if isFatal {
		p.procState.Health = types.ProcessHealthNotReady
		p.logBuffer.Write("Error: readiness check fail - " + err)
		_ = p.internalStop()
	} else if isOk {
		p.procState.Health = types.ProcessHealthReady
		p.readyCancelFn()
	} else {
		p.procState.Health = types.ProcessHealthNotReady
	}

	//log exec error if not healthy and output is not empty
	if p.procState.Health == types.ProcessHealthNotReady {
		rcMap, ok := details.(map[string]string)
		if ok {
			p.printDetails(rcMap, err, "readiness")
		}
	}
}

func (p *Process) validateProcess() error {
	if isStringDefined(p.procConf.WorkingDir) {
		stat, err := os.Stat(p.procConf.WorkingDir)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			return fmt.Errorf("%s is not a directory", p.procConf.WorkingDir)
		}
	}
	return nil
}

func (p *Process) getOpenPorts(ports *types.ProcessPorts) error {
	socks, err := netstat.TCPSocks(func(s *netstat.SockTabEntry) bool {
		return s.State == netstat.Listen
	})
	if err != nil {
		log.Err(err).Msgf("failed to get open ports for %s", p.getName())
		return err
	}
	socksv6, err := netstat.TCP6Socks(func(s *netstat.SockTabEntry) bool {
		return s.State == netstat.Listen
	})
	socks = append(socks, socksv6...)
	if err != nil {
		log.Err(err).Msgf("failed to get open ports for %s", p.getName())
		return err
	}
	for _, e := range socks {
		if e.Process != nil && e.Process.Pid == p.procState.Pid {
			log.Debug().Msgf("%s is listening on %d", p.getName(), e.LocalAddr.Port)
			ports.TcpPorts = append(ports.TcpPorts, e.LocalAddr.Port)
		}
	}
	return nil
}

func (p *Process) getExitCode() int {
	defer p.confMtx.Unlock()
	p.confMtx.Lock()
	return p.procState.ExitCode
}

func (p *Process) setExitCode(code int) {
	defer p.confMtx.Unlock()
	p.confMtx.Lock()
	p.procState.ExitCode = code
}

// set elevated process password
func (p *Process) setPassword(password string) error {
	if p.procConf.IsElevated && !p.passProvided && p.stdin != nil {
		log.Debug().Msgf(`Set password for elevated process %s`, p.getName())
		p.waitForPassCtx, p.waitForPassCancelFn = context.WithTimeout(context.Background(), 4*time.Second)
		_, err := p.stdin.Write([]byte(password + "\n"))
		if err != nil {
			log.Error().Err(err).Msgf(`Failed to write to stdin pipe for process %s`, p.getName())
			return err
		}
		// wait for password confirmation
		err = p.waitForPass()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Process) waitForPass() error {
	<-p.waitForPassCtx.Done()
	if !p.passProvided {
		return fmt.Errorf("wrong password for elevated process %s, %s", p.getName(), p.waitForPassCtx.Err())
	}
	return nil
}

func isWrongPasswordEntered(output string) bool {
	if runtime.GOOS != "windows" {
		return strings.Contains(output, "Sorry, try again")
	} else {
		return strings.Contains(output, "The user name or password is incorrect")
	}
}

func (p *Process) cancelReadyLogFunc(err error) {
	p.mtxStopFn.Lock()
	defer p.mtxStopFn.Unlock()
	if p.readyLogCancelFn != nil {
		p.readyLogCancelFn(err)
		p.readyLogCancelFn = nil
	}
}
