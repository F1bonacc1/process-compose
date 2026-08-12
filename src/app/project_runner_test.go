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
	}
}

func TestProjectRunner_EnvironmentExpansion(t *testing.T) {
	testProcess := types.ProcessConfig{
		Vars: map[string]any{
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
		Vars: map[string]any{
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

func TestProjectRunner_getNamespaceProcesses(t *testing.T) {
	tests := []struct {
		name      string
		processes map[string]types.ProcessConfig
		namespace string
		want      []string
		depsOrder []string
		// wantOrdered compares want to the result as an ordered slice instead
		// of a set. Only meaningful when the expected order is deterministic.
		wantOrdered bool
		wantErr     string
	}{
		{
			name: "DefaultNamespace",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"default"}},
				"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"other"}},
				"p3": {Name: "p3", ReplicaName: "p3"}, // default implied
			},
			namespace: "default",
			want:      []string{"p1", "p3"},
			depsOrder: []string{"p1", "p2", "p3"}, // order matters
		},
		{
			name: "OtherNamespace",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"default"}},
				"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"other"}},
			},
			namespace: "other",
			want:      []string{"p2"},
			depsOrder: []string{"p1", "p2"},
		},
		{
			name: "EmptyNamespaceSameAsDefault",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"default"}},
			},
			namespace: "",
			want:      []string{"p1"},
			depsOrder: []string{"p1"},
		},
		{
			name: "RespectDependencyOrder",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1"}},
				"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"ns1"}, DependsOn: types.DependsOnConfig{"p1": {}}},
			},
			namespace: "ns1",
			want:      []string{"p1", "p2"},
			depsOrder: []string{"p1", "p2"},
		},
		{
			name: "MultiNamespaceFirst",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1", "ns2"}},
				"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"ns2"}},
			},
			namespace: "ns1",
			want:      []string{"p1"},
			depsOrder: []string{"p1", "p2"},
		},
		{
			name: "MultiNamespaceSecond",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1", "ns2"}},
				"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"ns2"}},
			},
			namespace: "ns2",
			want:      []string{"p1", "p2"},
			depsOrder: []string{"p1", "p2"},
		},
		{
			// Every member excluded from an `up <process>...` selection.
			// Addresses issue #528.
			name: "AllMembersDisabled",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1"}},
				"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"ns2"}, Disabled: true},
				"p3": {Name: "p3", ReplicaName: "p3", Namespace: types.Namespaces{"ns2"}, Disabled: true,
					DependsOn: types.DependsOnConfig{"p2": {}}},
			},
			namespace:   "ns2",
			want:        []string{"p2", "p3"},
			wantOrdered: true,
		},
		{
			name: "MixedEnabledAndDisabledMembers",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1"}, Disabled: true},
				"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"ns1"},
					DependsOn: types.DependsOnConfig{"p1": {}}},
			},
			namespace:   "ns1",
			want:        []string{"p1", "p2"},
			wantOrdered: true,
		},
		{
			name: "ForegroundMemberExcluded",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1"}},
				"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"ns1"}, IsForeground: true},
			},
			namespace: "ns1",
			want:      []string{"p1"},
		},
		{
			name: "CrossNamespaceDependencyNotIncluded",
			processes: map[string]types.ProcessConfig{
				"db": {Name: "db", ReplicaName: "db", Namespace: types.Namespaces{"infra"}},
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1"},
					DependsOn: types.DependsOnConfig{"db": {}}},
			},
			namespace: "ns1",
			want:      []string{"p1"},
		},
		{
			// p2 -> db -> p1: the members are ordered only through a
			// dependency that belongs to another namespace.
			name: "OrderThroughCrossNamespaceDependency",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1"}},
				"db": {Name: "db", ReplicaName: "db", Namespace: types.Namespaces{"infra"},
					DependsOn: types.DependsOnConfig{"p1": {}}},
				"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"ns1"},
					DependsOn: types.DependsOnConfig{"db": {}}},
			},
			namespace:   "ns1",
			want:        []string{"p1", "p2"},
			wantOrdered: true,
		},
		{
			name: "UnknownNamespace",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1"}},
			},
			namespace: "nope",
			wantErr:   "namespace nope not found (no processes assigned)",
		},
		{
			// The namespace is listed by `namespace list` and by the TUI modal,
			// so the error has to say why it can't be operated on.
			name: "AllMembersForeground",
			processes: map[string]types.ProcessConfig{
				"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1"}, IsForeground: true},
				"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"ns2"}},
			},
			namespace: "ns1",
			wantErr:   "namespace ns1 has only foreground processes, which are excluded from namespace operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &types.Project{
				Processes: tt.processes,
			}

			runner := &ProjectRunner{
				project: p,
			}

			got, err := runner.getNamespaceProcesses(tt.namespace)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("getNamespaceProcesses() = %v, want error %q", got, tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Errorf("getNamespaceProcesses() error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("getNamespaceProcesses error: %v", err)
			}

			if tt.wantOrdered {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("getNamespaceProcesses() = %v, want %v (in order)", got, tt.want)
				}
				return
			}

			gotMap := make(map[string]bool)
			wantMap := make(map[string]bool)
			for _, g := range got {
				gotMap[g] = true
			}
			for _, w := range tt.want {
				wantMap[w] = true
			}
			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("getNamespaceProcesses() = %v, want %v", got, tt.want)
			}
		})
	}
}
