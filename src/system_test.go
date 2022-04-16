package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func getFixtures() []string {
	matches, err := filepath.Glob("../fixtures/process-compose-*.yaml")
	if err != nil {
		panic("no fixtures found")
	}
	return matches
}

func TestSystem_TestFixtures(t *testing.T) {
	fixtures := getFixtures()
	for _, fixture := range fixtures {

		if strings.Contains(fixture, "process-compose-with-log.yaml") {
			//there is a dedicated test for that TestSystem_TestComposeWithLog
			continue
		}

		t.Run(fixture, func(t *testing.T) {
			project := createProject(fixture)
			project.Run()
		})
	}
}

func TestSystem_TestComposeWithLog(t *testing.T) {
	fixture := filepath.Join("..", "fixtures", "process-compose-with-log.yaml")
	t.Run(fixture, func(t *testing.T) {
		project := createProject(fixture)
		project.Run()
		if _, err := os.Stat(project.LogLocation); err != nil {
			t.Errorf("log file %s not found", project.LogLocation)
		}
		if err := os.Remove(project.LogLocation); err != nil {
			t.Errorf("failed to delete the log file %s, %s", project.LogLocation, err.Error())
		}

		proc6log := project.Processes["process6"].LogLocation
		if _, err := os.Stat(proc6log); err != nil {
			t.Errorf("log file %s not found", proc6log)
		}
		if err := os.Remove(proc6log); err != nil {
			t.Errorf("failed to delete the log file %s, %s", proc6log, err.Error())
		}
	})
}

func TestSystem_TestComposeChain(t *testing.T) {
	fixture := filepath.Join("..", "fixtures", "process-compose-chain.yaml")
	t.Run(fixture, func(t *testing.T) {
		project := createProject(fixture)
		names, err := project.GetDependenciesOrderNames()
		if err != nil {
			t.Errorf("GetDependenciesOrderNames() error = %v", err)
			return
		}
		want := []string{
			"process8",
			"process7",
			"process6",
			"process5",
			"process4",
			"process3",
			"process2",
			"process1",
		}
		if !reflect.DeepEqual(names, want) {
			t.Errorf("Project.GetDependenciesOrderNames() = %v, want %v", names, want)
		}
	})
}

func Test_autoDiscoverComposeFile(t *testing.T) {
	type args struct {
		pwd string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Should not find",
			args: args{
				pwd: "../fixtures",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Should find process-compose.yaml",
			args: args{
				pwd: "../",
			},
			want:    filepath.Join("..", "process-compose.yaml"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := autoDiscoverComposeFile(tt.args.pwd)
			if (err != nil) != tt.wantErr {
				t.Errorf("autoDiscoverComposeFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("autoDiscoverComposeFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
