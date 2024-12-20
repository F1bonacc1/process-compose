package loader

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/templater"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"path/filepath"
)

type mutatorFunc func(p *types.Project)
type mutatorFuncE func(p *types.Project) error

func applyWithErr(p *types.Project, m ...mutatorFuncE) error {
	for _, mut := range m {
		if err := mut(p); err != nil {
			return err
		}
	}
	return nil
}

func apply(p *types.Project, m ...mutatorFunc) {
	for _, mut := range m {
		mut(p)
	}
}

func setDefaultShell(p *types.Project) {
	if p.ShellConfig == nil {
		p.ShellConfig = command.DefaultShellConfig()
	} else if p.ShellConfig.ElevatedShellCmd == "" || p.ShellConfig.ElevatedShellArg == "" {
		shell := command.DefaultShellConfig()
		p.ShellConfig.ElevatedShellCmd = shell.ElevatedShellCmd
		p.ShellConfig.ElevatedShellArg = shell.ElevatedShellArg
	}
	log.Info().Msgf("Global shell command: %s %s", p.ShellConfig.ShellCommand, p.ShellConfig.ShellArgument)
}

func assignDefaultProcessValues(p *types.Project) {
	if p.Processes == nil {
		p.Processes = make(map[string]types.ProcessConfig)
	}
	for name, proc := range p.Processes {
		if proc.Namespace == "" {
			proc.Namespace = types.DefaultNamespace
		}
		if proc.Replicas == 0 {
			proc.Replicas = 1
		}
		if proc.LaunchTimeout < 1 {
			proc.LaunchTimeout = types.DefaultLaunchTimeout
		}
		proc.Name = name
		p.Processes[name] = proc
	}
}

// this function is used only in extended projects to assign the location of the compose.yaml as the working dir
func copyWorkingDirToProcesses(p *types.Project, wd string) {
	for name, proc := range p.Processes {
		if proc.WorkingDir == "" {
			proc.WorkingDir = wd
			p.Processes[name] = proc
		} else if !filepath.IsAbs(proc.WorkingDir) {
			proc.WorkingDir = filepath.Join(wd, proc.WorkingDir)
			p.Processes[name] = proc
		}
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

func assignExecutableAndArgs(p *types.Project) {
	elevatedShellArg := p.GetElevatedShellArg()
	for name, proc := range p.Processes {
		proc.AssignProcessExecutableAndArgs(p.ShellConfig, elevatedShellArg)

		p.Processes[name] = proc
	}
}

func renderTemplates(p *types.Project) error {
	tpl := templater.New(p.Vars)
	for name, proc := range p.Processes {
		tpl.RenderProcess(&proc)

		if tpl.GetError() != nil {
			return fmt.Errorf("error rendering template for process %s: %w", name, tpl.GetError())
		}
		p.Processes[name] = proc
	}
	return nil
}
