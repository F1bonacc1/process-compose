package loader

import (
	"path/filepath"
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
)

// watchProject builds a single-process project whose watch path points at an
// existing directory, so that only the aspect under test can fail validation.
func watchProject(t *testing.T, strict bool, mutate func(*types.ProcessConfig)) *types.Project {
	t.Helper()
	proc := types.ProcessConfig{
		Name:     "api",
		Replicas: 1,
		Watch: &types.WatchConfig{
			Paths: []types.WatchPath{{Path: t.TempDir()}},
		},
	}
	if mutate != nil {
		mutate(&proc)
	}
	return &types.Project{
		Processes: types.Processes{"api": proc},
		IsStrict:  strict,
	}
}

func Test_validateWatchConfig(t *testing.T) {
	type args struct {
		p *types.Project
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Valid watch config",
			args:    args{p: watchProject(t, true, nil)},
			wantErr: false,
		},
		{
			name: "Valid watch config with patterns",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch.Debounce = "500ms"
				proc.Watch.Cascade = true
				proc.Watch.Paths[0].Include = []string{"**/*.go"}
				proc.Watch.Paths[0].Exclude = []string{"**/*_test.go", "bin/**"}
			})},
			wantErr: false,
		},
		{
			name: "No watch block at all",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch = nil
			})},
			wantErr: false,
		},

		// watch block with no paths
		{
			name: "Invalid empty paths (non strict)",
			args: args{p: watchProject(t, false, func(proc *types.ProcessConfig) {
				proc.Watch.Paths = nil
			})},
			wantErr: false,
		},
		{
			name: "Invalid empty paths (strict)",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch.Paths = nil
			})},
			wantErr: true,
		},

		// replicas
		{
			name: "Invalid scaled watched process (non strict)",
			args: args{p: watchProject(t, false, func(proc *types.ProcessConfig) {
				proc.Replicas = 2
			})},
			wantErr: false,
		},
		{
			name: "Invalid scaled watched process (strict)",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Replicas = 2
			})},
			wantErr: true,
		},

		// schedule
		{
			name: "Invalid scheduled watched process (non strict)",
			args: args{p: watchProject(t, false, func(proc *types.ProcessConfig) {
				proc.Schedule = &types.ScheduleConfig{Cron: "* * * * *"}
			})},
			wantErr: false,
		},
		{
			name: "Invalid scheduled watched process (strict)",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Schedule = &types.ScheduleConfig{Cron: "* * * * *"}
			})},
			wantErr: true,
		},
		{
			name: "Empty schedule block does not trip the schedule check",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Schedule = &types.ScheduleConfig{}
			})},
			wantErr: false,
		},

		// foreground
		{
			name: "Invalid foreground watched process (non strict)",
			args: args{p: watchProject(t, false, func(proc *types.ProcessConfig) {
				proc.IsForeground = true
			})},
			wantErr: false,
		},
		{
			name: "Invalid foreground watched process (strict)",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.IsForeground = true
			})},
			wantErr: true,
		},

		// debounce
		{
			name: "Invalid debounce (non strict)",
			args: args{p: watchProject(t, false, func(proc *types.ProcessConfig) {
				proc.Watch.Debounce = "300"
			})},
			wantErr: false,
		},
		{
			name: "Invalid debounce (strict)",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch.Debounce = "300"
			})},
			wantErr: true,
		},

		// buffer size
		{
			name: "Invalid buffer size below the fsnotify minimum (non strict)",
			args: args{p: watchProject(t, false, func(proc *types.ProcessConfig) {
				proc.Watch.BufferSize = 512
			})},
			wantErr: false,
		},
		{
			name: "Invalid buffer size below the fsnotify minimum (strict)",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch.BufferSize = 512
			})},
			wantErr: true,
		},
		{
			name: "Buffer size at the minimum is accepted",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch.BufferSize = types.MinWatchBufferSize
			})},
			wantErr: false,
		},
		{
			name: "Unset buffer size is accepted",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch.BufferSize = 0
			})},
			wantErr: false,
		},

		// empty path string
		{
			name: "Invalid empty path string (non strict)",
			args: args{p: watchProject(t, false, func(proc *types.ProcessConfig) {
				proc.Watch.Paths[0].Path = "   "
			})},
			wantErr: false,
		},
		{
			name: "Invalid empty path string (strict)",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch.Paths[0].Path = ""
			})},
			wantErr: true,
		},

		// template variables are not rendered for watch paths
		{
			name: "Invalid templated path (non strict)",
			args: args{p: watchProject(t, false, func(proc *types.ProcessConfig) {
				proc.Watch.Paths[0].Path = "{{.Vars.src}}"
			})},
			wantErr: false,
		},
		{
			name: "Invalid templated path (strict)",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch.Paths[0].Path = "{{.Vars.src}}/pkg"
			})},
			wantErr: true,
		},

		// glob patterns
		{
			name: "Invalid exclude pattern (non strict)",
			args: args{p: watchProject(t, false, func(proc *types.ProcessConfig) {
				proc.Watch.Paths[0].Exclude = []string{"[unclosed"}
			})},
			wantErr: false,
		},
		{
			name: "Invalid exclude pattern (strict)",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch.Paths[0].Exclude = []string{"[unclosed"}
			})},
			wantErr: true,
		},
		{
			name: "Invalid include pattern (strict)",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch.Paths[0].Include = []string{"[unclosed"}
			})},
			wantErr: true,
		},

		// missing path
		{
			name: "Invalid missing path (non strict)",
			args: args{p: watchProject(t, false, func(proc *types.ProcessConfig) {
				proc.Watch.Paths[0].Path = "/no/such/directory/anywhere"
			})},
			wantErr: false,
		},
		{
			name: "Invalid missing path (strict)",
			args: args{p: watchProject(t, true, func(proc *types.ProcessConfig) {
				proc.Watch.Paths[0].Path = "/no/such/directory/anywhere"
			})},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateWatchConfig(tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("validateWatchConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test_resolveWatchPaths pins path anchoring: a relative watch path resolves
// against the process working_dir, mirroring how exec probes are anchored.
func Test_resolveWatchPaths(t *testing.T) {
	workDir := t.TempDir()
	absPath := t.TempDir()

	project := &types.Project{
		Processes: types.Processes{
			"relative": {
				Name:       "relative",
				WorkingDir: workDir,
				Watch:      &types.WatchConfig{Paths: []types.WatchPath{{Path: "./src"}}},
			},
			"absolute": {
				Name:       "absolute",
				WorkingDir: workDir,
				Watch:      &types.WatchConfig{Paths: []types.WatchPath{{Path: absPath}}},
			},
			"unwatched": {
				Name:       "unwatched",
				WorkingDir: workDir,
			},
		},
	}

	resolveWatchPaths(project)

	if got, want := project.Processes["relative"].Watch.Paths[0].Path, filepath.Join(workDir, "src"); got != want {
		t.Errorf("relative path = %v, want %v", got, want)
	}
	if got := project.Processes["absolute"].Watch.Paths[0].Path; got != absPath {
		t.Errorf("absolute path = %v, want %v", got, absPath)
	}
	if project.Processes["unwatched"].Watch != nil {
		t.Error("resolveWatchPaths() created a watch config for an unwatched process")
	}
}

// Test_cloneProcess_DeepCopiesWatch guards replica independence: cloneReplicas
// copies ProcessConfig by value, so a shared *WatchConfig would let one
// replica's path resolution rewrite every other replica's paths.
func Test_cloneProcess_DeepCopiesWatch(t *testing.T) {
	original := &types.ProcessConfig{
		Name: "api",
		Watch: &types.WatchConfig{
			Paths: []types.WatchPath{{Path: "./src", Exclude: []string{"bin/**"}}},
		},
	}

	clone := cloneProcess(original)
	clone.Watch.Paths[0].Path = "/resolved/src"
	clone.Watch.Paths[0].Exclude[0] = "dist/**"

	if got := original.Watch.Paths[0].Path; got != "./src" {
		t.Errorf("cloneProcess() aliased watch path: original = %v, want ./src", got)
	}
	if got := original.Watch.Paths[0].Exclude[0]; got != "bin/**" {
		t.Errorf("cloneProcess() aliased watch exclude: original = %v, want bin/**", got)
	}

	if cloneProcess(&types.ProcessConfig{Name: "none"}).Watch != nil {
		t.Error("cloneProcess() invented a watch config where there was none")
	}
}
