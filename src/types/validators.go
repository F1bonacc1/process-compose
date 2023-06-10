package types

import (
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
	p.assignDefaultNamespace()
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

func (p *Project) assignDefaultNamespace() {
	for name, proc := range p.Processes {
		if proc.Namespace == "" {
			proc.Namespace = DefaultNamespace
			p.Processes[name] = proc
		}
	}
}
