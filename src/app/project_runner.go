package app

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

//var PROJ *ProjectRunner

type ProjectRunner struct {
	project          *types.Project
	runningProcesses map[string]*Process
	processStates    map[string]*types.ProcessState
	processLogs      map[string]*pclog.ProcessLogBuffer
	mapMutex         sync.Mutex
	logger           pclog.PcLogger
	waitGroup        sync.WaitGroup
	exitCode         int
}

func (p *ProjectRunner) GetLexicographicProcessNames() ([]string, error) {
	return p.project.GetLexicographicProcessNames()
}

func (p *ProjectRunner) WithProcesses(names []string, fn func(process types.ProcessConfig) error) error {
	return p.project.WithProcesses(names, fn)
}

func (p *ProjectRunner) init() {
	p.initProcessStates()
	p.initProcessLogs()
}

func (p *ProjectRunner) Run() int {
	p.runningProcesses = make(map[string]*Process)
	runOrder := []types.ProcessConfig{}
	_ = p.project.WithProcesses([]string{}, func(process types.ProcessConfig) error {
		runOrder = append(runOrder, process)
		return nil
	})
	var nameOrder []string
	for _, v := range runOrder {
		nameOrder = append(nameOrder, v.Name)
	}
	p.logger = pclog.NewNilLogger()
	if isStringDefined(p.project.LogLocation) {
		p.logger = pclog.NewLogger()
		p.logger.Open(p.project.LogLocation)
		defer p.logger.Close()
	}
	//zerolog.SetGlobalLevel(zerolog.PanicLevel)
	log.Debug().Msgf("Spinning up %d processes. Order: %q", len(runOrder), nameOrder)
	for _, proc := range runOrder {
		p.runProcess(proc)
	}
	p.waitGroup.Wait()
	log.Info().Msg("Project completed")
	return p.exitCode
}

func (p *ProjectRunner) runProcess(proc types.ProcessConfig) {
	procLogger := p.logger
	if isStringDefined(proc.LogLocation) {
		procLogger = pclog.NewLogger()
	}
	procLog, err := p.getProcessLog(proc.Name)
	if err != nil {
		// we shouldn't get here
		log.Error().Msgf("Error: Can't get log: %s using empty buffer", err.Error())
		procLog = pclog.NewLogBuffer(0)
	}
	procState, _ := p.GetProcessState(proc.Name)
	process := NewProcess(p.project.Environment, procLogger, proc, procState, procLog, 1, *p.project.ShellConfig)
	p.addRunningProcess(process)
	p.waitGroup.Add(1)
	go func() {
		defer p.removeRunningProcess(process.getName())
		defer p.waitGroup.Done()
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

func (p *ProjectRunner) waitIfNeeded(process types.ProcessConfig) error {
	for k := range process.DependsOn {
		if runningProc := p.getRunningProcess(k); runningProc != nil {

			switch process.DependsOn[k].Condition {
			case types.ProcessConditionCompleted:
				runningProc.waitForCompletion()
			case types.ProcessConditionCompletedSuccessfully:
				log.Info().Msgf("%s is waiting for %s to complete successfully", process.Name, k)
				exitCode := runningProc.waitForCompletion()
				if exitCode != 0 {
					return fmt.Errorf("process %s depended on %s to complete successfully, but it exited with status %d",
						process.Name, k, exitCode)
				}
			case types.ProcessConditionHealthy:
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

func (p *ProjectRunner) onProcessEnd(exitCode int, procConf types.ProcessConfig) {
	if (exitCode != 0 && procConf.RestartPolicy.Restart == types.RestartPolicyExitOnFailure) ||
		procConf.RestartPolicy.ExitOnEnd {
		p.ShutDownProject()
		p.exitCode = exitCode
	}
}

func (p *ProjectRunner) initProcessStates() {
	p.processStates = make(map[string]*types.ProcessState)
	for key, proc := range p.project.Processes {
		p.processStates[key] = &types.ProcessState{
			Name:       key,
			Namespace:  proc.Namespace,
			Status:     types.ProcessStatePending,
			SystemTime: "",
			Health:     types.ProcessHealthUnknown,
			Restarts:   0,
			ExitCode:   0,
			Pid:        0,
		}
		if proc.Disabled {
			p.processStates[key].Status = types.ProcessStateDisabled
		}
	}
}

func (p *ProjectRunner) initProcessLogs() {
	p.processLogs = make(map[string]*pclog.ProcessLogBuffer)
	for key := range p.project.Processes {
		p.processLogs[key] = pclog.NewLogBuffer(p.project.LogLength)
	}
}

func (p *ProjectRunner) GetProcessState(name string) (*types.ProcessState, error) {
	if procState, ok := p.processStates[name]; ok {
		proc := p.getRunningProcess(name)
		if proc != nil {
			proc.updateProcState()
		} else {
			procState.Pid = 0
			procState.SystemTime = ""
			procState.Age = time.Duration(0)
			procState.Health = types.ProcessHealthUnknown
			procState.IsRunning = false
		}
		return procState, nil
	}

	log.Error().Msgf("Error: process %s doesn't exist", name)
	return nil, fmt.Errorf("no such process: %s", name)
}

func (p *ProjectRunner) GetProcessesState() (*types.ProcessesState, error) {
	states := &types.ProcessesState{
		States: make([]types.ProcessState, 0),
	}
	for name, _ := range p.processStates {
		state, err := p.GetProcessState(name)
		if err != nil {
			continue
		}
		states.States = append(states.States, *state)
	}
	return states, nil
}

func (p *ProjectRunner) addRunningProcess(process *Process) {
	p.mapMutex.Lock()
	p.runningProcesses[process.getName()] = process
	p.mapMutex.Unlock()
}

func (p *ProjectRunner) getRunningProcess(name string) *Process {
	p.mapMutex.Lock()
	defer p.mapMutex.Unlock()
	if runningProc, ok := p.runningProcesses[name]; ok {
		return runningProc
	}
	return nil
}

func (p *ProjectRunner) removeRunningProcess(name string) {
	p.mapMutex.Lock()
	delete(p.runningProcesses, name)
	p.mapMutex.Unlock()
}

func (p *ProjectRunner) StartProcess(name string) error {
	proc := p.getRunningProcess(name)
	if proc != nil {
		log.Error().Msgf("Process %s is already running", name)
		return fmt.Errorf("process %s is already running", name)
	}
	if processConfig, ok := p.project.Processes[name]; ok {
		processConfig.Name = name
		p.runProcess(processConfig)
	} else {
		return fmt.Errorf("no such process: %s", name)
	}

	return nil
}

func (p *ProjectRunner) StopProcess(name string) error {
	proc := p.getRunningProcess(name)
	if proc == nil {
		log.Error().Msgf("Process %s is not running", name)
		return fmt.Errorf("process %s is not running", name)
	}
	_ = proc.shutDown()
	return nil
}

func (p *ProjectRunner) RestartProcess(name string) error {
	proc := p.getRunningProcess(name)
	if proc != nil {
		_ = proc.shutDown()
		if proc.isRestartable() {
			return nil
		}
		time.Sleep(proc.getBackoff())
	}

	if processConfig, ok := p.project.Processes[name]; ok {
		processConfig.Name = name
		p.runProcess(processConfig)
	} else {
		return fmt.Errorf("no such process: %s", name)
	}
	return nil
}

func (p *ProjectRunner) GetProcessInfo(name string) (*types.ProcessConfig, error) {
	if processConfig, ok := p.project.Processes[name]; ok {
		processConfig.Name = name
		return &processConfig, nil
	} else {
		return nil, fmt.Errorf("no such process: %s", name)
	}
}

func (p *ProjectRunner) ShutDownProject() {
	p.mapMutex.Lock()
	defer p.mapMutex.Unlock()
	runProc := p.runningProcesses
	for _, proc := range runProc {
		proc.prepareForShutDown()
	}
	wg := sync.WaitGroup{}
	for _, proc := range runProc {
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
	wg.Wait()
}

func (p *ProjectRunner) IsRemote() bool {
	return false
}

func (p *ProjectRunner) ErrorForSecs() int {
	return 0
}

func (p *ProjectRunner) GetHostName() (string, error) {
	return os.Hostname()
}

func (p *ProjectRunner) getProcessLog(name string) (*pclog.ProcessLogBuffer, error) {
	if procLogs, ok := p.processLogs[name]; ok {
		return procLogs, nil
	}
	log.Error().Msgf("Error: process %s doesn't exist", name)
	return nil, fmt.Errorf("process %s doesn't exist", name)
}

func (p *ProjectRunner) GetProcessLog(name string, offsetFromEnd, limit int) ([]string, error) {
	logs, err := p.getProcessLog(name)
	if err != nil {
		return nil, err
	}
	return logs.GetLogRange(offsetFromEnd, limit), nil
}

func (p *ProjectRunner) GetProcessLogLine(name string, lineIndex int) (string, error) {
	logs, err := p.getProcessLog(name)
	if err != nil {
		return "", err
	}
	return logs.GetLogLine(lineIndex), nil
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

func (p *ProjectRunner) selectRunningProcesses(procList []string) error {
	if len(procList) == 0 {
		return nil
	}
	newProcMap := types.Processes{}
	err := p.project.WithProcesses(procList, func(process types.ProcessConfig) error {
		newProcMap[process.Name] = process
		return nil
	})
	if err != nil {
		log.Err(err).Msgf("Failed select processes")
		return err
	}
	p.project.Processes = newProcMap
	return nil
}

func (p *ProjectRunner) selectRunningProcessesNoDeps(procList []string) error {
	if len(procList) == 0 {
		return nil
	}
	newProcMap := types.Processes{}
	for _, procName := range procList {
		if conf, ok := p.project.Processes[procName]; ok {
			conf.DependsOn = types.DependsOnConfig{}
			newProcMap[procName] = conf
		} else {
			err := fmt.Errorf("no such process: %s", procName)
			log.Err(err).Msgf("Failed select processes")
			return err
		}
	}
	p.project.Processes = newProcMap
	return nil
}

func (p *ProjectRunner) GetLogLength() int {
	return p.project.LogLength
}

func (p *ProjectRunner) GetDependenciesOrderNames() ([]string, error) {
	return p.project.GetDependenciesOrderNames()
}

func (p *ProjectRunner) GetProject() *types.Project {
	return p.project
}

func NewProjectRunner(project *types.Project, processesToRun []string, noDeps bool) (*ProjectRunner, error) {

	runner := &ProjectRunner{
		project: project,
	}

	var err error
	if noDeps {
		err = runner.selectRunningProcessesNoDeps(processesToRun)
	} else {
		err = runner.selectRunningProcesses(processesToRun)
	}
	if err != nil {
		return nil, err
	}
	//PROJ = runner
	runner.init()
	return runner, nil
}
