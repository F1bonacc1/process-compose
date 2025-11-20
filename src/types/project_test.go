package types

import (
	"slices"
	"testing"

	"github.com/f1bonacc1/process-compose/src/command"
)

func TestProject_WithProcesses(t *testing.T) {
	type fields struct {
		Version             string
		LogLocation         string
		LogLevel            string
		LogLength           int
		LoggerConfig        *LoggerConfig
		LogFormat           string
		Processes           Processes
		Environment         Environment
		ShellConfig         *command.ShellConfig
		IsStrict            bool
		Vars                Vars
		DisableEnvExpansion bool
		IsTuiDisabled       bool
		ExtendsProject      string
		FileNames           []string
		EnvFileNames        []string
		IsOrderedShutdown   bool
	}
	type args struct {
		names []string
		fn    ProcessFunc
	}
	order := []string{}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		wantOrder []string
	}{
		{
			name: "ShouldBeOk",
			fields: fields{
				Processes: map[string]ProcessConfig{
					"Process1": {
						Name:        "Process1",
						ReplicaName: "Process1",
					},
					"Process2": {
						Name:        "Process2",
						ReplicaName: "Process2",
					},
				},
			},
			args: args{
				names: []string{"Process1", "Process2"},
				fn: func(proc ProcessConfig) error {
					order = append(order, proc.Name)
					return nil
				},
			},
			wantErr: false,
			wantOrder: []string{
				"Process1",
				"Process2",
			},
		},
		{
			name: "ShouldBeError",
			fields: fields{
				Processes: map[string]ProcessConfig{
					"Process1": {
						Name:        "Process1",
						ReplicaName: "Process1",
					},
					"Process2": {
						Name:        "Process2",
						ReplicaName: "Process2",
					},
				},
			},
			args: args{
				names: []string{"Process1", "Process3"},
				fn: func(proc ProcessConfig) error {
					order = append(order, proc.Name)
					return nil
				},
			},
			wantErr: true,
		},
		{
			name: "ShouldBeOnlyOne",
			fields: fields{
				Processes: map[string]ProcessConfig{
					"Process1": {
						Name:        "Process1",
						ReplicaName: "Process1",
					},
					"Process2": {
						Name:        "Process2",
						ReplicaName: "Process2",
					},
				},
			},
			args: args{
				names: []string{"Process1"},
				fn: func(proc ProcessConfig) error {
					order = append(order, proc.Name)
					return nil
				},
			},
			wantErr: false,
			wantOrder: []string{
				"Process1",
			},
		},
		{
			name: "ShouldBeAllNoNames",
			fields: fields{
				Processes: map[string]ProcessConfig{
					"Process1": {
						Name:        "Process1",
						ReplicaName: "Process1",
					},
					"Process2": {
						Name:        "Process2",
						ReplicaName: "Process2",
					},
				},
			},
			args: args{
				names: []string{},
				fn: func(proc ProcessConfig) error {
					order = append(order, proc.Name)
					return nil
				},
			},
			wantErr: false,
			wantOrder: []string{
				"Process1",
				"Process2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order = []string{}
			p := &Project{
				Version:             tt.fields.Version,
				LogLocation:         tt.fields.LogLocation,
				LogLevel:            tt.fields.LogLevel,
				LogLength:           tt.fields.LogLength,
				LoggerConfig:        tt.fields.LoggerConfig,
				LogFormat:           tt.fields.LogFormat,
				Processes:           tt.fields.Processes,
				Environment:         tt.fields.Environment,
				ShellConfig:         tt.fields.ShellConfig,
				IsStrict:            tt.fields.IsStrict,
				Vars:                tt.fields.Vars,
				DisableEnvExpansion: tt.fields.DisableEnvExpansion,
				IsTuiDisabled:       tt.fields.IsTuiDisabled,
				ExtendsProject:      tt.fields.ExtendsProject,
				FileNames:           tt.fields.FileNames,
				EnvFileNames:        tt.fields.EnvFileNames,
				IsOrderedShutdown:   tt.fields.IsOrderedShutdown,
			}
			if err := p.WithProcesses(tt.args.names, tt.args.fn); (err != nil) != tt.wantErr {
				t.Errorf("WithProcesses() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if len(order) != len(tt.wantOrder) {
				t.Errorf("WithProcesses() order = %v, wantOrder %v", order, tt.wantOrder)
			}

			for _, name := range tt.wantOrder {
				if !slices.Contains(order, name) {
					t.Errorf("WithProcesses() order = %v, wantOrder %v", order, tt.wantOrder)
				}
			}
		})
	}
}

func TestProject_GetDependenciesOrderNames(t *testing.T) {
	p := &Project{
		Processes: map[string]ProcessConfig{
			"A": {Name: "A", ReplicaName: "A"},
			"B": {Name: "B", ReplicaName: "B", DependsOn: map[string]ProcessDependency{"A": {Condition: ProcessConditionStarted}}},
			"C": {Name: "C", ReplicaName: "C", DependsOn: map[string]ProcessDependency{"B": {Condition: ProcessConditionStarted}}},
		},
	}
	order, err := p.GetDependenciesOrderNames()
	if err != nil {
		t.Errorf("GetDependenciesOrderNames() error = %v", err)
	}
	expected := []string{"A", "B", "C"}
	if !slices.Equal(order, expected) {
		t.Errorf("GetDependenciesOrderNames() = %v, want %v", order, expected)
	}
}

func TestProject_GetLexicographicProcessNames(t *testing.T) {
	p := &Project{
		Processes: map[string]ProcessConfig{
			"B": {Name: "B"},
			"A": {Name: "A"},
			"C": {Name: "C"},
		},
	}
	names, err := p.GetLexicographicProcessNames()
	if err != nil {
		t.Errorf("GetLexicographicProcessNames() error = %v", err)
	}
	expected := []string{"A", "B", "C"}
	if !slices.Equal(names, expected) {
		t.Errorf("GetLexicographicProcessNames() = %v, want %v", names, expected)
	}
}

func TestProject_GetElevatedShellArg(t *testing.T) {
	tests := []struct {
		name          string
		shellConfig   *command.ShellConfig
		isTuiDisabled bool
		want          string
	}{
		{
			name: "TuiEnabled",
			shellConfig: &command.ShellConfig{
				ElevatedShellArg: "sudo",
			},
			isTuiDisabled: false,
			want:          "sudo",
		},
		{
			name: "TuiDisabled",
			shellConfig: &command.ShellConfig{
				ElevatedShellArg: "sudo",
			},
			isTuiDisabled: true,
			want:          "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Project{
				ShellConfig:   tt.shellConfig,
				IsTuiDisabled: tt.isTuiDisabled,
			}
			if got := p.GetElevatedShellArg(); got != tt.want {
				t.Errorf("GetElevatedShellArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProject_GetProcesses(t *testing.T) {
	p := &Project{
		Processes: map[string]ProcessConfig{
			"A": {Name: "A"},
			"B": {Name: "B"},
		},
	}

	// Test getting all processes
	procs, err := p.GetProcesses()
	if err != nil {
		t.Errorf("GetProcesses() error = %v", err)
	}
	if len(procs) != 2 {
		t.Errorf("GetProcesses() returned %d processes, want 2", len(procs))
	}

	// Test getting specific processes
	procs, err = p.GetProcesses("A")
	if err != nil {
		t.Errorf("GetProcesses('A') error = %v", err)
	}
	if len(procs) != 1 || procs[0].Name != "A" {
		t.Errorf("GetProcesses('A') returned %v, want [A]", procs)
	}

	// Test getting non-existent process
	_, err = p.GetProcesses("C")
	if err == nil {
		t.Errorf("GetProcesses('C') expected error, got nil")
	}
}
