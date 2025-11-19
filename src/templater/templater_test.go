package templater

import (
	"errors"
	"fmt"
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
)

func TestTemplater_Render(t *testing.T) {
	t.Run("Rendering valid template", func(t *testing.T) {
		vars := types.Vars{"Name": "Alice", "Age": 30}
		templater := New(vars)
		result := templater.Render("Name: {{.Name}}, Age: {{.Age}}")
		expected := "Name: Alice, Age: 30"
		if result != expected {
			t.Errorf("Expected %s but got %s", expected, result)
		}
	})

	t.Run("Rendering empty string", func(t *testing.T) {
		vars := types.Vars{"Key": "Value"}
		templater := New(vars)
		result := templater.Render("")
		if result != "" {
			t.Errorf("Expected an empty string but got %s", result)
		}
	})

	t.Run("Rendering with error", func(t *testing.T) {
		vars := types.Vars{"Key": "Value"}
		templater := New(vars)
		templater.err = errors.New("Error") // Simulating an error
		result := templater.Render("{{.Key}}")
		if result != "" {
			t.Errorf("Expected an empty string due to error but got %s", result)
		}
	})
}

func TestTemplater_RenderWithExtraVars(t *testing.T) {
	t.Run("Rendering with extra vars", func(t *testing.T) {
		vars := types.Vars{"Name": "Alice"}
		extraVars := types.Vars{"Age": 30}
		templater := New(vars)
		result := templater.RenderWithExtraVars("Name: {{.Name}}, Age: {{.Age}}", extraVars)
		expected := "Name: Alice, Age: 30"
		if result != expected {
			t.Errorf("Expected %s but got %s", expected, result)
		}
	})
	t.Run("Rendering with proc conf", func(t *testing.T) {
		vars := types.Vars{"Name": "Alice"}

		procConf := &types.ProcessConfig{
			ReplicaNum: 3,
			Command:    "Name: {{.Name}}, Replica: {{.PC_REPLICA_NUM}}",
		}
		templater := New(vars)
		templater.RenderProcess(procConf)
		expected := "Name: Alice, Replica: 3"
		if procConf.Command != expected {
			t.Errorf("Expected %s but got %s", expected, procConf.Command)
		}
	})
}

func TestTemplater_GetError(t *testing.T) {
	t.Run("Check error when rendering fails", func(t *testing.T) {
		vars := types.Vars{"Key": "Value"}
		templater := New(vars)
		templater.Render("{{ invalid .Key }}")
		err := templater.GetError()
		if err == nil {
			t.Error("Expected an error but got nil")
		}
	})
}

func TestTemplater_EnvVars(t *testing.T) {

	procs := make([]types.ProcessConfig, 3)

	for i := range procs {
		procs[i] = types.ProcessConfig{
			ReplicaNum:  i,
			Environment: []string{"REPL={{.PC_REPLICA_NUM}}"},
		}
	}

	templater := New(make(types.Vars))
	for i, proc := range procs {
		templater.RenderProcess(&proc)

		procs[i] = proc
	}

	for i, proc := range procs {
		expectedEnv := "REPL=" + fmt.Sprint(i)

		if proc.Environment[0] != expectedEnv {
			t.Errorf("Expected %s but got %s", expectedEnv, proc.Environment[0])
		}
	}

}
