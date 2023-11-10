package loader

import (
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
)

type mutatorFunc func(p *types.Project)

func apply(p *types.Project, m ...mutatorFunc) {
	for _, mut := range m {
		mut(p)
	}
}

func setDefaultShell(p *types.Project) {
	if p.ShellConfig == nil {
		p.ShellConfig = command.DefaultShellConfig()
	}
	log.Info().Msgf("Global shell command: %s %s", p.ShellConfig.ShellCommand, p.ShellConfig.ShellArgument)
}

func assignDefaultProcessValues(p *types.Project) {
	for name, proc := range p.Processes {
		if proc.Namespace == "" {
			proc.Namespace = types.DefaultNamespace
		}
		if proc.Replicas == 0 {
			proc.Replicas = 1
		}
		proc.Name = name
		p.Processes[name] = proc
	}
}

// Exec Probes should use the same working dir if not specified otherwise
func copyWorkingDirToProbes(p *types.Project) {
	for name, proc := range p.Processes {
		if proc.LivenessProbe != nil &&
			proc.LivenessProbe.Exec != nil &&
			proc.LivenessProbe.Exec.WorkingDir == "" {
			proc.LivenessProbe.Exec.WorkingDir = proc.WorkingDir
		}
		if proc.ReadinessProbe != nil &&
			proc.ReadinessProbe.Exec != nil &&
			proc.ReadinessProbe.Exec.WorkingDir == "" {
			proc.ReadinessProbe.Exec.WorkingDir = proc.WorkingDir
		}
		p.Processes[name] = proc
	}
}

func cloneReplicas(p *types.Project) {
	procsToAdd := make([]types.ProcessConfig, 0)
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
