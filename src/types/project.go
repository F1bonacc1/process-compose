package types

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/command"
	"sort"
)

type Project struct {
	Version     string               `yaml:"version"`
	LogLocation string               `yaml:"log_location,omitempty"`
	LogLevel    string               `yaml:"log_level,omitempty"`
	LogLength   int                  `yaml:"log_length,omitempty"`
	Processes   Processes            `yaml:"processes"`
	Environment Environment          `yaml:"environment,omitempty"`
	ShellConfig *command.ShellConfig `yaml:"shell,omitempty"`
}

type ProcessFunc func(process ProcessConfig) error

// WithProcesses run ProcesseFunc on each Process and dependencies in dependency order
func (p *Project) WithProcesses(names []string, fn ProcessFunc) error {
	return p.withProcesses(names, fn, map[string]bool{})
}

func (p *Project) GetDependenciesOrderNames() ([]string, error) {

	order := []string{}
	err := p.WithProcesses([]string{}, func(process ProcessConfig) error {
		order = append(order, process.Name)
		return nil
	})
	return order, err
}

func (p *Project) GetLexicographicProcessNames() ([]string, error) {

	names := []string{}
	for name := range p.Processes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
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
