package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
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
)

const (
	UndefinedShutdownTimeoutSec = 0
	DefaultShutdownTimeoutSec   = 10
	EnvReplicaNum               = "PC_REPLICA_NUM"
	LogReplicaNum               = "{" + EnvReplicaNum + "}"
)

type Process struct {
	sync.Mutex
	globalEnv        []string
	confMtx          sync.Mutex
	procConf         *types.ProcessConfig
	procState        *types.ProcessState
	stateMtx         sync.Mutex
	procCond         sync.Cond
	procStartedCond  sync.Cond
	procStateChan    chan string
	procReadyCtx     context.Context
	readyCancelFn    context.CancelFunc
	procLogReadyCtx  context.Context
	readyLogCancelFn context.CancelCauseFunc
	procRunCtx       context.Context
	runCancelFn      context.CancelFunc
	procColor        func(a ...interface{}) string
	noColor          func(a ...interface{}) string
	redColor         func(a ...interface{}) string
	logBuffer        *pclog.ProcessLogBuffer
	logger           pclog.PcLogger
	command          Commander
	started          bool
	done             bool
	timeMutex        sync.Mutex
	startTime        time.Time
	liveProber       *health.Prober
	readyProber      *health.Prober
	shellConfig      command.ShellConfig
	printLogs        bool
	isMain           bool
	extraArgs        []string
	isStopped        atomic.Bool
	isDevRestart     atomic.Bool
	devWatchers      []devWatch
}

type devWatch struct {
	sub    *WatchmanSub
	config *types.Watch
}

func NewProcess(
	globalEnv []string,
	logger pclog.PcLogger,
	procConf *types.ProcessConfig,
	processState *types.ProcessState,
	procLog *pclog.ProcessLogBuffer,
	shellConfig command.ShellConfig,
	printLogs bool,
	isMain bool,
	extraArgs []string,
	watchman *Watchman,
) *Process {
	colNumeric := rand.Intn(int(color.FgHiWhite)-int(color.FgHiBlack)) + int(color.FgHiBlack)

	proc := &Process{
		globalEnv:     globalEnv,
		procConf:      procConf,
		procState:     processState,
		procColor:     color.New(color.Attribute(colNumeric), color.Bold).SprintFunc(),
		redColor:      color.New(color.FgHiRed).SprintFunc(),
		noColor:       color.New(color.Reset).SprintFunc(),
		logger:        logger,
		started:       false,
		done:          false,
		logBuffer:     procLog,
		shellConfig:   shellConfig,
		procStateChan: make(chan string, 1),
		printLogs:     printLogs,
		isMain:        isMain,
		extraArgs:     extraArgs,
	}

	proc.procReadyCtx, proc.readyCancelFn = context.WithCancel(context.Background())
	proc.procLogReadyCtx, proc.readyLogCancelFn = context.WithCancelCause(context.Background())
	proc.procRunCtx, proc.runCancelFn = context.WithCancel(context.Background())
	proc.setUpProbes()
	proc.procCond = *sync.NewCond(proc)
	proc.procStartedCond = *sync.NewCond(proc)
	proc.setUpWatchman(watchman)
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
		p.stateMtx.Unlock()
		log.Info().
			Str("process", p.getName()).
			Strs("command", p.getCommand()).
			Msg("Started")

		p.startProbes()

		// Wait should wait for I/O consumption, but if the execution is too fast
		// e.g. echo 'hello world' the output will not reach the pipe
		// TODO Fix this
		time.Sleep(50 * time.Millisecond)
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
			break
		case <-time.After(p.getBackoff()):
			p.handleInfo("\n")
			continue
		}
	}
	p.onProcessEnd(types.ProcessStateCompleted)
	return p.getExitCode()
}

func (p *Process) getProcessStarter() func() error {
	return func() error {
		p.command = p.getCommander()
		p.command.SetEnv(p.getProcessEnvironment())
		p.command.SetDir(p.procConf.WorkingDir)

		if p.isMain {
			p.command.AttachIo()
		} else {
			p.command.SetCmdArgs()
			stdout, _ := p.command.StdoutPipe()
			go p.handleOutput(stdout, p.handleInfo)
			if !p.procConf.IsTty {
				stderr, _ := p.command.StderrPipe()
				go p.handleOutput(stderr, p.handleError)
			}
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
	return env
}

func (p *Process) isRestartable() bool {
	p.Lock()
	exitCode := p.getExitCode()
	p.Unlock()

	if p.isDevRestart.Swap(false) {
		return true
	}

	if p.isStopped.Swap(false) {
		return false
	}
	if p.procConf.RestartPolicy.Restart == types.RestartPolicyNo ||
		p.procConf.RestartPolicy.Restart == "" {
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
	p.Lock()
	defer p.Unlock()

	for !p.started {
		p.procStartedCond.Wait()
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
	for {
		select {
		case <-p.procReadyCtx.Done():
			if p.procState.Health == types.ProcessHealthReady {
				return true
			}
			log.Error().Msgf("Process %s was aborted and won't become ready", p.getName())
			p.setExitCode(1)
			return false
		case <-p.procLogReadyCtx.Done():
			err := context.Cause(p.procLogReadyCtx)
			if errors.Is(err, context.Canceled) {
				return true
			}
			log.Error().Err(err).Msgf("Process %s was aborted and won't become ready", p.getName())
			return false
		}
	}
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
	p.runCancelFn()
	if !p.isRunning() {
		log.Debug().Msgf("process %s is in state %s not shutting down", p.getName(), p.getStatusName())
		// prevent pending process from running
		p.onProcessEnd(types.ProcessStateTerminating)
		return nil
	}
	p.setState(types.ProcessStateTerminating)
	p.stopProbes()
	p.readyLogCancelFn(fmt.Errorf("process %s was shut down", p.getName()))
	if isStringDefined(p.procConf.ShutDownParams.ShutDownCommand) {
		return p.doConfiguredStop(p.procConf.ShutDownParams)
	}

	return p.command.Stop(p.procConf.ShutDownParams.Signal, p.procConf.ShutDownParams.ParentOnly)
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
	return p.isOneOfStates(types.ProcessStateRunning, types.ProcessStateLaunched)
}

func (p *Process) prepareForShutDown() {
	// prevent restart during global shutdown or scale down
	// p.procConf.RestartPolicy.Restart = types.RestartPolicyNo
	p.isStopped.Store(true)
}

func (p *Process) onProcessStart() {
	if isStringDefined(p.procConf.LogLocation) {
		p.logger.Open(p.getLogPath(), p.procConf.LoggerConfig)
	}

	for _, watch := range p.devWatchers {
		go func() {
			for {
				files, ok := watch.sub.Recv()
				if !ok {
					log.
						Debug().
						Msg("watchman sub closed, exiting")
					return
				}

				if len(files) == 0 {
					continue
				}

				p.isDevRestart.Store(true)
			}
		}()
	}

	p.Lock()
	p.started = true
	p.Unlock()
	p.procStartedCond.Broadcast()
}

func (p *Process) onProcessEnd(state string) {
	if isStringDefined(p.procConf.LogLocation) {
		p.logger.Close()
	}
	p.stopProbes()
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
		p.procState.SystemTime = durationToString(dur)
		p.procState.Age = dur
		p.procState.Name = p.getName()
	}
	p.procState.IsRunning = isRunning
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
		pipe.Write([]byte(input))
	}
}

func (p *Process) handleOutput(pipe io.ReadCloser, handler func(message string)) {
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
				Msg("error reading from stdout")
			break
		}
		if p.procConf.ReadyLogLine != "" && p.procState.Health == types.ProcessHealthUnknown && strings.Contains(line, p.procConf.ReadyLogLine) {
			p.procState.Health = types.ProcessHealthReady
			p.readyLogCancelFn(nil)
		}
		handler(strings.TrimSuffix(line, "\n"))
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
		p.readyCancelFn()
	}
}

func (p *Process) onLivenessCheckEnd(_, isFatal bool, err string) {
	if isFatal {
		log.Info().Msgf("%s is not alive anymore - %s", p.getName(), err)
		p.logBuffer.Write("Error: liveness check fail - " + err)
		p.notifyDaemonStopped()
	}
}

func (p *Process) onReadinessCheckEnd(isOk, isFatal bool, err string) {
	if isFatal {
		p.procState.Health = types.ProcessHealthNotReady
		log.Info().Msgf("%s is not ready anymore - %s", p.getName(), err)
		p.logBuffer.Write("Error: readiness check fail - " + err)
		_ = p.shutDown()
	} else if isOk {
		p.procState.Health = types.ProcessHealthReady
		p.readyCancelFn()
	} else {
		p.procState.Health = types.ProcessHealthNotReady
	}
}

func (p *Process) setUpWatchman(watchman *Watchman) {
	for _, config := range p.procConf.Watch {
		recv := watchman.Subscribe(config)
		p.devWatchers = append(p.devWatchers, devWatch{recv, config})
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
