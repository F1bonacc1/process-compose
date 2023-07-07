package app

import (
	"github.com/f1bonacc1/process-compose/src/types"
	"reflect"
	"testing"
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
