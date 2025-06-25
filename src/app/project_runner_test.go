package app

import (
	"reflect"
	"strings"
	"testing"

	"github.com/f1bonacc1/process-compose/src/templater"
	"github.com/f1bonacc1/process-compose/src/types"
)

func TestProject_GetDependenciesOrderNames(t *testing.T) {
	type fields struct {
		Version     string
		LogLevel    string
		LogLocation string
		Processes   map[string]types.ProcessConfig
		Environment []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    [][]string
		wantErr bool
	}{
		{
			name: "ShouldBe_4321",
			fields: fields{
				Processes: map[string]types.ProcessConfig{
					"Process1": {
						Name:        "Process1",
						ReplicaName: "Process1",
						DependsOn: types.DependsOnConfig{
							"Process2": {},
						},
					},
					"Process2": {
						Name:        "Process2",
						ReplicaName: "Process2",
						DependsOn: types.DependsOnConfig{
							"Process3": {},
						},
					},
					"Process3": {
						Name:        "Process3",
						ReplicaName: "Process3",
						DependsOn: types.DependsOnConfig{
							"Process4": {},
						},
					},
					"Process4": {
						Name:        "Process4",
						ReplicaName: "Process4",
					},
				},
			},
			want: [][]string{
				{"Process4", "Process3", "Process2", "Process1"},
			},
			wantErr: false,
		},
		{
			name: "ShouldBe_Err",
			fields: fields{
				Processes: map[string]types.ProcessConfig{
					"Process1": {
						Name:        "Process1",
						ReplicaName: "Process1",
						DependsOn: types.DependsOnConfig{
							"Process2": {},
						},
					},
					"Process2": {
						Name:        "Process2",
						ReplicaName: "Process2",
						DependsOn: types.DependsOnConfig{
							"Process4": {},
						},
					},
				},
			},
			want:    [][]string{},
			wantErr: true,
		},
		{
			name: "ShouldBe_1",
			fields: fields{
				Processes: map[string]types.ProcessConfig{
					"Process1": {
						Name:        "Process1",
						ReplicaName: "Process1",
						DependsOn: types.DependsOnConfig{
							"Process2": {},
						},
					},
					"Process2": {
						Name:     "Process2",
						Disabled: true,
					},
				},
			},
			want:    [][]string{{"Process1"}},
			wantErr: false,
		},
		{
			name: "ShouldBe_2",
			fields: fields{
				Processes: map[string]types.ProcessConfig{
					"Process1": {
						Name:     "Process1",
						Disabled: true,
						DependsOn: types.DependsOnConfig{
							"Process2": {},
						},
					},
					"Process2": {
						Name:        "Process2",
						ReplicaName: "Process2",
					},
				},
			},
			want:    [][]string{{"Process2"}},
			wantErr: false,
		},
		{
			name: "WithReplicaDependees",
			fields: fields{
				Processes: map[string]types.ProcessConfig{
					"Process1": {
						Name:        "Process1",
						ReplicaName: "Process1",
						DependsOn: types.DependsOnConfig{
							"Process2": {},
						},
					},
					"Process2-0": {
						Name:        "Process2",
						ReplicaName: "Process2-0",
						Replicas:    2,
					},
					"Process2-1": {
						Name:        "Process2",
						ReplicaName: "Process2-1",
						Replicas:    2,
					},
				},
			},
			want: [][]string{

				{"Process2-0", "Process2-1", "Process1"},
				{"Process2-1", "Process2-0", "Process1"},
			},
			wantErr: false,
		},
		{
			name: "WithReplicas",
			fields: fields{
				Processes: map[string]types.ProcessConfig{
					"Process1": {
						Name:        "Process1",
						ReplicaName: "Process1",
					},
					"Process2-0": {
						Name:        "Process2",
						ReplicaName: "Process2-0",
						Replicas:    2,
						DependsOn: types.DependsOnConfig{
							"Process1": {},
						},
					},
					"Process2-1": {
						Name:        "Process2",
						ReplicaName: "Process2-1",
						Replicas:    2,
						DependsOn: types.DependsOnConfig{
							"Process1": {},
						},
					},
				},
			},
			want: [][]string{
				{"Process1", "Process2-1", "Process2-0"},
				{"Process1", "Process2-0", "Process2-1"},
			},
			wantErr: false,
		},
		{
			name: "WithReplicasBoth",
			fields: fields{
				Processes: map[string]types.ProcessConfig{
					"Process1-0": {
						Name:        "Process1",
						ReplicaName: "Process1-0",
						Replicas:    2,
					},
					"Process1-1": {
						Name:        "Process1",
						ReplicaName: "Process1-1",
						Replicas:    2,
					},
					"Process2-0": {
						Name:        "Process2",
						ReplicaName: "Process2-0",
						Replicas:    2,
						DependsOn: types.DependsOnConfig{
							"Process1": {},
						},
					},
					"Process2-1": {
						Name:        "Process2",
						ReplicaName: "Process2-1",
						Replicas:    2,
						DependsOn: types.DependsOnConfig{
							"Process1": {},
						},
					},
				},
			},
			want: [][]string{
				{"Process1-0", "Process1-1", "Process2-0", "Process2-1"},
				{"Process1-0", "Process1-1", "Process2-1", "Process2-0"},
				{"Process1-1", "Process1-0", "Process2-0", "Process2-1"},
				{"Process1-1", "Process1-0", "Process2-1", "Process2-0"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &types.Project{
				Version:     tt.fields.Version,
				LogLocation: tt.fields.LogLocation,
				Processes:   tt.fields.Processes,
				Environment: tt.fields.Environment,
			}
			got, err := p.GetDependenciesOrderNames()
			if (err != nil) != tt.wantErr {
				t.Errorf("Project.GetDependenciesOrderNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//if !reflect.DeepEqual(got, tt.want) && (tt.wantOr != nil && !reflect.DeepEqual(got, tt.wantOr)) {
			//	t.Errorf("Project.GetDependenciesOrderNames() = %v, want %v", got, tt.want)
			//}
			found := false
			for _, want := range tt.want {
				if reflect.DeepEqual(got, want) {
					found = true
					break
				}
			}
			if !found && !tt.wantErr {
				t.Errorf("Project.GetDependenciesOrderNames() = %v, want one of %v", got, tt.want)
			}
		})
	}
}

func TestProjectRunner_GetProjectName(t *testing.T) {
	type fields struct{ Name string }
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name:   "ShouldContain_project name",
			fields: fields{Name: "project name"},
			want:   "project name",
		},
		{
			name: "ShouldContain_app",
			want: "app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProjectRunner{
				project: &types.Project{
					Name: tt.fields.Name,
				},
			}

			got, err := p.GetProjectName()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProjectRunner.GetProjectName() error = %v, wantErr %v", err, nil)
				return
			}

			if !strings.Contains(got, tt.want) {
				t.Errorf("ProjectRunner.GetProjectName() = %s, want %s", got, tt.want)
			}
		})
func TestProjectRunner_EnvironmentExpansion(t *testing.T) {
	testProcess := types.ProcessConfig{
		Vars: map[string]interface{}{
			"PROCESS_VAR": "process_value",
		},
		Name:    "test-process",
		Command: "echo hello",
		Environment: []string{
			"LOCAL_VAR={{.GLOBAL_VAR}}",
			"PROCESS_VAR={{.PROCESS_VAR}}",
			"ANOTHER_VAR=fixed_value",
		},
	}
	p := &types.Project{
		Vars: map[string]interface{}{
			"GLOBAL_VAR": "global_value",
		},
		Processes: map[string]types.ProcessConfig{
			"test-process": testProcess,
		},
	}

	tpl := templater.New(p.Vars)
	for name, proc := range p.Processes {
		tpl.RenderProcess(&proc)
		p.Processes[name] = proc
	}

	expectedEnv := map[string]string{
		"LOCAL_VAR":   "global_value",
		"PROCESS_VAR": "process_value",
		"ANOTHER_VAR": "fixed_value",
	}

	actualEnv := make(map[string]string)
	for _, envVar := range testProcess.Environment {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			actualEnv[parts[0]] = parts[1]
		}
	}

	// Assert environment variables are correctly expanded
	for key, expectedValue := range expectedEnv {
		if actualValue, ok := actualEnv[key]; !ok {
			t.Errorf("Expected environment variable %s not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("Environment variable %s: expected %s, got %s", key, expectedValue, actualValue)
		}
	}
}
