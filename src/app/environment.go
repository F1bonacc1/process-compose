package app

import (
	"os"
	"strconv"

	"github.com/f1bonacc1/process-compose/src/types"
)

func buildProcessEnvironment(
	proc *types.ProcessConfig,
	globalEnv []string,
	dotEnvVars map[string]string,
) []string {
	env := []string{
		"PC_PROC_NAME=" + proc.Name,
		EnvReplicaNum + "=" + strconv.Itoa(proc.ReplicaNum),
	}

	// .env variables and system environment MUST come BEFORE YAML configurations
	// so that explicit YAML configs can override both .env and system defaults.
	// Precedence order (lowest to highest):
	// 1. .env file variables (baseline defaults)
	// 2. System environment (os.Environ - can override .env from shell)
	// 3. Global YAML environment section (explicit config overrides)
	// 4. Local process YAML environment section (highest - process-specific overrides)
	if dotEnvVars != nil && !proc.DisableDotEnv {
		for k, v := range dotEnvVars {
			env = append(env, k+"="+v)
		}
	}

	env = append(env, os.Environ()...)
	env = append(env, globalEnv...)
	env = append(env, proc.Environment...)
	return env
}
