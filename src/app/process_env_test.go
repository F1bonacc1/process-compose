package app

import (
	"strings"
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
)

// Test_getProcessEnvironment_Precedence tests the environment variable precedence order.
// The correct order (lowest to highest) should be:
// 1. System environment (os.Environ)
// 2. .env file variables
// 3. Global YAML environment section
// 4. Local process YAML environment section (highest)
func Test_getProcessEnvironment_Precedence(t *testing.T) {
	tests := []struct {
		name           string
		globalEnv      []string
		procEnv        []string
		dotEnvVars     map[string]string
		disableDotEnv  bool
		expectedKey    string
		expectedValue  string
		shouldNotExist bool
	}{
		{
			name:          "TestEnvPrecedence_DotenvOnly - .env variable should apply when no conflicts",
			globalEnv:     []string{},
			procEnv:       []string{},
			dotEnvVars:    map[string]string{"TEST_VAR": "from_dotenv"},
			disableDotEnv: false,
			expectedKey:   "TEST_VAR",
			expectedValue: "from_dotenv",
		},
		{
			name:          "TestEnvPrecedence_SystemVsDotenv - system env should override .env when no YAML config",
			globalEnv:     []string{},
			procEnv:       []string{},
			dotEnvVars:    map[string]string{"TEST_SYSTEM_VAR": "from_dotenv"},
			disableDotEnv: false,
			expectedKey:   "TEST_SYSTEM_VAR",
			expectedValue: "from_system_env",
		},
		{
			name:          "TestEnvPrecedence_DotenvVsGlobal - global YAML should override .env",
			globalEnv:     []string{"TEST_VAR=from_global_yaml"},
			procEnv:       []string{},
			dotEnvVars:    map[string]string{"TEST_VAR": "from_dotenv"},
			disableDotEnv: false,
			expectedKey:   "TEST_VAR",
			expectedValue: "from_global_yaml",
		},
		{
			name:          "TestEnvPrecedence_DotenvVsLocal - local YAML should override .env",
			globalEnv:     []string{},
			procEnv:       []string{"TEST_VAR=from_local_yaml"},
			dotEnvVars:    map[string]string{"TEST_VAR": "from_dotenv"},
			disableDotEnv: false,
			expectedKey:   "TEST_VAR",
			expectedValue: "from_local_yaml",
		},
		{
			name:          "TestEnvPrecedence_DotenvVsGlobalAndLocal - local YAML should override both .env and global",
			globalEnv:     []string{"TEST_VAR=from_global_yaml"},
			procEnv:       []string{"TEST_VAR=from_local_yaml"},
			dotEnvVars:    map[string]string{"TEST_VAR": "from_dotenv"},
			disableDotEnv: false,
			expectedKey:   "TEST_VAR",
			expectedValue: "from_local_yaml",
		},
		{
			name:          "TestEnvPrecedence_GlobalVsLocal - local YAML should override global YAML",
			globalEnv:     []string{"TEST_VAR=from_global_yaml"},
			procEnv:       []string{"TEST_VAR=from_local_yaml"},
			dotEnvVars:    map[string]string{},
			disableDotEnv: false,
			expectedKey:   "TEST_VAR",
			expectedValue: "from_local_yaml",
		},
		{
			name:           "TestEnvPrecedence_DotenvDisabled - .env should not be present when disabled",
			globalEnv:      []string{},
			procEnv:        []string{},
			dotEnvVars:     map[string]string{"TEST_VAR": "from_dotenv"},
			disableDotEnv:  true,
			expectedKey:    "TEST_VAR",
			expectedValue:  "",
			shouldNotExist: true,
		},
		{
			name:          "TestEnvPrecedence_SystemEnv - global YAML should override system env",
			globalEnv:     []string{"TEST_SYSTEM_VAR=from_global_yaml"},
			procEnv:       []string{},
			dotEnvVars:    map[string]string{"TEST_SYSTEM_VAR": "from_dotenv"},
			disableDotEnv: false,
			expectedKey:   "TEST_SYSTEM_VAR",
			expectedValue: "from_global_yaml",
		},
	}

	// Set a system environment variable for the last test
	t.Setenv("TEST_SYSTEM_VAR", "from_system_env")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Process{
				globalEnv:  tt.globalEnv,
				dotEnvVars: tt.dotEnvVars,
				procConf: &types.ProcessConfig{
					Name:          "test-process",
					ReplicaNum:    0,
					Environment:   tt.procEnv,
					DisableDotEnv: tt.disableDotEnv,
				},
			}

			env := p.getProcessEnvironment()

			// Find the last occurrence of the key (last = highest precedence)
			lastValue := ""
			found := false
			for _, e := range env {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 && parts[0] == tt.expectedKey {
					lastValue = parts[1]
					found = true
				}
			}

			if tt.shouldNotExist {
				// Variable should not be present when is_dotenv_disabled is true
				if found && strings.Contains(lastValue, "from_dotenv") {
					t.Errorf("Expected %s to not contain 'from_dotenv', but found: %s", tt.expectedKey, lastValue)
				}
			} else {
				// Variable should have expected value
				if !found {
					t.Errorf("Expected to find %s in environment, but it was not present", tt.expectedKey)
				} else if lastValue != tt.expectedValue {
					t.Errorf("Expected %s=%s, got %s=%s", tt.expectedKey, tt.expectedValue, tt.expectedKey, lastValue)
				}
			}
		})
	}
}

// Test_getProcessEnvironment_CompleteChain tests the complete precedence chain
// with all four sources (system, .env, global, local)
func Test_getProcessEnvironment_CompleteChain(t *testing.T) {
	// Set up system environment
	t.Setenv("CHAIN_VAR", "from_system_env")

	p := &Process{
		globalEnv: []string{"CHAIN_VAR=from_global_yaml"},
		dotEnvVars: map[string]string{
			"CHAIN_VAR": "from_dotenv",
		},
		procConf: &types.ProcessConfig{
			Name:        "test-process",
			ReplicaNum:  0,
			Environment: []string{"CHAIN_VAR=from_local_yaml"},
		},
	}

	env := p.getProcessEnvironment()

	// Find the last occurrence of CHAIN_VAR
	lastValue := ""
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 && parts[0] == "CHAIN_VAR" {
			lastValue = parts[1]
		}
	}

	expectedValue := "from_local_yaml"
	if lastValue != expectedValue {
		t.Errorf("Expected CHAIN_VAR=%s (local should win entire chain), got CHAIN_VAR=%s", expectedValue, lastValue)
	}
}

// Test_getProcessEnvironment_MetadataVariables tests that PC_PROC_NAME and PC_REPLICA_NUM
// are always present in the environment
func Test_getProcessEnvironment_MetadataVariables(t *testing.T) {
	p := &Process{
		globalEnv:  []string{},
		dotEnvVars: map[string]string{},
		procConf: &types.ProcessConfig{
			Name:        "my-test-process",
			ReplicaNum:  5,
			Environment: []string{},
		},
	}

	env := p.getProcessEnvironment()

	foundProcName := false
	foundReplicaNum := false

	for _, e := range env {
		if strings.HasPrefix(e, "PC_PROC_NAME=my-test-process") {
			foundProcName = true
		}
		if strings.HasPrefix(e, "PC_REPLICA_NUM=5") {
			foundReplicaNum = true
		}
	}

	if !foundProcName {
		t.Error("Expected PC_PROC_NAME to be present in environment")
	}
	if !foundReplicaNum {
		t.Error("Expected PC_REPLICA_NUM to be present in environment")
	}
}

// Test_getProcessEnvironment_EmptyValue tests that empty string values
// from YAML should override .env values
func Test_getProcessEnvironment_EmptyValue(t *testing.T) {
	p := &Process{
		globalEnv: []string{"EMPTY_VAR="},
		dotEnvVars: map[string]string{
			"EMPTY_VAR": "from_dotenv",
		},
		procConf: &types.ProcessConfig{
			Name:        "test-process",
			ReplicaNum:  0,
			Environment: []string{},
		},
	}

	env := p.getProcessEnvironment()

	// Find the last occurrence of EMPTY_VAR
	lastValue := ""
	found := false
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 && parts[0] == "EMPTY_VAR" {
			lastValue = parts[1]
			found = true
		}
	}

	if !found {
		t.Error("Expected EMPTY_VAR to be present in environment")
	} else if lastValue != "" {
		t.Errorf("Expected EMPTY_VAR to be empty string (override from global YAML), got: '%s'", lastValue)
	}
}

// Test_getProcessEnvironment_MultipleVariables tests precedence with multiple variables
func Test_getProcessEnvironment_MultipleVariables(t *testing.T) {
	p := &Process{
		globalEnv: []string{
			"VAR1=global1",
			"VAR2=global2",
		},
		dotEnvVars: map[string]string{
			"VAR1": "dotenv1",
			"VAR2": "dotenv2",
			"VAR3": "dotenv3",
		},
		procConf: &types.ProcessConfig{
			Name:       "test-process",
			ReplicaNum: 0,
			Environment: []string{
				"VAR2=local2",
			},
		},
	}

	env := p.getProcessEnvironment()

	// Extract all variable values
	vars := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}

	// VAR1: global should override dotenv
	if vars["VAR1"] != "global1" {
		t.Errorf("Expected VAR1=global1, got VAR1=%s", vars["VAR1"])
	}

	// VAR2: local should override both global and dotenv
	if vars["VAR2"] != "local2" {
		t.Errorf("Expected VAR2=local2, got VAR2=%s", vars["VAR2"])
	}

	// VAR3: dotenv should apply (no conflicts)
	if vars["VAR3"] != "dotenv3" {
		t.Errorf("Expected VAR3=dotenv3, got VAR3=%s", vars["VAR3"])
	}
}
