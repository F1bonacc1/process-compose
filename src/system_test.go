package main

import (
	"path/filepath"
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

		t.Run(fixture, func(t *testing.T) {
			project := createProject(fixture)
			project.Run()
		})
	}
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
			want:    "../process-compose.yaml",
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
