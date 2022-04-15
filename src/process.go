package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
)

type Process struct {
	globalEnv       []string
	procConf        ProcessConfig
	restartsCounter int
	sync.Mutex
	procCond  sync.Cond
	exitCode  int
	procColor func(a ...interface{}) string
	noColor   func(a ...interface{}) string
	redColor  func(a ...interface{}) string
	logger    PcLogger
	done      bool
	replica   int
}

func NewProcess(globalEnv []string, logger PcLogger, procConf ProcessConfig, replica int) *Process {
	colNumeric := rand.Intn(int(color.FgHiWhite)-int(color.FgHiBlack)) + int(color.FgHiBlack)
	//logger, _ := zap.NewProduction()

	proc := &Process{
		globalEnv: globalEnv,
		procConf:  procConf,
		procColor: color.New(color.Attribute(colNumeric), color.Bold).SprintFunc(),
		redColor:  color.New(color.FgHiRed).SprintFunc(),
		noColor:   color.New(color.Reset).SprintFunc(),
		logger:    logger,
		exitCode:  -1,
		done:      false,
		replica:   replica,
	}
	proc.procCond = *sync.NewCond(proc)
	return proc
}

func (p *Process) Run() error {
	for {
		cmd := exec.Command(getRunnerShell(), getRunnerArg(), p.getCommand())
		cmd.Env = p.getProcessEnvironment()
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		go p.handleOutput(stdout, p.handleInfo)
		go p.handleOutput(stderr, p.handleError)
		cmd.Start()

		cmd.Wait()
		p.Lock()
		p.exitCode = cmd.ProcessState.ExitCode()
		p.Unlock()
		log.Info().Msgf("%s exited with status %d", p.procConf.Name, p.exitCode)

		if !p.isRestartable(p.exitCode) {
			break
		}
		p.restartsCounter += 1
		log.Info().Msgf("Restarting %s in %v second(s)... Restarts: %d",
			p.procConf.Name, p.getBackoff().Seconds(), p.restartsCounter)

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
		"PC_PROC_NAME=" + p.GetName(),
		"PC_REPLICA_NUM=1",
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
		return p.restartsCounter < p.procConf.RestartPolicy.MaxRestarts
	}

	if p.procConf.RestartPolicy.Restart == RestartPolicyAlways {
		if p.procConf.RestartPolicy.MaxRestarts == 0 {
			return true
		}
		return p.restartsCounter < p.procConf.RestartPolicy.MaxRestarts
	}

	return false
}

func (p *Process) WaitForCompletion(waitee string) int {
	p.Lock()
	defer p.Unlock()

	for !p.done {
		p.procCond.Wait()
	}
	return p.exitCode
}

func (p *Process) WontRun() {
	p.onProcessEnd()

}

func (p *Process) onProcessEnd() {
	if isStringDefined(p.procConf.LogLocation) {
		p.logger.Close()
	}
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
	fmt.Printf("[%s]\t%s\n", p.procColor(p.GetNameWithReplica()), p.noColor(message))
}

func (p *Process) handleError(message string) {
	p.logger.Error(message, p.GetName(), p.replica)
	fmt.Printf("[%s]\t%s\n", p.procColor(p.GetNameWithReplica()), p.redColor(message))
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
