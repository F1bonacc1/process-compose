package loader

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/templater"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
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
			newProc := cloneProcess(&proc)
			newProc.ReplicaNum = replica
			repName := newProc.CalculateReplicaName()
			newProc.ReplicaName = repName
			if proc.Replicas == 1 {
				// Even if replicas == 1, we use newProc to ensure
				// it has its own memory separate from any other references.
				p.Processes[repName] = *newProc
			} else {
				procsToAdd = append(procsToAdd, *newProc)
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

func cloneProcess(proc *types.ProcessConfig) *types.ProcessConfig {
	// 1. Create a copy of the struct (Shallow copy)
	newProc := *proc

	// 2. DEEP COPY the Vars Map
	maps.Copy(newProc.Vars, proc.Vars)

	// 3. DEEP COPY the Environment Slices
	newProc.Environment = slices.Clone(proc.Environment)
	newProc.Args = slices.Clone(proc.Args)
	newProc.Entrypoint = slices.Clone(proc.Entrypoint)

	return &newProc
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

func convertStrDisabledToBool(p *types.Project) {
	for name, proc := range p.Processes {
		switch proc.IsDisabled {
		case "false":
			proc.Disabled = false
		case "true":
			proc.Disabled = true
		}
		p.Processes[name] = proc
	}
}

func disableProcsInEnv(p *types.Project) {
	procsToDisable := os.Getenv(config.EnvVarDisabledProcs)
	if procsToDisable == "" {
		return
	}
	disabledProcs := make(map[string]bool)
	for name := range strings.SplitSeq(procsToDisable, ",") {
		disabledProcs[name] = true
	}
	for name, proc := range p.Processes {
		if _, found := disabledProcs[name]; found {
			proc.Disabled = true
		}
		p.Processes[name] = proc
	}
}
