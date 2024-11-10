package loader

import (
	"path/filepath"
	"testing"
)

func Test_autoDiscoverComposeFile(t *testing.T) {
	type args struct {
		opts *LoaderOptions
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Should not find",
			args: args{
				opts: &LoaderOptions{
					workingDir: "../../fixtures",
					FileNames:  nil,
					projects:   nil,
				},
			},
			wantErr: true,
		},
		{
			name: "Should find process-compose.yaml",
			args: args{
				opts: &LoaderOptions{
					workingDir: "../../",
					FileNames:  nil,
					projects:   nil,
				},
			},
			wantErr: false,
		},
		{
			name: "Should find process-compose.yaml no CWD",
			args: args{
				opts: &LoaderOptions{
					workingDir: "",
					FileNames:  []string{"../../process-compose.yaml"},
					projects:   nil,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := autoDiscoverComposeFile(tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("autoDiscoverComposeFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				filesNum := len(tt.args.opts.FileNames)
				if filesNum == 0 {
					t.Errorf("autoDiscoverComposeFile() filesNum = %v, want Files %v", filesNum, 1)
				}
			}
		})
	}
}

func TestLoadExtendProject(t *testing.T) {
	fixture := filepath.Join("..", "..", "fixtures-code", "process-compose-with-log.yaml")
	opts := &LoaderOptions{
		FileNames:        []string{fixture},
		IsInternalLoader: true,
	}
	project, err := Load(opts)
	if err != nil {
		t.Error("failed to load project", err.Error())
		return
	}
	t.Run("no extend", func(t *testing.T) {
		err = loadExtendProject(project, opts, "", 0)
		if err != nil {
			t.Error("failed to load project", err.Error())
			return
		}
		if len(opts.projects) != 1 {
			t.Errorf("expected 1 project, got %d", len(opts.projects))
		}
	})
	t.Run("extend", func(t *testing.T) {
		project.ExtendsProject = "process-compose-chain.yaml"
		err = loadExtendProject(project, opts, fixture, 0)
		if err != nil {
			t.Error("failed to load project", err.Error())
			return
		}
		if len(opts.projects) != 2 {
			t.Errorf("expected 2 projects, got %d", len(opts.projects))
			return
		}
		if len(opts.FileNames) != 2 {
			t.Errorf("expected 2 files, got %d", len(opts.FileNames))
			return
		}
		//check files order
		if opts.FileNames[0] != project.ExtendsProject {
			t.Errorf("expected %s, got %s", project.ExtendsProject, opts.FileNames[0])
		}
		if opts.FileNames[1] != fixture {
			t.Errorf("expected %s, got %s", fixture, opts.FileNames[1])
		}
	})
	t.Run("prevent same project", func(t *testing.T) {
		project.ExtendsProject = filepath.Base(fixture)
		err = loadExtendProject(project, opts, fixture, 0)
		if err == nil {
			t.Error("expected error for same project, got nil")
			return
		}
	})
	t.Run("missing file", func(t *testing.T) {
		project.ExtendsProject = "missing.yaml"
		err = loadExtendProject(project, opts, "", 0)
		if err == nil {
			t.Error("expected error for missing extend project file, got nil")
			return
		}
	})
}

func TestLoadFileWithExtendProject(t *testing.T) {
	fixture := filepath.Join("..", "..", "fixtures-code", "process-compose-with-extends.yaml")
	opts := &LoaderOptions{
		FileNames:        []string{fixture},
		IsInternalLoader: true,
	}
	project, err := Load(opts)
	if err != nil {
		t.Error("failed to load project", err.Error())
		return
	}
	if len(opts.projects) != 2 {
		t.Errorf("expected 2 project, got %d", len(opts.projects))
	}
	if len(opts.FileNames) != 2 {
		t.Fatalf("expected 2 file, got %d", len(opts.FileNames))
	}

	//check files order
	expected := filepath.Join("..", "..", "fixtures-code", "process-compose-chain.yaml")
	if opts.FileNames[0] != expected {
		t.Errorf("expected %s, got %s", expected, opts.FileNames[1])
	}
	if opts.FileNames[1] != fixture {
		t.Errorf("expected %s, got %s", fixture, opts.FileNames[0])
	}
	if project.Processes["process1"].Command != "echo extending" {
		t.Errorf("expected %s, got %s", "echo extending", project.Processes["process1"].Command)
	}
}
