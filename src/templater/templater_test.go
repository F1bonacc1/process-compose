package templater

import (
	"errors"
	"fmt"
	"testing"

	"github.com/f1bonacc1/process-compose/src/health"
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

func TestTemplater_DisableCommandRendering(t *testing.T) {
	t.Run("Command rendering is skipped when DisableCommandRendering is true", func(t *testing.T) {
		vars := types.Vars{"Name": "Alice"}

		procConf := &types.ProcessConfig{
			ReplicaNum:              3,
			Command:                 `curl -d '{"key":"{{value}}"}'`,
			WorkingDir:              "{{.Name}}-dir",
			DisableCommandRendering: true,
		}
		templater := New(vars)
		templater.RenderProcess(procConf)

		if templater.GetError() != nil {
			t.Errorf("Expected no error but got %v", templater.GetError())
		}

		expectedCmd := `curl -d '{"key":"{{value}}"}'`
		if procConf.Command != expectedCmd {
			t.Errorf("Expected command %q but got %q", expectedCmd, procConf.Command)
		}

		expectedDir := "Alice-dir"
		if procConf.WorkingDir != expectedDir {
			t.Errorf("Expected working dir %q but got %q", expectedDir, procConf.WorkingDir)
		}
	})
}

// TestTemplater_RenderProcess_AllFields pins the full set of process fields
// that take Go template syntax. Every entry here is also listed under
// "Rendered Parameters" in www/docs/configuration.md - keep the two in sync.
func TestTemplater_RenderProcess_AllFields(t *testing.T) {
	procConf := &types.ProcessConfig{
		ReplicaNum:     2,
		Command:        "serve --port 404{{.PC_REPLICA_NUM}}",
		Entrypoint:     []string{"sh", "-c", "serve 404{{.PC_REPLICA_NUM}}"},
		WorkingDir:     "/srv/{{.APP}}",
		LogLocation:    "./{{.APP}}-{{.PC_REPLICA_NUM}}.log",
		Description:    "{{.APP}} replica {{.PC_REPLICA_NUM}}",
		ReadyLogLine:   "listening on 404{{.PC_REPLICA_NUM}}",
		EnvFile:        ".env.{{.APP}}",
		Environment:    []string{"PORT=404{{.PC_REPLICA_NUM}}"},
		ShutDownParams: types.ShutDownParams{ShutDownCommand: "kill-{{.APP}} 404{{.PC_REPLICA_NUM}}"},
		LoggerConfig: &types.LoggerConfig{
			Rotation: &types.LogRotationConfig{
				Directory: "/var/log/{{.APP}}",
				Filename:  "{{.APP}}-{{.PC_REPLICA_NUM}}.log",
			},
		},
		ReadinessProbe: &health.Probe{
			HttpGet: &health.HttpProbe{
				Host:    "{{.HOST}}",
				Path:    "/{{.APP}}",
				Scheme:  "http",
				Port:    "404{{.PC_REPLICA_NUM}}",
				Headers: map[string]string{"X-Replica": "{{.PC_REPLICA_NUM}}", "X-App": "{{.APP}}"},
			},
		},
		LivenessProbe: &health.Probe{
			Exec: &health.ExecProbe{
				Command:    "check {{.APP}} 404{{.PC_REPLICA_NUM}}",
				WorkingDir: "/srv/{{.APP}}",
			},
		},
	}

	templater := New(types.Vars{"APP": "web", "HOST": "127.0.0.1"})
	templater.RenderProcess(procConf)
	if err := templater.GetError(); err != nil {
		t.Fatalf("Expected no error but got %v (field %s)", err, templater.GetErrorField())
	}

	checks := []struct{ name, got, want string }{
		{"command", procConf.Command, "serve --port 4042"},
		{"entrypoint[2]", procConf.Entrypoint[2], "serve 4042"},
		{"working_dir", procConf.WorkingDir, "/srv/web"},
		{"log_location", procConf.LogLocation, "./web-2.log"},
		{"description", procConf.Description, "web replica 2"},
		{"ready_log_line", procConf.ReadyLogLine, "listening on 4042"},
		{"env_file", procConf.EnvFile, ".env.web"},
		{"environment[0]", procConf.Environment[0], "PORT=4042"},
		{"shutdown.command", procConf.ShutDownParams.ShutDownCommand, "kill-web 4042"},
		{"rotation.directory", procConf.LoggerConfig.Rotation.Directory, "/var/log/web"},
		{"rotation.filename", procConf.LoggerConfig.Rotation.Filename, "web-2.log"},
		{"http_get.host", procConf.ReadinessProbe.HttpGet.Host, "127.0.0.1"},
		{"http_get.path", procConf.ReadinessProbe.HttpGet.Path, "/web"},
		{"http_get.port", procConf.ReadinessProbe.HttpGet.Port, "4042"},
		{"http_get.headers.X-Replica", procConf.ReadinessProbe.HttpGet.Headers["X-Replica"], "2"},
		{"http_get.headers.X-App", procConf.ReadinessProbe.HttpGet.Headers["X-App"], "web"},
		{"exec.command", procConf.LivenessProbe.Exec.Command, "check web 4042"},
		{"exec.working_dir", procConf.LivenessProbe.Exec.WorkingDir, "/srv/web"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: expected %q but got %q", c.name, c.want, c.got)
		}
	}
}

// TestTemplater_DisableCommandRenderingCommandFamily pins the scope of
// is_template_disabled: it covers the shell-string command family (command,
// entrypoint, shutdown.command) and nothing else.
func TestTemplater_DisableCommandRenderingCommandFamily(t *testing.T) {
	rawCmd := `curl -d '{"key":"{{value}}"}'`
	rawShutdown := `docker inspect -f '{{.State.Running}}' web`
	rawEntry := `{{.State.Status}}`

	procConf := &types.ProcessConfig{
		ReplicaNum:              1,
		Command:                 rawCmd,
		Entrypoint:              []string{"sh", "-c", rawEntry},
		ShutDownParams:          types.ShutDownParams{ShutDownCommand: rawShutdown},
		WorkingDir:              "{{.Name}}-dir",
		ReadyLogLine:            "up {{.Name}}",
		DisableCommandRendering: true,
	}
	templater := New(types.Vars{"Name": "Alice"})
	templater.RenderProcess(procConf)

	if err := templater.GetError(); err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}
	if procConf.Command != rawCmd {
		t.Errorf("Expected command %q but got %q", rawCmd, procConf.Command)
	}
	if procConf.ShutDownParams.ShutDownCommand != rawShutdown {
		t.Errorf("Expected shutdown command %q but got %q", rawShutdown, procConf.ShutDownParams.ShutDownCommand)
	}
	if procConf.Entrypoint[2] != rawEntry {
		t.Errorf("Expected entrypoint %q but got %q", rawEntry, procConf.Entrypoint[2])
	}
	if procConf.WorkingDir != "Alice-dir" {
		t.Errorf("Expected working dir %q but got %q", "Alice-dir", procConf.WorkingDir)
	}
	if procConf.ReadyLogLine != "up Alice" {
		t.Errorf("Expected ready log line %q but got %q", "up Alice", procConf.ReadyLogLine)
	}
}

// TestTemplater_GetErrorField makes a broken template point at the field it
// came from - with a dozen rendered fields, the process name alone is not
// enough to find it.
func TestTemplater_GetErrorField(t *testing.T) {
	procConf := &types.ProcessConfig{
		Command:        "echo ok",
		ShutDownParams: types.ShutDownParams{ShutDownCommand: "{{ invalid .Key }}"},
		Description:    "{{.Key}}",
	}
	templater := New(types.Vars{"Key": "Value"})
	templater.RenderProcess(procConf)

	if templater.GetError() == nil {
		t.Fatal("Expected an error but got nil")
	}
	if got := templater.GetErrorField(); got != "shutdown.command" {
		t.Errorf("Expected error field %q but got %q", "shutdown.command", got)
	}
	// Fields after the failure keep their raw value rather than being blanked.
	if procConf.Description != "{{.Key}}" {
		t.Errorf("Expected description to be left untouched but got %q", procConf.Description)
	}
}
