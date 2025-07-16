package types

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/command"
	"sort"
)

type Vars map[string]any

type Project struct {
	Version             string               `yaml:"version"`
	Name                string               `yaml:"name,omitempty"`
	LogLocation         string               `yaml:"log_location,omitempty"`
	LogLevel            string               `yaml:"log_level,omitempty"`
	LogLength           int                  `yaml:"log_length,omitempty"`
	LoggerConfig        *LoggerConfig        `yaml:"log_configuration,omitempty"`
	LogFormat           string               `yaml:"log_format,omitempty"`
	Processes           Processes            `yaml:"processes"`
	Environment         Environment          `yaml:"environment,omitempty"`
	ShellConfig         *command.ShellConfig `yaml:"shell,omitempty"`
	IsStrict            bool                 `yaml:"is_strict,omitempty"`
	Vars                Vars                 `yaml:"vars,omitempty"`
	DisableEnvExpansion bool                 `yaml:"disable_env_expansion,omitempty"`
	IsTuiDisabled       bool                 `yaml:"is_tui_disabled,omitempty"`
	ExtendsProject      string               `yaml:"extends,omitempty"`
	EnvCommands         EnvCmd               `yaml:"env_cmds,omitempty"`
	FileNames           []string             `yaml:"file_names,omitempty"`
	EnvFileNames        []string             `yaml:"env_file_names,omitempty"`
	DotEnvVars          map[string]string    `yaml:"dot_env_vars,omitempty"`
}

type ProcessFunc func(process ProcessConfig) error

// WithProcesses run ProcessFunc on each Process and dependencies in dependency order
func (p *Project) WithProcesses(names []string, fn ProcessFunc) error {
	return p.withProcesses(names, fn, map[string]bool{})
}

func (p *Project) GetDependenciesOrderNames() ([]string, error) {
	order := []string{}
	err := p.WithProcesses([]string{}, func(process ProcessConfig) error {
		if process.IsDeferred() {
			return nil
		}
		order = append(order, process.ReplicaName)
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

func (p *Project) GetElevatedShellArg() string {
	elevatedShellArg := p.ShellConfig.ElevatedShellArg
	if p.IsTuiDisabled {
		elevatedShellArg = ""
	}
	return elevatedShellArg
}

func (p *Project) GetProcesses(names ...string) ([]ProcessConfig, error) {
	processes := []ProcessConfig{}
	if len(names) == 0 {
		for _, proc := range p.Processes {
			processes = append(processes, proc)
		}
		return processes, nil
	}
	for _, name := range names {
		if proc, ok := p.Processes[name]; ok {
			processes = append(processes, proc)
		} else {
			found := false
			for _, process := range p.Processes {
				if process.Name == name {
					found = true
					processes = append(processes, process)
				}
			}
			if !found {
				return processes, fmt.Errorf("no such process: %s", name)
			}
		}
	}

	return processes, nil
}

func (p *Project) withProcesses(names []string, fn ProcessFunc, done map[string]bool) error {
	processes, err := p.GetProcesses(names...)
	if err != nil {
		return err
	}
	var finalErr error
	for _, process := range processes {
		if done[process.ReplicaName] {
			continue
		}
		done[process.ReplicaName] = true

		dependencies := process.GetDependencies()
		if len(dependencies) > 0 {
			err = p.withProcesses(dependencies, fn, done)
			if err != nil {
				finalErr = fmt.Errorf("error in process %s dependency: %w", process.Name, err)
				continue
			}
		}
		if err = fn(process); err != nil {
			return err
		}
	}
	return finalErr
}
