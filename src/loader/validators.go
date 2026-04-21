package loader

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	for _, proc := range p.Processes {
		err := proc.ValidateProcessConfig()
		if err != nil {
			log.Err(err).Msgf("Process config validation failed")
			if p.IsStrict {
				return err
			}
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
					return errors.New(errStr)
				}
				log.Error().Msg(errStr)
				continue
			}
			if dep.Condition == types.ProcessConditionHealthy && depProc.ReadinessProbe == nil && depProc.LivenessProbe == nil {
				errStr := fmt.Sprintf("health dependency defined in '%s' but no health check exists in '%s'", procName, depName)
				if p.IsStrict {
					return errors.New(errStr)
				}
				log.Error().Msg(errStr)
			}
			if dep.Condition == types.ProcessConditionLogReady && depProc.ReadyLogLine == "" {
				errStr := fmt.Sprintf("log ready dependency defined in '%s' but no ready log line exists in '%s'", procName, depName)
				log.Error().Msg(errStr)
				return errors.New(errStr)
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
			return errors.New(errStr)
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
				return errors.New(errStr)
			}
			if depProc.Disabled && !proc.Disabled {
				errStr := fmt.Sprintf("dependency process '%s' in process '%s' is disabled", depName, procName)
				if p.IsStrict {
					return errors.New(errStr)
				}
				log.Error().Msg(errStr)
			}
		}
	}
	return nil
}

func validateProject(p *types.Project) error {
	for key := range p.Extensions {
		if strings.HasPrefix(key, "x-") {
			continue
		}
		errStr := fmt.Sprintf("Unknown field '%s' in project file", key)
		if p.IsStrict {
			return errors.New(errStr)
		}
		log.Error().Msg(errStr)
	}
	return nil
}

func validateScheduledProcessScaling(p *types.Project) error {
	for name, proc := range p.Processes {
		if proc.Schedule != nil && proc.Replicas > 1 {
			errStr := fmt.Sprintf("scheduled process '%s' cannot be scaled (replicas > 1)", name)
			if p.IsStrict {
				return errors.New(errStr)
			}
			log.Error().Msg(errStr)
		}
	}
	return nil
}

func validateMCPConfig(p *types.Project) error {
	// Validate MCP server configuration
	if p.MCPServer != nil {
		if err := p.MCPServer.Validate(); err != nil {
			if p.IsStrict {
				return err
			}
			log.Error().Err(err).Msg("MCP server configuration invalid")
		}
	}

	// Validate Control MCP server configuration
	if p.MCPCtlServer != nil {
		if err := p.MCPCtlServer.Validate(); err != nil {
			if p.IsStrict {
				return err
			}
			log.Error().Err(err).Msg("Control MCP server configuration invalid")
		}
	}

	// Validate MCP process configurations
	for name, proc := range p.Processes {
		if proc.IsMCP() {
			log.Debug().
				Str("process", name).
				Str("command", proc.Command).
				Int("argCount", len(proc.MCP.Arguments)).
				Msg("Validating MCP process")

			if err := proc.MCP.Validate(name, proc.Command, proc.Args); err != nil {
				if p.IsStrict {
					return err
				}
				log.Error().Err(err).Msgf("MCP process '%s' configuration invalid", name)
			}

			// MCP processes should be disabled initially
			if !proc.Disabled {
				log.Warn().Msgf("MCP process '%s' should be disabled (setting disabled=true)", name)
				proc.Disabled = true
				p.Processes[name] = proc
			}
		}
	}

	return nil
}

func validateProcessEnvFileExists(p *types.Project) error {
	for procName, proc := range p.Processes {
		if proc.EnvFile != "" {
			envFile := proc.EnvFile
			if !filepath.IsAbs(envFile) && proc.WorkingDir != "" {
				envFile = filepath.Join(proc.WorkingDir, envFile)
			}
			if _, err := os.Stat(envFile); os.IsNotExist(err) {
				errStr := fmt.Sprintf("env_file '%s' for process '%s' does not exist", envFile, procName)
				if p.IsStrict {
					return errors.New(errStr)
				}
				log.Error().Msg(errStr)
			}
		}
	}
	return nil
}
