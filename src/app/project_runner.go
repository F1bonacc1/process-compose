package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/templater"
	"github.com/f1bonacc1/process-compose/src/types"
	"os"
	"os/user"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("project non-zero exit code: %d", e.Code)
}

type ProjectRunner struct {
	procConfMutex     sync.Mutex
	project           *types.Project
	logsMutex         sync.Mutex
	processLogs       map[string]*pclog.ProcessLogBuffer
	statesMutex       sync.Mutex
	processStates     map[string]*types.ProcessState
	runProcMutex      sync.Mutex
	runningProcesses  map[string]*Process
	doneProcMutex     sync.Mutex
	doneProcesses     map[string]*Process
	logger            pclog.PcLogger
	waitGroup         sync.WaitGroup
	exitCode          int
	projectState      *types.ProjectState
	mainProcess       string
	mainProcessArgs   []string
	isTuiOn           bool
	isOrderedShutDown bool
	ctxApp            context.Context
	cancelAppFn       context.CancelFunc
	disableDotenv     bool
}

func (p *ProjectRunner) GetLexicographicProcessNames() ([]string, error) {
	return p.project.GetLexicographicProcessNames()
}

func (p *ProjectRunner) init() {
	p.initProcessStates()
	p.initProcessLogs()
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
		return fmt.Errorf("failed to build project run order: %e", err)
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
	for _, proc := range runOrder {
		newConf := proc
		p.runProcess(&newConf)
	}
	p.waitGroup.Wait()
	log.Info().Msg("Project completed")
	if p.exitCode != 0 {
		err = &ExitError{p.exitCode}
	}
	return err
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
	isMain := config.Name == p.mainProcess
	hasMain := p.mainProcess != ""
	printLogs := !hasMain && !p.isTuiOn
	extraArgs := []string{}
	if isMain {
		extraArgs = p.mainProcessArgs
		config.RestartPolicy.ExitOnEnd = true
	}
	process := NewProcess(
		withTuiOn(p.isTuiOn),
		withGlobalEnv(p.project.Environment),
		withLogger(procLogger),
		withProcConf(config),
		withProcState(procState),
		withProcLog(procLog),
		withShellConfig(*p.project.ShellConfig),
		withPrintLogs(printLogs),
		withIsMain(isMain),
		withExtraArgs(extraArgs),
	)
	p.addRunningProcess(process)
	p.waitGroup.Add(1)
	go func(proc *Process) {
		defer p.removeRunningProcess(proc)
		defer p.waitGroup.Done()
		if err = p.waitIfNeeded(proc.procConf); err != nil {
			log.Error().Msgf("Error: %s", err.Error())
			log.Error().Msgf("Error: process %s won't run", proc.getName())
			proc.wontRun()
			p.onProcessSkipped(proc.procConf)
		} else {
			exitCode := proc.run()
			p.addDoneProcess(proc)
			p.onProcessEnd(exitCode, proc.procConf)
		}
	}(process)
}

func (p *ProjectRunner) waitIfNeeded(process *types.ProcessConfig) error {
	for k := range process.DependsOn {
		if proc := p.getDoneOrRunningProcess(k); proc != nil {
			switch process.DependsOn[k].Condition {
			case types.ProcessConditionCompleted:
				proc.waitForCompletion()
			case types.ProcessConditionCompletedSuccessfully:
				log.Info().Msgf("%s is waiting for %s to complete successfully", process.ReplicaName, k)
				exitCode := proc.waitForCompletion()
				if exitCode != 0 {
					return fmt.Errorf("process %s depended on %s to complete successfully, but it exited with status %d",
						process.ReplicaName, k, exitCode)
				}
			case types.ProcessConditionHealthy:
				log.Info().Msgf("%s is waiting for %s to be healthy", process.ReplicaName, k)
				ready := proc.waitUntilReady()
				if !ready {
					return fmt.Errorf("process %s depended on %s to become ready, but it was terminated", process.ReplicaName, k)
				}
			case types.ProcessConditionLogReady:
				log.Info().Msgf("%s is waiting for %s log line %s", process.ReplicaName, k, proc.procConf.ReadyLogLine)
				ready := proc.waitUntilLogReady()
				if !ready {
					return fmt.Errorf("process %s depended on %s to become ready, but it was terminated", process.ReplicaName, k)
				}
			case types.ProcessConditionStarted:
				log.Info().Msgf("%s is waiting for %s to start", process.ReplicaName, k)
				proc.waitForStarted()
			}
		} else {
			log.Error().Msgf("Error: process %s depends on %s, but it isn't running or completed", process.ReplicaName, k)
		}

	}
	return nil
}

func (p *ProjectRunner) onProcessEnd(exitCode int, procConf *types.ProcessConfig) {
	if (exitCode != 0 && procConf.RestartPolicy.Restart == types.RestartPolicyExitOnFailure) ||
		procConf.RestartPolicy.ExitOnEnd {
		_ = p.ShutDownProject()
		p.exitCode = exitCode
	}
}

func (p *ProjectRunner) onProcessSkipped(procConf *types.ProcessConfig) {
	if procConf.RestartPolicy.ExitOnSkipped {
		_ = p.ShutDownProject()
		p.exitCode = 1
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

func (p *ProjectRunner) addDoneProcess(process *Process) {
	p.doneProcMutex.Lock()
	p.doneProcesses[process.getName()] = process
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
	if doneProc := p.getDoneProcess(name); doneProc != nil {
		return doneProc
	}
	return p.getRunningProcess(name)
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
		if _, ok := p.project.Processes[name]; !ok {
			log.Error().Msgf("Process %s does not exist", name)
			return fmt.Errorf("process %s does not exist", name)
		}
		log.Error().Msgf("Process %s is not running", name)
		return fmt.Errorf("process %s is not running", name)
	}
	err := proc.shutDownNoRestart()
	if err != nil {
		log.Err(err).Msgf("failed to stop process %s", name)
	}
	return err
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

func (p *ProjectRunner) RestartProcess(name string) error {
	log.Debug().Msgf("Restarting %s", name)
	proc := p.getRunningProcess(name)
	if proc != nil {
		err := proc.shutDownNoRestart()
		if err != nil {
			log.Err(err).Msgf("failed to stop process %s", name)
			return err
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

func (p *ProjectRunner) SetProcessPassword(name, pass string) error {
	p.runProcMutex.Lock()

	var wg sync.WaitGroup
	for _, process := range p.runningProcesses {
		if process.procConf.IsElevated && !process.passProvided {
			wg.Add(1)
			go func(process *Process) {
				defer wg.Done()
				err := process.setPassword(pass)
				if err != nil {
					log.Err(err).Msgf("failed to set password for elevated process %s", process.getName())
				}
			}(process)
		}
	}
	p.runProcMutex.Unlock()
	wg.Wait()
	p.runProcMutex.Lock()
	defer p.runProcMutex.Unlock()
	for _, process := range p.runningProcesses {
		if process.procConf.IsElevated && process.passProvided {
			return nil
		}
	}

	return errors.New("password not accepted")

}

func (p *ProjectRunner) runningProcessesReverseDependencies() map[string]map[string]*Process {
	reverseDependencies := make(map[string]map[string]*Process)

	// `p.runProcMutex` lock is assumed to have been acquired when calling
	// this function. It is currently called by `ShutDownProject()`.
	for _, process := range p.runningProcesses {
		for k := range process.procConf.DependsOn {
			if runningProc, ok := p.runningProcesses[k]; ok {
				if _, ok := reverseDependencies[runningProc.getName()]; !ok {
					dep := make(map[string]*Process)
					dep[process.getName()] = process
					reverseDependencies[runningProc.getName()] = dep
				}
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
	if p.isOrderedShutDown {
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
	p.runProcMutex.Lock()
	defer p.runProcMutex.Unlock()

	shutdownOrder := []*Process{}
	if p.isOrderedShutDown {
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

func (p *ProjectRunner) GetHostName() (string, error) {
	return p.projectState.HostName, nil
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
	for i := 0; i < toAdd; i++ {
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
	procConf, ok := p.project.Processes[name]
	if ok {
		delete(p.project.Processes, name)
		procConf.ReplicaName = newName
		p.project.Processes[newName] = procConf
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
	if !proc.IsDeferred() {
		p.runProcess(&proc)
	}
}

func (p *ProjectRunner) selectRunningProcesses(procList []string) error {
	if len(procList) == 0 {
		return nil
	}
	newProcMap := types.Processes{}
	err := p.project.WithProcesses(procList, func(process types.ProcessConfig) error {
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
	for name, proc := range p.project.Processes {
		if _, ok := newProcMap[name]; !ok {
			proc.Disabled = true
		} else {
			proc.Disabled = false
		}
		p.project.Processes[name] = proc
	}
	return nil
}

func (p *ProjectRunner) selectRunningProcessesNoDeps(procList []string) error {
	if len(procList) == 0 {
		return nil
	}
	for name, proc := range p.project.Processes {
		found := false
		for _, procName := range procList {
			if proc.Name == procName {
				found = true
				break
			}
		}
		if !found {
			proc.Disabled = true
		} else {
			proc.DependsOn = types.DependsOnConfig{}
			proc.Disabled = false
		}
		p.project.Processes[name] = proc
	}

	return nil
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

	hostname, err := os.Hostname()
	if err != nil {
		log.Err(err).Msg("Failed get hostname")
		hostname = "unknown"
	}
	current, err := user.Current()
	username := "unknown-user"
	if err != nil {
		log.Err(err).Msg("Failed get user")
	} else {
		username = current.Username
	}
	runner := &ProjectRunner{
		project:           opts.project,
		mainProcess:       opts.mainProcess,
		mainProcessArgs:   opts.mainProcessArgs,
		isTuiOn:           opts.isTuiOn,
		isOrderedShutDown: opts.isOrderedShutDown,
		disableDotenv:     opts.disableDotenv,
		projectState: &types.ProjectState{
			FileNames: opts.project.FileNames,
			StartTime: time.Now(),
			UserName:  username,
			HostName:  hostname,
			Version:   config.Version,
		},
	}

	if opts.noDeps {
		err = runner.selectRunningProcessesNoDeps(opts.processesToRun)
	} else {
		err = runner.selectRunningProcesses(opts.processesToRun)
	}
	if err != nil {
		return nil, err
	}
	runner.projectState.ProcessNum = len(runner.project.Processes)
	runner.init()
	runner.ctxApp, runner.cancelAppFn = context.WithCancel(context.Background())
	return runner, nil
}

func (p *ProjectRunner) UpdateProject(project *types.Project) (map[string]string, error) {
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
