package app

import (
	"errors"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

const (
	DEFAULT_LOG_LENGTH = 1000
)

var PROJ *Project

func (p *Project) init() {
	p.initProcessStates()
	p.initProcessLogs()
	p.deprecationCheck()
}

func (p *Project) Run() int {
	p.runningProcesses = make(map[string]*Process)
	runOrder := []ProcessConfig{}
	_ = p.WithProcesses([]string{}, func(process ProcessConfig) error {
		runOrder = append(runOrder, process)
		return nil
	})
	var nameOrder []string
	for _, v := range runOrder {
		nameOrder = append(nameOrder, v.Name)
	}
	p.logger = pclog.NewNilLogger()
	if isStringDefined(p.LogLocation) {
		p.logger = pclog.NewLogger(p.LogLocation)
		defer p.logger.Close()
	}
	//zerolog.SetGlobalLevel(zerolog.PanicLevel)
	log.Debug().Msgf("Spinning up %d processes. Order: %q", len(runOrder), nameOrder)
	for _, proc := range runOrder {
		p.runProcess(proc)
	}
	p.wg.Wait()
	log.Info().Msg("Project completed")
	return p.exitCode
}

func (p *Project) runProcess(proc ProcessConfig) {
	procLogger := p.logger
	if isStringDefined(proc.LogLocation) {
		procLogger = pclog.NewLogger(proc.LogLocation)
	}
	procLog, err := p.getProcessLog(proc.Name)
	if err != nil {
		// we shouldn't get here
		log.Error().Msgf("Error: Can't get log: %s using empty buffer", err.Error())
		procLog = pclog.NewLogBuffer(0)
	}
	process := NewProcess(p.Environment, procLogger, proc, p.GetProcessState(proc.Name), procLog, 1)
	p.addRunningProcess(process)
	p.wg.Add(1)
	go func() {
		defer p.removeRunningProcess(process.getName())
		defer p.wg.Done()
		if err := p.waitIfNeeded(process.procConf); err != nil {
			log.Error().Msgf("Error: %s", err.Error())
			log.Error().Msgf("Error: process %s won't run", process.getName())
			process.wontRun()
		} else {
			exitCode := process.run()
			p.onProcessEnd(exitCode, process.procConf)
		}
	}()
}

func (p *Project) waitIfNeeded(process ProcessConfig) error {
	for k := range process.DependsOn {
		if runningProc := p.getRunningProcess(k); runningProc != nil {

			switch process.DependsOn[k].Condition {
			case ProcessConditionCompleted:
				runningProc.waitForCompletion()
			case ProcessConditionCompletedSuccessfully:
				log.Info().Msgf("%s is waiting for %s to complete successfully", process.Name, k)
				exitCode := runningProc.waitForCompletion()
				if exitCode != 0 {
					return fmt.Errorf("process %s depended on %s to complete successfully, but it exited with status %d",
						process.Name, k, exitCode)
				}
			case ProcessConditionHealthy:
				log.Info().Msgf("%s is waiting for %s to be healthy", process.Name, k)
				ready := runningProc.waitUntilReady()
				if !ready {
					return fmt.Errorf("process %s depended on %s to become ready, but it was terminated", process.Name, k)
				}

			}
		}
	}
	return nil
}

func (p *Project) onProcessEnd(exitCode int, procConf ProcessConfig) {
	if exitCode != 0 && procConf.RestartPolicy.Restart == RestartPolicyExitOnFailure {
		p.ShutDownProject()
		p.exitCode = exitCode
	}
}

func (p *Project) initProcessStates() {
	p.processStates = make(map[string]*ProcessState)
	for key, proc := range p.Processes {
		p.processStates[key] = &ProcessState{
			Name:       key,
			Status:     ProcessStatePending,
			SystemTime: "",
			Health:     ProcessHealthUnknown,
			Restarts:   0,
			ExitCode:   0,
			Pid:        0,
		}
		if proc.Disabled {
			p.processStates[key].Status = ProcessStateDisabled
		}
	}
}

func (p *Project) initProcessLogs() {
	p.processLogs = make(map[string]*pclog.ProcessLogBuffer)
	for key := range p.Processes {
		p.processLogs[key] = pclog.NewLogBuffer(p.LogLength)
	}
}

func (p *Project) deprecationCheck() {
	for key, proc := range p.Processes {
		if proc.RestartPolicy.Restart == RestartPolicyOnFailureDeprecated {
			deprecationHandler("2022-10-30", key, RestartPolicyOnFailureDeprecated, RestartPolicyOnFailure, "restart policy")
		}
	}
}

func (p *Project) GetProcessState(name string) *ProcessState {
	if procState, ok := p.processStates[name]; ok {
		proc := p.getRunningProcess(name)
		if proc != nil {
			proc.updateProcState()
		} else {
			procState.Pid = 0
			procState.SystemTime = ""
			procState.Health = ProcessHealthUnknown
		}
		return procState
	}

	log.Error().Msgf("Error: process %s doesn't exist", name)
	return nil
}

func (p *Project) addRunningProcess(process *Process) {
	p.mapMutex.Lock()
	p.runningProcesses[process.getName()] = process
	p.mapMutex.Unlock()
}

func (p *Project) getRunningProcess(name string) *Process {
	p.mapMutex.Lock()
	defer p.mapMutex.Unlock()
	if runningProc, ok := p.runningProcesses[name]; ok {
		return runningProc
	}
	return nil
}

func (p *Project) removeRunningProcess(name string) {
	p.mapMutex.Lock()
	delete(p.runningProcesses, name)
	p.mapMutex.Unlock()
}

func (p *Project) StartProcess(name string) error {
	proc := p.getRunningProcess(name)
	if proc != nil {
		log.Error().Msgf("Process %s is already running", name)
		return fmt.Errorf("process %s is already running", name)
	}
	if processConfig, ok := p.Processes[name]; ok {
		processConfig.Name = name
		p.runProcess(processConfig)
	} else {
		return fmt.Errorf("no such process: %s", name)
	}

	return nil
}

func (p *Project) StopProcess(name string) error {
	proc := p.getRunningProcess(name)
	if proc == nil {
		log.Error().Msgf("Process %s is not running", name)
		return fmt.Errorf("process %s is not running", name)
	}
	_ = proc.shutDown()
	return nil
}

func (p *Project) RestartProcess(name string) error {
	proc := p.getRunningProcess(name)
	if proc != nil {
		_ = proc.shutDown()
		if proc.isRestartable() {
			return nil
		}
		time.Sleep(proc.getBackoff())
	}

	if processConfig, ok := p.Processes[name]; ok {
		processConfig.Name = name
		p.runProcess(processConfig)
	} else {
		return fmt.Errorf("no such process: %s", name)
	}
	return nil
}

func (p *Project) ShutDownProject() {
	p.mapMutex.Lock()
	defer p.mapMutex.Unlock()
	runProc := p.runningProcesses
	for _, proc := range runProc {
		proc.prepareForShutDown()
	}
	for _, proc := range runProc {
		_ = proc.shutDown()
	}
}

func (p *Project) getProcessLog(name string) (*pclog.ProcessLogBuffer, error) {
	if procLogs, ok := p.processLogs[name]; ok {
		return procLogs, nil
	}
	log.Error().Msgf("Error: process %s doesn't exist", name)
	return nil, fmt.Errorf("process %s doesn't exist", name)
}

func (p *Project) GetProcessLog(name string, offsetFromEnd, limit int) ([]string, error) {
	logs, err := p.getProcessLog(name)
	if err != nil {
		return nil, err
	}
	return logs.GetLogRange(offsetFromEnd, limit), nil
}

func (p *Project) GetProcessLogLine(name string, lineIndex int) (string, error) {
	logs, err := p.getProcessLog(name)
	if err != nil {
		return "", err
	}
	return logs.GetLogLine(lineIndex), nil
}

func (p *Project) GetProcessLogLength(name string) int {
	logs, err := p.getProcessLog(name)
	if err != nil {
		return 0
	}
	return logs.GetLogLength()
}

func (p *Project) GetLogsAndSubscribe(name string, observer pclog.PcLogObserver) {

	logs, err := p.getProcessLog(name)
	if err != nil {
		return
	}
	logs.GetLogsAndSubscribe(observer)
}

func (p *Project) UnSubscribeLogger(name string) {
	logs, err := p.getProcessLog(name)
	if err != nil {
		return
	}
	logs.UnSubscribe()
}

func (p *Project) getProcesses(names ...string) ([]ProcessConfig, error) {
	processes := []ProcessConfig{}
	if len(names) == 0 {
		for name, proc := range p.Processes {
			if proc.Disabled {
				continue
			}
			proc.Name = name
			processes = append(processes, proc)
		}
		return processes, nil
	}
	for _, name := range names {
		if proc, ok := p.Processes[name]; ok {
			if proc.Disabled {
				continue
			}
			proc.Name = name
			processes = append(processes, proc)
		} else {
			return processes, fmt.Errorf("no such process: %s", name)
		}
	}

	return processes, nil
}

type ProcessFunc func(process ProcessConfig) error

// WithProcesses run ProcesseFunc on each Process and dependencies in dependency order
func (p *Project) WithProcesses(names []string, fn ProcessFunc) error {
	return p.withProcesses(names, fn, map[string]bool{})
}

func (p *Project) withProcesses(names []string, fn ProcessFunc, done map[string]bool) error {
	processes, err := p.getProcesses(names...)
	if err != nil {
		return err
	}
	for _, process := range processes {
		if done[process.Name] {
			continue
		}
		done[process.Name] = true

		dependencies := process.GetDependencies()
		if len(dependencies) > 0 {
			err := p.withProcesses(dependencies, fn, done)
			if err != nil {
				return err
			}
		}
		if err := fn(process); err != nil {
			return err
		}
	}
	return nil
}

func (p *Project) GetDependenciesOrderNames() ([]string, error) {

	order := []string{}
	err := p.WithProcesses([]string{}, func(process ProcessConfig) error {
		order = append(order, process.Name)
		return nil
	})
	return order, err
}

func (p *Project) GetLexicographicProcessNames() []string {

	names := []string{}
	for name := range p.Processes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func NewProject(inputFile string) *Project {
	yamlFile, err := os.ReadFile(inputFile)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Error().Msgf("File %s doesn't exist", inputFile)
		}
		log.Fatal().Msg(err.Error())
	}

	// .env is optional we don't care if it errors
	_ = godotenv.Load()

	yamlFile = []byte(os.ExpandEnv(string(yamlFile)))

	project := Project{
		LogLength: DEFAULT_LOG_LENGTH,
		exitCode:  0,
	}
	err = yaml.Unmarshal(yamlFile, &project)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	if project.LogLevel != "" {
		lvl, err := zerolog.ParseLevel(project.LogLevel)
		if err != nil {
			log.Error().Msgf("Unknown log level %s defaulting to %s",
				project.LogLevel, zerolog.GlobalLevel().String())
		} else {
			zerolog.SetGlobalLevel(lvl)
		}

	}
	PROJ = &project
	project.init()
	return &project
}

func findFiles(names []string, pwd string) []string {
	candidates := []string{}
	for _, n := range names {
		f := filepath.Join(pwd, n)
		if _, err := os.Stat(f); err == nil {
			candidates = append(candidates, f)
		}
	}
	return candidates
}

// DefaultFileNames defines the Compose file names for auto-discovery (in order of preference)
var DefaultFileNames = []string{"compose.yml", "compose.yaml", "process-compose.yml", "process-compose.yaml"}

func AutoDiscoverComposeFile(pwd string) (string, error) {
	candidates := findFiles(DefaultFileNames, pwd)
	if len(candidates) > 0 {
		winner := candidates[0]
		if len(candidates) > 1 {
			log.Warn().Msgf("Found multiple config files with supported names: %s", strings.Join(candidates, ", "))
			log.Warn().Msgf("Using %s", winner)
		}
		return winner, nil
	}
	return "", fmt.Errorf("no config files found in %s", pwd)
}
