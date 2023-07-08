package types

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"strings"
)

func (p *Project) Validate() {
	p.validateLogLevel()
	p.setConfigDefaults()
	p.deprecationCheck()
	p.validateProcessConfig()
}

func (p *Project) ValidateAfterMerge() error {
	p.assignDefaultProcessValues()
	p.cloneReplicas()
	return p.validateNoCircularDependencies()
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

func (p *Project) assignDefaultProcessValues() {
	for name, proc := range p.Processes {
		if proc.Namespace == "" {
			proc.Namespace = DefaultNamespace
		}
		if proc.Replicas == 0 {
			proc.Replicas = 1
		}
		proc.Name = name
		p.Processes[name] = proc
	}
}

func (p *Project) cloneReplicas() {
	procsToAdd := make([]ProcessConfig, 0)
	procsToDel := make([]string, 0)
	for name, proc := range p.Processes {
		if proc.Replicas > 1 {
			procsToDel = append(procsToDel, name)
		}
		for replica := 0; replica < proc.Replicas; replica++ {
			proc.ReplicaNum = replica
			repName := proc.CalculateReplicaName()
			proc.ReplicaName = repName
			if proc.Replicas == 1 {
				p.Processes[repName] = proc
			} else {
				procsToAdd = append(procsToAdd, proc)
			}
		}
	}
	for _, name := range procsToDel {
		delete(p.Processes, name)
	}
	for _, proc := range procsToAdd {
		p.Processes[proc.ReplicaName] = proc
	}
}

func (p *Project) validateNoCircularDependencies() error {
	visited := make(map[string]bool, len(p.Processes))
	stack := make(map[string]bool)
	for name := range p.Processes {
		if !visited[name] {
			if p.isCyclicHelper(name, visited, stack) {
				return fmt.Errorf("circular dependency found in %s", name)
			}
		}
	}
	return nil
}

func (p *Project) isCyclicHelper(procName string, visited map[string]bool, stack map[string]bool) bool {
	visited[procName] = true
	stack[procName] = true

	processes, err := p.getProcesses(procName)
	if err != nil {
		return false
	}
	for _, process := range processes {
		dependencies := process.GetDependencies()
		for _, neighbor := range dependencies {
			if !visited[neighbor] {
				if p.isCyclicHelper(neighbor, visited, stack) {
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
