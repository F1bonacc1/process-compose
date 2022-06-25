package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/f1bonacc1/process-compose/src/pclog"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
)

const (
	DEFAULT_SHUTDOWN_TIMEOUT_SEC = 10
)

type Process struct {
	globalEnv []string
	procConf  ProcessConfig
	procState *ProcessState
	sync.Mutex
	stateMtx  sync.Mutex
	procCond  sync.Cond
	procColor func(a ...interface{}) string
	noColor   func(a ...interface{}) string
	redColor  func(a ...interface{}) string
	logBuffer *pclog.ProcessLogBuffer
	logger    pclog.PcLogger
	cmd       *exec.Cmd
	done      bool
	replica   int
	startTime time.Time
}

func NewProcess(
	globalEnv []string,
	logger pclog.PcLogger,
	procConf ProcessConfig,
	procState *ProcessState,
	procLog *pclog.ProcessLogBuffer,
	replica int) *Process {
	colNumeric := rand.Intn(int(color.FgHiWhite)-int(color.FgHiBlack)) + int(color.FgHiBlack)
	//logger, _ := zap.NewProduction()

	proc := &Process{
		globalEnv: globalEnv,
		procConf:  procConf,
		procColor: color.New(color.Attribute(colNumeric), color.Bold).SprintFunc(),
		redColor:  color.New(color.FgHiRed).SprintFunc(),
		noColor:   color.New(color.Reset).SprintFunc(),
		logger:    logger,
		procState: procState,
		done:      false,
		replica:   replica,
		logBuffer: procLog,
	}
	proc.procCond = *sync.NewCond(proc)
	return proc
}

func (p *Process) run() error {
	if p.isState(ProcessStateTerminating) {
		return nil
	}
	for {
		starter := func() error {
			p.cmd = exec.Command(getRunnerShell(), getRunnerArg(), p.getCommand())
			p.cmd.Env = p.getProcessEnvironment()
			p.setProcArgs()
			stdout, _ := p.cmd.StdoutPipe()
			stderr, _ := p.cmd.StderrPipe()
			go p.handleOutput(stdout, p.handleInfo)
			go p.handleOutput(stderr, p.handleError)
			return p.cmd.Start()
		}
		p.setStateAndRun(ProcessStateRunning, starter)

		p.startTime = time.Now()
		p.procState.Pid = p.cmd.Process.Pid

		//Wait should wait for I/O consumption, but if the execution is too fast
		//e.g. echo 'hello world' the output will not reach the pipe
		//TODO Fix this
		time.Sleep(50 * time.Millisecond)
		p.cmd.Wait()
		p.Lock()
		p.procState.ExitCode = p.cmd.ProcessState.ExitCode()
		p.Unlock()
		log.Info().Msgf("%s exited with status %d", p.procConf.Name, p.procState.ExitCode)

		if !p.isRestartable(p.procState.ExitCode) {
			break
		}
		p.setState(ProcessStateRestarting)
		p.procState.Restarts += 1
		log.Info().Msgf("Restarting %s in %v second(s)... Restarts: %d",
			p.procConf.Name, p.getBackoff().Seconds(), p.procState.Restarts)

		time.Sleep(p.getBackoff())
	}
	p.onProcessEnd()
	return nil
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

func (p *Process) isRestartable(exitCode int) bool {
	if p.procConf.RestartPolicy.Restart == RestartPolicyNo ||
		p.procConf.RestartPolicy.Restart == "" {
		return false
	}

	if exitCode != 0 && p.procConf.RestartPolicy.Restart == RestartPolicyOnFailure {
		if p.procConf.RestartPolicy.MaxRestarts == 0 {
			return true
		}
		return p.procState.Restarts < p.procConf.RestartPolicy.MaxRestarts
	}

	if p.procConf.RestartPolicy.Restart == RestartPolicyAlways {
		if p.procConf.RestartPolicy.MaxRestarts == 0 {
			return true
		}
		return p.procState.Restarts < p.procConf.RestartPolicy.MaxRestarts
	}

	return false
}

func (p *Process) waitForCompletion(waitee string) int {
	p.Lock()
	defer p.Unlock()

	for !p.done {
		p.procCond.Wait()
	}
	return p.procState.ExitCode
}

func (p *Process) wontRun() {
	p.onProcessEnd()

}

// perform gracefull process shutdown if defined in configuration
func (p *Process) shutDown() error {
	if !p.isState(ProcessStateRunning) {
		log.Debug().Msgf("process %s is in state %s not shutting down", p.getName(), p.procState.Status)
		// prevent pending process from running
		p.setState(ProcessStateTerminating)
		return nil
	}
	p.setState(ProcessStateTerminating)
	if isStringDefined(p.procConf.ShutDownParams.ShutDownCommand) {
		return p.doConfiguredStop(p.procConf.ShutDownParams)
	}
	return p.stop(p.procConf.ShutDownParams.Signal)
}

func (p *Process) doConfiguredStop(params ShutDownParams) error {
	timeout := params.ShutDownTimeout
	if timeout == 0 {
		timeout = DEFAULT_SHUTDOWN_TIMEOUT_SEC
	}
	log.Debug().Msgf("terminating %s with timeout %d ...", p.getName(), timeout)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, getRunnerShell(), getRunnerArg(), params.ShutDownCommand)
	cmd.Env = p.getProcessEnvironment()

	if err := cmd.Run(); err != nil {
		// the process termination timedout and it will be killed
		log.Error().Msgf("terminating %s with timeout %d failed - %s", p.getName(), timeout, err.Error())
		return p.stop(int(syscall.SIGKILL))
	}
	return nil
}

func (p *Process) prepareForShutDown() {
	// prevent restart during global shutdown
	p.procConf.RestartPolicy.Restart = RestartPolicyNo
}

func (p *Process) onProcessEnd() {
	if isStringDefined(p.procConf.LogLocation) {
		p.logger.Close()
	}
	p.setState(ProcessStateCompleted)

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
	if p.isState(ProcessStateRunning) {
		dur := time.Since(p.startTime)
		p.procState.SystemTime = durationToString(dur)
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
	fmt.Printf("[%s]\t%s\n", p.procColor(p.getNameWithReplica()), message)
	p.logBuffer.Write(message)
}

func (p *Process) handleError(message string) {
	p.logger.Error(message, p.getName(), p.replica)
	fmt.Printf("[%s]\t%s\n", p.procColor(p.getNameWithReplica()), p.redColor(message))
	p.logBuffer.Write(message)
}

func (p *Process) isState(state string) bool {
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	return p.procState.Status == state
}

func (p *Process) setState(state string) {
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	p.procState.Status = state
}

func (p *Process) setStateAndRun(state string, runnable func() error) error {
	p.stateMtx.Lock()
	defer p.stateMtx.Unlock()
	p.procState.Status = state
	return runnable()
}

func getRunnerShell() string {
	shell, ok := os.LookupEnv("SHELL")
	if !ok {
		if runtime.GOOS == "windows" {
			shell = "cmd"
		} else {
			shell = "bash"
		}
	}
	return shell
}

func getRunnerArg() string {
	if runtime.GOOS == "windows" {
		return "/C"
	} else {
		return "-c"
	}
}
