package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/pclog"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
)

const (
	UndefinedShutdownTimeoutSec = 0
	DefaultShutdownTimeoutSec   = 10
)

type Process struct {
	sync.Mutex

	globalEnv     []string
	procConf      ProcessConfig
	procState     *ProcessState
	stateMtx      sync.Mutex
	procCond      sync.Cond
	procStateChan chan string
	procReadyChan chan string
	procReadyCtx  context.Context
	readyCancelFn context.CancelFunc
	procColor     func(a ...interface{}) string
	noColor       func(a ...interface{}) string
	redColor      func(a ...interface{}) string
	logBuffer     *pclog.ProcessLogBuffer
	logger        pclog.PcLogger
	command       *exec.Cmd
	done          bool
	replica       int
	startTime     time.Time
	liveProber    *health.Prober
	readyProber   *health.Prober
	shellConfig   command.ShellConfig
}

func NewProcess(
	globalEnv []string,
	logger pclog.PcLogger,
	procConf ProcessConfig,
	procState *ProcessState,
	procLog *pclog.ProcessLogBuffer,
	replica int,
	shellConfig command.ShellConfig) *Process {
	colNumeric := rand.Intn(int(color.FgHiWhite)-int(color.FgHiBlack)) + int(color.FgHiBlack)

	proc := &Process{
		globalEnv:     globalEnv,
		procConf:      procConf,
		procColor:     color.New(color.Attribute(colNumeric), color.Bold).SprintFunc(),
		redColor:      color.New(color.FgHiRed).SprintFunc(),
		noColor:       color.New(color.Reset).SprintFunc(),
		logger:        logger,
		procState:     procState,
		done:          false,
		replica:       replica,
		logBuffer:     procLog,
		shellConfig:   shellConfig,
		procStateChan: make(chan string, 1),
		procReadyChan: make(chan string, 1),
	}

	proc.procReadyCtx, proc.readyCancelFn = context.WithCancel(context.Background())
	proc.setUpProbes()
	proc.procCond = *sync.NewCond(proc)
	return proc
}

func (p *Process) run() int {
	if p.isState(ProcessStateTerminating) {
		return 0
	}

	if err := p.validateProcess(); err != nil {
		log.Error().Err(err).Msgf("Failed to run command %s for process %s", p.getCommand(), p.getName())
		p.onProcessEnd(ProcessStateError)
		return 1
	}

	for {
		err := p.setStateAndRun(p.getStartingStateName(), p.getProcessStarter())
		if err != nil {
			log.Error().Err(err).Msgf("Failed to run command %s for process %s", p.getCommand(), p.getName())
			p.logBuffer.Write(err.Error())
			p.onProcessEnd(ProcessStateError)
			return 1
		}

		p.startTime = time.Now()
		p.procState.Pid = p.command.Process.Pid
		log.Info().Msgf("%s started", p.getName())

		p.startProbes()

		//Wait should wait for I/O consumption, but if the execution is too fast
		//e.g. echo 'hello world' the output will not reach the pipe
		//TODO Fix this
		time.Sleep(50 * time.Millisecond)
		_ = p.command.Wait()
		p.Lock()
		p.procState.ExitCode = p.command.ProcessState.ExitCode()
		p.Unlock()
		log.Info().Msgf("%s exited with status %d", p.getName(), p.procState.ExitCode)

		if p.isDaemonLaunched() {
			p.setState(ProcessStateLaunched)
			p.waitForDaemonCompletion()
		}

		if !p.isRestartable() {
			break
		}
		p.setState(ProcessStateRestarting)
		p.procState.Restarts += 1
		log.Info().Msgf("Restarting %s in %v second(s)... Restarts: %d",
			p.procConf.Name, p.getBackoff().Seconds(), p.procState.Restarts)

		time.Sleep(p.getBackoff())
	}
	p.onProcessEnd(ProcessStateCompleted)
	return p.procState.ExitCode
}

func (p *Process) getProcessStarter() func() error {
	return func() error {
		p.command = command.BuildCommandShellArg(p.shellConfig, p.getCommand())
		p.command.Env = p.getProcessEnvironment()
		p.command.Dir = p.procConf.WorkingDir
		p.setProcArgs()
		stdout, _ := p.command.StdoutPipe()
		stderr, _ := p.command.StderrPipe()
		go p.handleOutput(stdout, p.handleInfo)
		go p.handleOutput(stderr, p.handleError)
		return p.command.Start()
	}
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
		"PC_PROC_NAME=" + p.getName(),
		"PC_REPLICA_NUM=" + strconv.Itoa(p.replica),
	}
	env = append(env, os.Environ()...)
	env = append(env, p.globalEnv...)
	env = append(env, p.procConf.Environment...)
	return env
}

func (p *Process) isRestartable() bool {
	exitCode := p.procState.ExitCode
	if p.procConf.RestartPolicy.Restart == RestartPolicyNo ||
		p.procConf.RestartPolicy.Restart == "" {
		return false
	}

	if exitCode != 0 && p.procConf.RestartPolicy.Restart == RestartPolicyExitOnFailure {
		return false
	}

	if exitCode != 0 && (p.procConf.RestartPolicy.Restart == RestartPolicyOnFailureDeprecated ||
		p.procConf.RestartPolicy.Restart == RestartPolicyOnFailure) {
		if p.procConf.RestartPolicy.MaxRestarts == 0 {
			return true
		}
		return p.procState.Restarts < p.procConf.RestartPolicy.MaxRestarts
	}

	// TODO consider if forking daemon should disable RestartPolicyAlways
	if p.procConf.RestartPolicy.Restart == RestartPolicyAlways {
		if p.procConf.RestartPolicy.MaxRestarts == 0 {
			return true
		}
		return p.procState.Restarts < p.procConf.RestartPolicy.MaxRestarts
	}

	return false
}

func (p *Process) waitForCompletion() int {
	p.Lock()
	defer p.Unlock()

	for !p.done {
		p.procCond.Wait()
	}
	return p.procState.ExitCode
}

func (p *Process) waitUntilReady() bool {
	for {
		select {
		case <-p.procReadyCtx.Done():
			log.Error().Msgf("Process %s was aborted and won't become ready", p.getName())
			return false
		case ready := <-p.procReadyChan:
			if ready == ProcessHealthReady {
				return true
			}
		}
	}
}

func (p *Process) wontRun() {
	p.onProcessEnd(ProcessStateCompleted)

}

// perform graceful process shutdown if defined in configuration
func (p *Process) shutDown() error {
	if !p.isRunning() {
		log.Debug().Msgf("process %s is in state %s not shutting down", p.getName(), p.procState.Status)
		// prevent pending process from running
		p.onProcessEnd(ProcessStateTerminating)
		return nil
	}
	p.setState(ProcessStateTerminating)
	p.stopProbes()
	if isStringDefined(p.procConf.ShutDownParams.ShutDownCommand) {
		return p.doConfiguredStop(p.procConf.ShutDownParams)
	}
	return p.stop(p.procConf.ShutDownParams.Signal)
}

func (p *Process) doConfiguredStop(params ShutDownParams) error {
	timeout := params.ShutDownTimeout
	if timeout == UndefinedShutdownTimeoutSec {
		timeout = DefaultShutdownTimeoutSec
	}
	log.Debug().Msgf("terminating %s with timeout %d ...", p.getName(), timeout)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	defer p.notifyDaemonStopped()

	cmd := command.BuildCommandShellArgContext(ctx, p.shellConfig, params.ShutDownCommand)
	cmd.Env = p.getProcessEnvironment()

	if err := cmd.Run(); err != nil {
		// the process termination timedout and it will be killed
		log.Error().Msgf("terminating %s with timeout %d failed - %s", p.getName(), timeout, err.Error())
		return p.stop(int(syscall.SIGKILL))
	}
	return nil
}

func (p *Process) isRunning() bool {
	return p.isOneOfStates(ProcessStateRunning, ProcessStateLaunched)
}

func (p *Process) prepareForShutDown() {
	// prevent restart during global shutdown
	p.procConf.RestartPolicy.Restart = RestartPolicyNo
}

func (p *Process) onProcessEnd(state string) {
	if isStringDefined(p.procConf.LogLocation) {
		p.logger.Close()
	}
	p.stopProbes()
	p.setState(state)

	p.Lock()
	p.done = true
	p.Unlock()
	p.procCond.Broadcast()
}

func (p *Process) getName() string {
	return p.procConf.Name
}

func (p *Process) getNameWithReplica() string {
	return fmt.Sprintf("%s_%d", p.procConf.Name, p.replica)
}

func (p *Process) getCommand() string {
	return p.procConf.Command
}

func (p *Process) updateProcState() {
	if p.isRunning() {
		dur := time.Since(p.startTime)
		p.procState.SystemTime = durationToString(dur)
		p.procState.IsRunning = true
	}
}

func (p *Process) handleOutput(pipe io.ReadCloser,
	handler func(message string)) {
	outscanner := bufio.NewScanner(pipe)
	outscanner.Split(bufio.ScanLines)
	for outscanner.Scan() {
		handler(outscanner.Text())
	}
}

func (p *Process) handleInfo(message string) {
	p.logger.Info(message, p.getName(), p.replica)
	fmt.Printf("[%s\t] %s\n", p.procColor(p.getNameWithReplica()), message)
	p.logBuffer.Write(message)
}

func (p *Process) handleError(message string) {
	p.logger.Error(message, p.getName(), p.replica)
	fmt.Printf("[%s\t] %s\n", p.procColor(p.getNameWithReplica()), p.redColor(message))
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

func (p *Process) setStateAndRun(state string, runnable func() error) error {
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	p.procState.Status = state
	p.onStateChange(state)
	return runnable()
}

func (p *Process) onStateChange(state string) {
	switch state {
	case ProcessStateRestarting:
		fallthrough
	case ProcessStateLaunching:
		fallthrough
	case ProcessStateTerminating:
		p.procState.Health = ProcessHealthUnknown
	}
}

func (p *Process) getStartingStateName() string {
	if p.procConf.IsDaemon {
		return ProcessStateLaunching
	}
	return ProcessStateRunning
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
		p.procState.Health = ProcessHealthNotReady
		log.Info().Msgf("%s is not ready anymore - %s", p.getName(), err)
		p.logBuffer.Write("Error: readiness check fail - " + err)
		_ = p.shutDown()
	} else if isOk {
		p.procState.Health = ProcessHealthReady
		p.procReadyChan <- ProcessHealthReady
	} else {
		p.procState.Health = ProcessHealthNotReady
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
