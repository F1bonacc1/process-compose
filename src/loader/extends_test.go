package loader

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/types"
)

func TestResolveProcessExtends_InheritsAndOverrides(t *testing.T) {
	p := &types.Project{
		Processes: types.Processes{
			"base": {
				Command:     "echo base",
				WorkingDir:  "/base/dir",
				Environment: types.Environment{"FOO=bar", "SHARED=base"},
				Entrypoint:  []string{"/bin/base"},
				RestartPolicy: types.RestartPolicyConfig{
					Restart: types.RestartPolicyAlways,
				},
			},
			"derived": {
				Extends:     "base",
				Command:     "echo derived",
				Environment: types.Environment{"FOO=override", "EXTRA=2"},
				Entrypoint:  []string{"/bin/derived"},
			},
		},
	}

	if err := resolveProcessExtends(p); err != nil {
		t.Fatalf("resolveProcessExtends() error = %v", err)
	}

	derived := p.Processes["derived"]

	if derived.Extends != "" {
		t.Errorf("Extends not cleared, got %q", derived.Extends)
	}
	// scalar override: child wins
	if derived.Command != "echo derived" {
		t.Errorf("Command = %q, want %q", derived.Command, "echo derived")
	}
	// scalar inheritance: unset child field inherits the parent value
	if derived.WorkingDir != "/base/dir" {
		t.Errorf("WorkingDir = %q, want %q", derived.WorkingDir, "/base/dir")
	}
	// nested struct inheritance
	if derived.RestartPolicy.Restart != types.RestartPolicyAlways {
		t.Errorf("RestartPolicy.Restart = %v, want always", derived.RestartPolicy.Restart)
	}
	// environment merged by key (child wins), then sorted
	wantEnv := types.Environment{"EXTRA=2", "FOO=override", "SHARED=base"}
	if !reflect.DeepEqual(derived.Environment, wantEnv) {
		t.Errorf("Environment = %v, want %v", derived.Environment, wantEnv)
	}
	// list fields appended: parent first, then child
	wantEntrypoint := []string{"/bin/base", "/bin/derived"}
	if !reflect.DeepEqual(derived.Entrypoint, wantEntrypoint) {
		t.Errorf("Entrypoint = %v, want %v", derived.Entrypoint, wantEntrypoint)
	}

	// the parent process must not be mutated by resolving the child
	base := p.Processes["base"]
	if base.Command != "echo base" ||
		!reflect.DeepEqual(base.Environment, types.Environment{"FOO=bar", "SHARED=base"}) ||
		!reflect.DeepEqual(base.Entrypoint, []string{"/bin/base"}) {
		t.Errorf("parent process was mutated: %+v", base)
	}
}

func TestResolveProcessExtends_EnvironmentValuesWithEquals(t *testing.T) {
	parentEnv := types.Environment{"BASE=a=b", "TOKEN=YWJjZA==", "SHARED=old=value", "EMPTY="}
	p := &types.Project{Processes: types.Processes{
		"base": {Environment: parentEnv},
		"derived": {
			Extends:     "base",
			Environment: types.Environment{"SHARED=new=value", "ADDED=x=y=z"},
		},
	}}

	if err := resolveProcessExtends(p); err != nil {
		t.Fatalf("resolveProcessExtends() error = %v", err)
	}
	want := types.Environment{
		"ADDED=x=y=z",
		"BASE=a=b",
		"EMPTY=",
		"SHARED=new=value",
		"TOKEN=YWJjZA==",
	}
	if got := p.Processes["derived"].Environment; !reflect.DeepEqual(got, want) {
		t.Errorf("Environment = %v, want %v", got, want)
	}
	if got := p.Processes["base"].Environment; !reflect.DeepEqual(got, parentEnv) {
		t.Errorf("parent Environment = %v, want %v", got, parentEnv)
	}
}

func TestResolveProcessExtends_TransitiveChain(t *testing.T) {
	p := &types.Project{
		Processes: types.Processes{
			"a": {Command: "echo a", Environment: types.Environment{"A=1"}},
			"b": {Extends: "a", Environment: types.Environment{"B=2"}},
			"c": {Extends: "b", Environment: types.Environment{"C=3"}},
		},
	}

	if err := resolveProcessExtends(p); err != nil {
		t.Fatalf("resolveProcessExtends() error = %v", err)
	}

	c := p.Processes["c"]
	if c.Command != "echo a" {
		t.Errorf("Command = %q, want %q (inherited through the chain)", c.Command, "echo a")
	}
	wantEnv := types.Environment{"A=1", "B=2", "C=3"}
	if !reflect.DeepEqual(c.Environment, wantEnv) {
		t.Errorf("Environment = %v, want %v", c.Environment, wantEnv)
	}
	if c.Extends != "" {
		t.Errorf("Extends not cleared, got %q", c.Extends)
	}
}

func TestResolveProcessExtends_NoExtendsIsNoop(t *testing.T) {
	p := &types.Project{
		Processes: types.Processes{
			"solo": {Command: "echo solo", Environment: types.Environment{"K=V"}},
		},
	}
	want := p.Processes["solo"]

	if err := resolveProcessExtends(p); err != nil {
		t.Fatalf("resolveProcessExtends() error = %v", err)
	}
	if !reflect.DeepEqual(p.Processes["solo"], want) {
		t.Errorf("process without extends changed: got %+v, want %+v", p.Processes["solo"], want)
	}
}

// TestResolveProcessExtends_DoesNotMutateParentPointers guards against the
// child's merge leaking into the base process's pointer fields (probes etc.).
// mergo merges pointer targets in place, so resolution must deep-copy the base.
func TestResolveProcessExtends_DoesNotMutateParentPointers(t *testing.T) {
	p := &types.Project{
		Processes: types.Processes{
			"base": {
				Command: "echo base",
				ReadinessProbe: &health.Probe{
					HttpGet: &health.HttpProbe{Host: "base-host", Port: "80", Path: "/", Scheme: "http"},
				},
			},
			"derived": {
				Extends: "base",
				ReadinessProbe: &health.Probe{
					HttpGet: &health.HttpProbe{Host: "derived-host", Port: "80", Path: "/", Scheme: "http"},
				},
			},
		},
	}

	if err := resolveProcessExtends(p); err != nil {
		t.Fatalf("resolveProcessExtends() error = %v", err)
	}

	if got := p.Processes["base"].ReadinessProbe.HttpGet.Host; got != "base-host" {
		t.Errorf("base ReadinessProbe host = %q, want %q (parent was mutated by child extends)", got, "base-host")
	}
	if got := p.Processes["derived"].ReadinessProbe.HttpGet.Host; got != "derived-host" {
		t.Errorf("derived ReadinessProbe host = %q, want %q", got, "derived-host")
	}
}

func TestResolveProcessExtends_Errors(t *testing.T) {
	tests := []struct {
		name      string
		processes types.Processes
		errSubstr string
	}{
		{
			name: "missing parent",
			processes: types.Processes{
				"a": {Extends: "ghost", Command: "echo a"},
			},
			errSubstr: "unknown process",
		},
		{
			name: "self extend",
			processes: types.Processes{
				"a": {Extends: "a", Command: "echo a"},
			},
			errSubstr: "cannot extend itself",
		},
		{
			name: "circular chain",
			processes: types.Processes{
				"a": {Extends: "b", Command: "echo a"},
				"b": {Extends: "a", Command: "echo b"},
			},
			errSubstr: "circular",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &types.Project{Processes: tt.processes}
			err := resolveProcessExtends(p)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
			}
			if !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.errSubstr)
			}
		})
	}
}

// TestLoadProcessExtends exercises process-level extends end-to-end through
// Load (including the chain derived -> base and a transitive chain_top ->
// derived), verifying wiring and ordering relative to the rest of the pipeline.
func TestLoadProcessExtends(t *testing.T) {
	fixture := filepath.Join("..", "..", "fixtures-code", "process-compose-process-extends.yaml")
	opts := &LoaderOptions{
		FileNames:        []string{fixture},
		IsInternalLoader: true,
	}
	project, err := Load(opts)
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	derived := project.Processes["derived"]
	if derived.Command != "echo base" {
		t.Errorf("derived.Command = %q, want %q", derived.Command, "echo base")
	}
	if derived.WorkingDir != "/tmp" {
		t.Errorf("derived.WorkingDir = %q, want %q", derived.WorkingDir, "/tmp")
	}
	if derived.RestartPolicy.Restart != types.RestartPolicyAlways {
		t.Errorf("derived.RestartPolicy.Restart = %v, want always", derived.RestartPolicy.Restart)
	}
	wantEnv := types.Environment{"EXTRA=2", "FOO=override", "SHARED=base"}
	if !reflect.DeepEqual(derived.Environment, wantEnv) {
		t.Errorf("derived.Environment = %v, want %v", derived.Environment, wantEnv)
	}
	if derived.Extends != "" {
		t.Errorf("derived.Extends not cleared, got %q", derived.Extends)
	}

	chainTop := project.Processes["chain_top"]
	if chainTop.Command != "echo top" {
		t.Errorf("chain_top.Command = %q, want %q", chainTop.Command, "echo top")
	}
	if chainTop.RestartPolicy.Restart != types.RestartPolicyAlways {
		t.Errorf("chain_top.RestartPolicy.Restart = %v, want always (inherited transitively)", chainTop.RestartPolicy.Restart)
	}
	if !reflect.DeepEqual(chainTop.Environment, wantEnv) {
		t.Errorf("chain_top.Environment = %v, want %v", chainTop.Environment, wantEnv)
	}
}
