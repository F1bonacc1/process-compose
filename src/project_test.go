package main

import (
	"reflect"
	"testing"
)

func TestProject_GetDependenciesOrderNames(t *testing.T) {
	type fields struct {
		Version          string
		LogLevel         string
		LogLocation      string
		Processes        map[string]ProcessConfig
		Environment      []string
		runningProcesses map[string]*Process
	}
	tests := []struct {
		name    string
		fields  fields
		want    []string
		wantErr bool
	}{
		{
			name: "ShouldBe_4321",
			fields: fields{
				Processes: map[string]ProcessConfig{
					"Process1": {
						Name: "Process1",
						DependsOn: DependsOnConfig{
							"Process2": {},
						},
					},
					"Process2": {
						Name: "Process2",
						DependsOn: DependsOnConfig{
							"Process3": {},
						},
					},
					"Process3": {
						Name: "Process3",
						DependsOn: DependsOnConfig{
							"Process4": {},
						},
					},
					"Process4": {
						Name: "Process4",
					},
				},
			},
			want:    []string{"Process4", "Process3", "Process2", "Process1"},
			wantErr: false,
		},
		{
			name: "ShouldBe_Err",
			fields: fields{
				Processes: map[string]ProcessConfig{
					"Process1": {
						Name: "Process1",
						DependsOn: DependsOnConfig{
							"Process2": {},
						},
					},
					"Process2": {
						Name: "Process2",
						DependsOn: DependsOnConfig{
							"Process4": {},
						},
					},
				},
			},
			want:    []string{},
			wantErr: true,
		},
		{
			name: "ShouldBe_1",
			fields: fields{
				Processes: map[string]ProcessConfig{
					"Process1": {
						Name: "Process1",
						DependsOn: DependsOnConfig{
							"Process2": {},
						},
					},
					"Process2": {
						Name:     "Process2",
						Disabled: true,
					},
				},
			},
			want:    []string{"Process1"},
			wantErr: false,
		},
		{
			name: "ShouldBe_2",
			fields: fields{
				Processes: map[string]ProcessConfig{
					"Process1": {
						Name:     "Process1",
						Disabled: true,
						DependsOn: DependsOnConfig{
							"Process2": {},
						},
					},
					"Process2": {
						Name: "Process2",
					},
				},
			},
			want:    []string{"Process2"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Project{
				Version:          tt.fields.Version,
				LogLocation:      tt.fields.LogLocation,
				Processes:        tt.fields.Processes,
				Environment:      tt.fields.Environment,
				runningProcesses: tt.fields.runningProcesses,
			}
			got, err := p.GetDependenciesOrderNames()
			if (err != nil) != tt.wantErr {
				t.Errorf("Project.GetDependenciesOrderNames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Project.GetDependenciesOrderNames() = %v, want %v", got, tt.want)
			}
		})
	}
}
