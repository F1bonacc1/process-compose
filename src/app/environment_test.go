package app

import (
	"strings"
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
)

func TestBuildProcessEnvironment(t *testing.T) {
	proc := &types.ProcessConfig{
		Name:          "test-proc",
		ReplicaNum:    1,
		Environment:   []string{"PROC_ENV=proc_val", "COMMON_ENV=proc_override"},
		DisableDotEnv: false,
	}

	globalEnv := []string{"GLOBAL_ENV=global_val", "COMMON_ENV=global_val"}
	dotEnvVars := map[string]string{
		"DOTENV_VAR": "dotenv_val",
		"COMMON_ENV": "dotenv_val",
	}

	// Set a system env var to test inheritance
	t.Setenv("SYSTEM_ENV", "system_val")

	env := buildProcessEnvironment(proc, globalEnv, dotEnvVars)

	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"PC_PROC_NAME", "test-proc"},
		{EnvReplicaNum, "1"},
		{"DOTENV_VAR", "dotenv_val"},
		{"SYSTEM_ENV", "system_val"},
		{"GLOBAL_ENV", "global_val"},
		{"PROC_ENV", "proc_val"},
		{"COMMON_ENV", "proc_override"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := envMap[tt.key]; got != tt.expected {
				t.Errorf("expected %s=%q, got %q", tt.key, tt.expected, got)
			}
		})
	}
}

func TestBuildProcessEnvironment_DisableDotEnv(t *testing.T) {
	proc := &types.ProcessConfig{
		Name:          "test-proc",
		DisableDotEnv: true,
	}
	dotEnvVars := map[string]string{"DOTENV_VAR": "dotenv_val"}

	env := buildProcessEnvironment(proc, nil, dotEnvVars)

	for _, e := range env {
		if strings.HasPrefix(e, "DOTENV_VAR=") {
			t.Error("Expected DOTENV_VAR to be absent when DisableDotEnv is true")
		}
	}
}
