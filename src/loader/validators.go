package loader

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os/exec"
	"strings"
)

type validatorFunc func(p *types.Project) error

func validate(p *types.Project, v ...validatorFunc) error {
	for _, f := range v {
		if err := f(p); err != nil {
			return err
		}
	}
	return nil
}

func validateLogLevel(p *types.Project) error {
	if p.LogLevel != "" {
		lvl, err := zerolog.ParseLevel(p.LogLevel)
		if err != nil {
			if p.IsStrict {
				return fmt.Errorf("unknown log level %s", p.LogLevel)
			}
			log.Warn().Msgf("Unknown log level %s defaulting to %s",
				p.LogLevel, zerolog.GlobalLevel().String())
		} else {
			zerolog.SetGlobalLevel(lvl)
		}
	}
	return nil
}

func validateProcessConfig(p *types.Project) error {
	for key, proc := range p.Processes {
		if len(proc.Extensions) == 0 {
			continue
		}
		for extKey := range proc.Extensions {
			if strings.HasPrefix(extKey, "x-") {
				continue
			}
			if p.IsStrict {
				return fmt.Errorf("unknown key %s found in process %s", extKey, key)
			}
			log.Error().Msgf("Unknown key %s found in process %s", extKey, key)
		}
	}
	return nil
}

func validateShellConfig(p *types.Project) error {
	_, err := exec.LookPath(p.ShellConfig.ShellCommand)
	if err != nil {
		log.Err(err).Msgf("Shell command '%s' not found", p.ShellConfig.ShellCommand)
	}
	return err
}

func validateNoCircularDependencies(p *types.Project) error {
	visited := make(map[string]bool, len(p.Processes))
	stack := make(map[string]bool)
	for name := range p.Processes {
		if !visited[name] {
			if isCyclicHelper(p, name, visited, stack) {
				return fmt.Errorf("circular dependency found in %s", name)
			}
		}
	}
	return nil
}

func isCyclicHelper(p *types.Project, procName string, visited map[string]bool, stack map[string]bool) bool {
	visited[procName] = true
	stack[procName] = true

	processes, err := p.GetProcesses(procName)
	if err != nil {
		return false
	}
	for _, process := range processes {
		dependencies := process.GetDependencies()
		for _, neighbor := range dependencies {
			if !visited[neighbor] {
				if isCyclicHelper(p, neighbor, visited, stack) {
					return true
				}
			} else if stack[neighbor] {
				return true
			}
		}
	}

	stack[procName] = false
	return false
}
