package main

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

func (p *Project) Run() {
	p.runningProcesses = make(map[string]*Process)
	runOrder := []ProcessConfig{}
	p.WithServices([]string{}, func(process ProcessConfig) error {
		runOrder = append(runOrder, process)
		return nil
	})
	var nameOrder []string
	for _, v := range runOrder {
		nameOrder = append(nameOrder, v.Name)
	}
	var logger PcLogger = NewNilLogger("")
	if isStringDefined(p.LogLocation) {
		logger = NewLogger(p.LogLocation)
		defer logger.Close()
	}
	log.Debug().Msgf("Spinning up %d processes. Order: %q", len(runOrder), nameOrder)
	var wg sync.WaitGroup
	for _, proc := range runOrder {

		procLogger := logger
		if isStringDefined(proc.LogLocation) {
			procLogger = NewLogger(proc.LogLocation)
		}
		process := NewProcess(p.Environment, procLogger, proc, 1)
		p.addRunningProcess(process)
		wg.Add(1)
		go func() {
			defer p.removeRunningProcess(process.GetName())
			defer wg.Done()
			if err := p.WaitIfNeeded(process.procConf); err != nil {
				log.Error().Msgf("Error: %s", err.Error())
				log.Error().Msgf("Error: process %s won't run", process.GetName())
				process.WontRun()
			} else {
				process.Run()
			}
		}()
	}
	wg.Wait()
}

func (p *Project) WaitIfNeeded(process ProcessConfig) error {
	for k := range process.DependsOn {
		if runningProc := p.getRunningProcess(k); runningProc != nil {

			switch process.DependsOn[k].Condition {
			case ProcessConditionCompleted:
				runningProc.WaitForCompletion(process.Name)
			case ProcessConditionCompletedSuccessfully:
				log.Info().Msgf("%s is waiting for %s to complete successfully", process.Name, k)
				exitCode := runningProc.WaitForCompletion(process.Name)
				if exitCode != 0 {
					return fmt.Errorf("process %s depended on %s to complete successfully, but it exited with status %d",
						process.Name, k, exitCode)
				}
			}
		}
	}
	return nil
}

func (p *Project) addRunningProcess(process *Process) {
	p.mapMutex.Lock()
	p.runningProcesses[process.GetName()] = process
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

func (p *Project) GetProcesses(names ...string) ([]ProcessConfig, error) {
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
func (p *Project) WithServices(names []string, fn ProcessFunc) error {
	return p.withProcesses(names, fn, map[string]bool{})
}

func (p *Project) withProcesses(names []string, fn ProcessFunc, done map[string]bool) error {
	processes, err := p.GetProcesses(names...)
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
	err := p.WithServices([]string{}, func(process ProcessConfig) error {
		order = append(order, process.Name)
		return nil
	})
	return order, err
}
