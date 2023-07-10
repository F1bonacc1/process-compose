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
	procConfMutex    sync.Mutex
	project          *types.Project
	logsMutex        sync.Mutex
	processLogs      map[string]*pclog.ProcessLogBuffer
	statesMutex      sync.Mutex
	processStates    map[string]*types.ProcessState
	runProcMutex     sync.Mutex
	runningProcesses map[string]*Process
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
		nameOrder = append(nameOrder, v.ReplicaName)
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
		newConf := proc
		p.runProcess(&newConf)
	}
	p.waitGroup.Wait()
	log.Info().Msg("Project completed")
	return p.exitCode
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
	procState, _ := p.GetProcessState(config.ReplicaName)
	process := NewProcess(p.project.Environment, procLogger, config, procState, procLog, *p.project.ShellConfig)
	p.addRunningProcess(process)
	p.waitGroup.Add(1)
	go func() {
		defer p.removeRunningProcess(process)
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

func (p *ProjectRunner) waitIfNeeded(process *types.ProcessConfig) error {
	for k := range process.DependsOn {
		if runningProc := p.getRunningProcess(k); runningProc != nil {

			switch process.DependsOn[k].Condition {
			case types.ProcessConditionCompleted:
				runningProc.waitForCompletion()
			case types.ProcessConditionCompletedSuccessfully:
				log.Info().Msgf("%s is waiting for %s to complete successfully", process.ReplicaName, k)
				exitCode := runningProc.waitForCompletion()
				if exitCode != 0 {
					return fmt.Errorf("process %s depended on %s to complete successfully, but it exited with status %d",
						process.ReplicaName, k, exitCode)
				}
			case types.ProcessConditionHealthy:
				log.Info().Msgf("%s is waiting for %s to be healthy", process.ReplicaName, k)
				ready := runningProc.waitUntilReady()
				if !ready {
					return fmt.Errorf("process %s depended on %s to become ready, but it was terminated", process.ReplicaName, k)
				}

			}
		}
	}
	return nil
}

func (p *ProjectRunner) onProcessEnd(exitCode int, procConf *types.ProcessConfig) {
	if exitCode != 0 && procConf.RestartPolicy.Restart == types.RestartPolicyExitOnFailure {
		p.ShutDownProject()
		p.exitCode = exitCode
	}
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

func (p *ProjectRunner) initProcessLog(name string) {
	p.processLogs[name] = pclog.NewLogBuffer(p.project.LogLength)
}

func (p *ProjectRunner) GetProcessState(name string) (*types.ProcessState, error) {
	proc := p.getRunningProcess(name)
	if proc != nil {
		return proc.getState(), nil
	} else {
		p.statesMutex.Lock()
		defer p.statesMutex.Unlock()
		state, ok := p.processStates[name]
		if !ok {
			log.Error().Msgf("Error: process %s doesn't exist", name)
			return nil, fmt.Errorf("can't get state of process %s: no such process", name)
		}
		return state, nil
	}
}

func (p *ProjectRunner) GetProcessesState() (*types.ProcessesState, error) {
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

func (p *ProjectRunner) addRunningProcess(process *Process) {
	p.runProcMutex.Lock()
	p.runningProcesses[process.getName()] = process
	p.runProcMutex.Unlock()
}

func (p *ProjectRunner) getRunningProcess(name string) *Process {
	p.runProcMutex.Lock()
	defer p.runProcMutex.Unlock()
	if runningProc, ok := p.runningProcesses[name]; ok {
		return runningProc
	}
	return nil
}

func (p *ProjectRunner) removeRunningProcess(process *Process) {
	p.runProcMutex.Lock()
	delete(p.runningProcesses, process.getName())
	p.runProcMutex.Unlock()
}

func (p *ProjectRunner) StartProcess(name string) error {
	proc := p.getRunningProcess(name)
	if proc != nil {
		log.Error().Msgf("Process %s is already running", name)
		return fmt.Errorf("process %s is already running", name)
	}
	if processConfig, ok := p.project.Processes[name]; ok {
		p.runProcess(&processConfig)
	} else {
		return fmt.Errorf("no such process: %s", name)
	}

	return nil
}

func (p *ProjectRunner) StopProcess(name string) error {
	log.Info().Msgf("Stopping %s", name)
	proc := p.getRunningProcess(name)
	if proc == nil {
		log.Error().Msgf("Process %s is not running", name)
		return fmt.Errorf("process %s is not running", name)
	}
	err := proc.shutDown()
	if err != nil {
		log.Err(err).Msgf("failed to stop process %s", name)
	}
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

func (p *ProjectRunner) ShutDownProject() {
	p.runProcMutex.Lock()
	defer p.runProcMutex.Unlock()
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

func (p *ProjectRunner) ScaleProcess(name string, scale int) error {
	if scale < 1 {
		err := fmt.Errorf("cannot scale process %s to a negative or zero value %d", name, scale)
		log.Err(err).Msg("scale failed")
		return err
	}
	if processConfig, ok := p.project.Processes[name]; ok {
		scaleDelta := scale - processConfig.Replicas
		if scaleDelta < 0 {
			log.Info().Msgf("scaling down %s by %d", name, scaleDelta*-1)
			p.scaleDownProcess(processConfig.Name, scale)
		} else if scaleDelta > 0 {
			log.Info().Msgf("scaling up %s by %d", name, scaleDelta)
			p.scaleUpProcess(processConfig, scaleDelta, scale)
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

func (p *ProjectRunner) scaleUpProcess(proc types.ProcessConfig, toAdd, scale int) {
	origScale := proc.Replicas
	for i := 0; i < toAdd; i++ {
		proc.ReplicaNum = origScale + i
		proc.Replicas = scale
		proc.ReplicaName = proc.CalculateReplicaName()
		p.addProcessAndRun(proc)
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
		p.removeRunningProcess(process)
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
	config, ok := p.project.Processes[name]
	if ok {
		delete(p.project.Processes, name)
		config.ReplicaName = newName
		p.project.Processes[newName] = config
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
	p.project.Processes[proc.ReplicaName] = proc
	p.initProcessLog(proc.ReplicaName)
	p.runProcess(&proc)
}

func (p *ProjectRunner) selectRunningProcesses(procList []string) error {
	if len(procList) == 0 {
		return nil
	}
	newProcMap := types.Processes{}
	err := p.project.WithProcesses(procList, func(process types.ProcessConfig) error {
		newProcMap[process.ReplicaName] = process
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

//func getProcessName(process *Process) string {
//	return process.getNameWithSmartReplica()
//}
//
//func getProcessNameFromConf(process types.ProcessConfig, replica int) string {
//	if process.Replicas > 1 {
//		return fmt.Sprintf("%s-%d", process.Name, replica)
//	}
//	return process.Name
//}
