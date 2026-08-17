package loader

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
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

// validateWatchConfig rejects watch configurations that are malformed, or that
// combine with features whose interaction is unsafe or undefined. Each rejection
// follows the house convention: fail the load under strict mode, log otherwise.
func validateWatchConfig(p *types.Project) error {
	for name, proc := range p.Processes {
		if proc.Watch == nil {
			continue
		}
		reject := func(format string, args ...any) error {
			errStr := fmt.Sprintf(format, args...)
			if p.IsStrict {
				return errors.New(errStr)
			}
			log.Error().Msg(errStr)
			return nil
		}

		if len(proc.Watch.Paths) == 0 {
			if err := reject("process '%s' has a 'watch' block with no 'paths'", name); err != nil {
				return err
			}
			continue
		}

		// A replicated process would install one watcher per replica on the
		// same tree - N times the descriptors, events and simultaneous
		// restarts - and whether replicas should restart together or in a
		// rolling fashion is undefined. Same shape as scheduled + scaling.
		if proc.Replicas > 1 {
			if err := reject("watched process '%s' cannot be scaled (replicas > 1)", name); err != nil {
				return err
			}
		}

		// A restart neither pauses nor resumes the cron job, and the
		// scheduler's max_concurrent guard counts only scheduler-initiated
		// starts, so a watch-triggered start would silently violate it.
		if proc.Schedule.IsScheduled() {
			if err := reject("process '%s' cannot combine 'schedule' with 'watch'", name); err != nil {
				return err
			}
		}

		// Foreground processes run inside the TUI with the terminal suspended
		// and never enter the runner's process registry, so a watch-triggered
		// restart would silently start one as an ordinary background process.
		if proc.IsForeground {
			if err := reject("process '%s' cannot combine 'is_foreground' with 'watch'", name); err != nil {
				return err
			}
		}

		// fsnotify refuses a buffer this small on Windows, which would leave the
		// process unwatched with no obvious cause. Reject it everywhere.
		if proc.Watch.BufferSize != 0 && proc.Watch.BufferSize < types.MinWatchBufferSize {
			if err := reject("process '%s' has a watch 'buffer_size' of %d, below the minimum of %d bytes",
				name, proc.Watch.BufferSize, types.MinWatchBufferSize); err != nil {
				return err
			}
		}

		if _, err := proc.Watch.GetDebounceDuration(); err != nil {
			if err := reject("process '%s' has an invalid watch 'debounce' value '%s': %v (expected a duration such as '300ms' or '1s')",
				name, proc.Watch.Debounce, err); err != nil {
				return err
			}
		}

		for _, watchPath := range proc.Watch.Paths {
			if strings.TrimSpace(watchPath.Path) == "" {
				if err := reject("process '%s' has a watch path with an empty 'path'", name); err != nil {
					return err
				}
				continue
			}
			// Watch paths are not run through the templater, so a variable
			// reference would silently watch a literal directory of that name.
			if strings.Contains(watchPath.Path, "{{") {
				if err := reject("process '%s' watch path '%s' uses a template variable, which is not supported for watch paths",
					name, watchPath.Path); err != nil {
					return err
				}
				continue
			}
			if err := validateWatchPatterns(p, name, watchPath); err != nil {
				return err
			}
			if info, err := os.Stat(watchPath.Path); os.IsNotExist(err) {
				if err := reject("watch path '%s' for process '%s' does not exist", watchPath.Path, name); err != nil {
					return err
				}
			} else if err == nil && !info.IsDir() {
				// Watching a file directly is lost the moment an editor saves
				// atomically (write to a temp file, then rename over it).
				log.Warn().Msgf("watch path '%s' for process '%s' is a file; its parent directory will be watched and filtered to this name",
					watchPath.Path, name)
			}
		}
	}
	return nil
}

func validateWatchPatterns(p *types.Project, name string, watchPath types.WatchPath) error {
	for _, group := range []struct {
		kind     string
		patterns []string
	}{
		{"include", watchPath.Include},
		{"exclude", watchPath.Exclude},
	} {
		for _, pattern := range group.patterns {
			if doublestar.ValidatePattern(pattern) {
				continue
			}
			errStr := fmt.Sprintf("process '%s' watch path '%s' has an invalid %s pattern '%s'",
				name, watchPath.Path, group.kind, pattern)
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
