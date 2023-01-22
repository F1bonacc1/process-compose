package types

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"sort"
	"strings"
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

func (p *Project) Validate() {
	p.validateLogLevel()
	p.setConfigDefaults()
	p.deprecationCheck()
	p.validateProcessConfig()
}

func (p *Project) validateLogLevel() {
	if p.LogLevel != "" {
		lvl, err := zerolog.ParseLevel(p.LogLevel)
		if err != nil {
			log.Warn().Msgf("Unknown log level %s defaulting to %s",
				p.LogLevel, zerolog.GlobalLevel().String())
		} else {
			zerolog.SetGlobalLevel(lvl)
		}

	}
}

func (p *Project) setConfigDefaults() {
	if p.ShellConfig == nil {
		p.ShellConfig = command.DefaultShellConfig()
	}
	log.Info().Msgf("Global shell command: %s %s", p.ShellConfig.ShellCommand, p.ShellConfig.ShellArgument)
	command.ValidateShellConfig(*p.ShellConfig)
}

func (p *Project) deprecationCheck() {
	for key, proc := range p.Processes {
		if proc.RestartPolicy.Restart == RestartPolicyOnFailureDeprecated {
			deprecationHandler("2022-10-30", key, RestartPolicyOnFailureDeprecated, RestartPolicyOnFailure, "restart policy")
		}
	}
}

func (p *Project) validateProcessConfig() {
	for key, proc := range p.Processes {
		if len(proc.Extensions) == 0 {
			continue
		}
		for extKey := range proc.Extensions {
			if strings.HasPrefix(extKey, "x-") {
				continue
			}
			log.Error().Msgf("Unknown key %s found in process %s", extKey, key)
		}
	}
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

func (p *Project) GetLexicographicProcessNames() []string {

	names := []string{}
	for name := range p.Processes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
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
