package app

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/f1bonacc1/process-compose/src/pclog"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
)

type Process struct {
	globalEnv []string
	procConf  ProcessConfig
	procState *ProcessState
	sync.Mutex
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

func (p *Process) Run() error {
	for {
		p.cmd = exec.Command(getRunnerShell(), getRunnerArg(), p.getCommand())
		p.cmd.Env = p.getProcessEnvironment()
		stdout, _ := p.cmd.StdoutPipe()
		stderr, _ := p.cmd.StderrPipe()
		go p.handleOutput(stdout, p.handleInfo)
		go p.handleOutput(stderr, p.handleError)
		p.cmd.Start()
		p.startTime = time.Now()
		p.procState.Status = ProcessStateRunning
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
		p.procState.Status = ProcessStateRestarting
		p.procState.Restarts += 1
		log.Info().Msgf("Restarting %s in %v second(s)... Restarts: %d",
			p.procConf.Name, p.getBackoff().Seconds(), p.procState.Restarts)

		time.Sleep(p.getBackoff())
	}
	p.onProcessEnd()
	return nil
}

func (p *Process) stop() error {
	p.cmd.Process.Kill()
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
		"PC_PROC_NAME=" + p.GetName(),
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

func (p *Process) WaitForCompletion(waitee string) int {
	p.Lock()
	defer p.Unlock()

	for !p.done {
		p.procCond.Wait()
	}
	return p.procState.ExitCode
}

func (p *Process) WontRun() {
	p.onProcessEnd()

}

func (p *Process) onProcessEnd() {
	if isStringDefined(p.procConf.LogLocation) {
		p.logger.Close()
	}
	p.procState.Status = ProcessStateCompleted

	p.Lock()
	p.done = true
	p.Unlock()
	p.procCond.Broadcast()
}

func (p *Process) GetName() string {
	return p.procConf.Name
}

func (p *Process) GetNameWithReplica() string {
	return fmt.Sprintf("%s_%d", p.procConf.Name, p.replica)
}

func (p *Process) getCommand() string {
	return p.procConf.Command
}

func (p *Process) updateProcState() {
	if p.procState.Status == ProcessStateRunning {
		dur := time.Since(p.startTime)
		p.procState.SystemTime = durationToString(dur)
	}
}

func durationToString(dur time.Duration) string {
	if dur.Minutes() < 3 {
		return dur.Round(time.Second).String()
	} else if dur.Minutes() < 60 {
		return fmt.Sprintf("%.0fm", dur.Minutes())
	} else if dur.Hours() < 24 {
		return fmt.Sprintf("%dh%dm", int(dur.Hours()), int(dur.Minutes())%60)
	} else {
		return fmt.Sprintf("%dh", int(dur.Hours()))
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
	p.logger.Info(message, p.GetName(), p.replica)
	fmt.Printf("[%s]\t%s\n", p.procColor(p.GetNameWithReplica()), message)
	p.logBuffer.Write(message)
}

func (p *Process) handleError(message string) {
	p.logger.Error(message, p.GetName(), p.replica)
	fmt.Printf("[%s]\t%s\n", p.procColor(p.GetNameWithReplica()), p.redColor(message))
	p.logBuffer.Write(message)
}

func getRunnerShell() string {
	shell, ok := os.LookupEnv("SHELL")
	if !ok {
		if runtime.GOOS == "windows" {
			shell = "cmd"
		} else {
			shell = "bash"
		}
	} else {
		return shell
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
