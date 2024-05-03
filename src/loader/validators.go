package loader

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os/exec"
	"runtime"
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
			errStr := fmt.Sprintf("unknown key '%s' found in process '%s'", extKey, key)
			if p.IsStrict {
				return fmt.Errorf(errStr)
			}
			log.Error().Msgf(errStr)
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

func validatePlatformCompatibility(p *types.Project) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	for name, proc := range p.Processes {
		if proc.IsTty {
			return fmt.Errorf("PTY for process '%s' is not yet supported on Windows", name)
		}
	}
	return nil
}

func validateNoCircularDependencies(p *types.Project) error {
	visited := make(map[string]bool, len(p.Processes))
	stack := make(map[string]bool)
	for name := range p.Processes {
		if !visited[name] {
			if isCyclicHelper(p, name, visited, stack) {
				return fmt.Errorf("circular dependency found in '%s'", name)
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

func validateHealthDependencyHasHealthCheck(p *types.Project) error {
	for procName, proc := range p.Processes {
		for depName, dep := range proc.DependsOn {
			depProc, ok := p.Processes[depName]
			if !ok {
				errStr := fmt.Sprintf("dependency process '%s' in process '%s' is not defined", depName, procName)
				if p.IsStrict {
					return fmt.Errorf(errStr)
				}
				log.Error().Msg(errStr)
				continue
			}
			if dep.Condition == types.ProcessConditionHealthy && depProc.ReadinessProbe == nil && depProc.LivenessProbe == nil {
				errStr := fmt.Sprintf("health dependency defined in '%s' but no health check exists in '%s'", procName, depName)
				if p.IsStrict {
					return fmt.Errorf(errStr)
				}
				log.Error().Msg(errStr)
			}
			if dep.Condition == types.ProcessConditionLogReady && depProc.ReadyLogLine == "" {
				errStr := fmt.Sprintf("log ready dependency defined in '%s' but no ready log line exists in '%s'", procName, depName)
				log.Error().Msg(errStr)
				return fmt.Errorf(errStr)
			}
		}
	}
	return nil
}

func validateNoIncompatibleHealthChecks(p *types.Project) error {
	for procName, proc := range p.Processes {
		if proc.ReadinessProbe != nil && proc.ReadyLogLine != "" {
			errStr := fmt.Sprintf("'ready_log_line' and readiness probe defined in '%s' are incompatible", procName)
			log.Error().Msg(errStr)
			return fmt.Errorf(errStr)
		}
	}
	return nil
}

func validateDependencyIsEnabled(p *types.Project) error {
	for procName, proc := range p.Processes {
		for depName := range proc.DependsOn {
			depProc, ok := p.Processes[depName]
			if !ok {
				errStr := fmt.Sprintf("dependency process '%s' in process '%s' is not defined", depName, procName)
				if p.IsStrict {
					return fmt.Errorf(errStr)
				}
				log.Error().Msg(errStr)
				continue
			}
			if depProc.Disabled {
				errStr := fmt.Sprintf("dependency process '%s' in process '%s' is disabled", depName, procName)
				if p.IsStrict {
					return fmt.Errorf(errStr)
				}
				log.Error().Msg(errStr)
			}
		}
	}
	return nil
}
