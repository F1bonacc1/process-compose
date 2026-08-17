package loader

import (
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/types"
	"reflect"
	"testing"
)

func getBaseProcess() *types.ProcessConfig {
	return &types.ProcessConfig{
		Name:        "proc1",
		Disabled:    false,
		IsDaemon:    false,
		Command:     "command",
		LogLocation: "",
		Environment: types.Environment{
			"k1=v1",
			"k2=v2",
		},
		RestartPolicy: types.RestartPolicyConfig{
			Restart:        types.RestartPolicyNo,
			BackoffSeconds: 1,
			MaxRestarts:    1,
		},
		DependsOn: types.DependsOnConfig{
			"proc1": {
				Condition: types.ProcessConditionCompleted,
			},
			"proc3": {
				Condition: types.ProcessConditionCompletedSuccessfully,
			},
		},
		LivenessProbe: nil,
		ReadinessProbe: &health.Probe{
			HttpGet: &health.HttpProbe{
				Host:   "127.0.0.1",
				Path:   "/is",
				Scheme: "http",
				Port:   "80",
			},
			InitialDelay:     5,
			PeriodSeconds:    4,
			TimeoutSeconds:   3,
			SuccessThreshold: 2,
			FailureThreshold: 1,
		},
		ShutDownParams: types.ShutDownParams{
			ShutDownCommand: "command",
			ShutDownTimeout: 3,
			Signal:          1,
		},
		DisableAnsiColors: false,
		WorkingDir:        "working/dir",
		Extensions:        nil,
	}
}

func getOverrideProcess() *types.ProcessConfig {
	return &types.ProcessConfig{
		Name:        "proc1",
		Disabled:    false,
		IsDaemon:    false,
		Command:     "override command",
		LogLocation: "",
		Environment: types.Environment{
			"k0=v0",
			"k1=override",
			"k3=v3",
			"k4=v4",
		},
		RestartPolicy: types.RestartPolicyConfig{
			Restart:        types.RestartPolicyAlways,
			BackoffSeconds: 2,
			MaxRestarts:    2,
		},
		DependsOn: types.DependsOnConfig{
			"proc1": {
				Condition: types.ProcessConditionCompletedSuccessfully,
			},
			"proc2": {
				Condition: types.ProcessConditionCompletedSuccessfully,
			},
		},
		LivenessProbe: &health.Probe{
			HttpGet: &health.HttpProbe{
				Host:   "google.com",
				Path:   "/isAlive",
				Scheme: "https",
				Port:   "443",
			},
			InitialDelay:     1,
			PeriodSeconds:    2,
			TimeoutSeconds:   3,
			SuccessThreshold: 4,
			FailureThreshold: 5,
		},
		ReadinessProbe: &health.Probe{
			HttpGet: &health.HttpProbe{
				Host:   "google.com",
				Path:   "/isAlive",
				Scheme: "https",
				Port:   "443",
			},
			InitialDelay:     1,
			PeriodSeconds:    2,
			TimeoutSeconds:   3,
			SuccessThreshold: 4,
			FailureThreshold: 5,
		},
		ShutDownParams: types.ShutDownParams{
			ShutDownCommand: "override command",
			ShutDownTimeout: 1,
			Signal:          2,
		},
		DisableAnsiColors: true,
		//WorkingDir:        "",
		Extensions: nil,
	}
}

func getMergedProcess() *types.ProcessConfig {
	return &types.ProcessConfig{
		Name:        "proc1",
		Disabled:    false,
		IsDaemon:    false,
		Command:     "override command",
		LogLocation: "",
		Environment: types.Environment{
			"k0=v0",
			"k1=override",
			"k2=v2",
			"k3=v3",
			"k4=v4",
		},
		RestartPolicy: types.RestartPolicyConfig{
			Restart:        types.RestartPolicyAlways,
			BackoffSeconds: 2,
			MaxRestarts:    2,
		},
		DependsOn: types.DependsOnConfig{
			"proc1": {
				Condition: types.ProcessConditionCompletedSuccessfully,
			},
			"proc2": {
				Condition: types.ProcessConditionCompletedSuccessfully,
			},
			"proc3": {
				Condition: types.ProcessConditionCompletedSuccessfully,
			},
		},
		LivenessProbe: &health.Probe{
			HttpGet: &health.HttpProbe{
				Host:   "google.com",
				Path:   "/isAlive",
				Scheme: "https",
				Port:   "443",
			},
			InitialDelay:     1,
			PeriodSeconds:    2,
			TimeoutSeconds:   3,
			SuccessThreshold: 4,
			FailureThreshold: 5,
		},
		ReadinessProbe: &health.Probe{
			HttpGet: &health.HttpProbe{
				Host:   "google.com",
				Path:   "/isAlive",
				Scheme: "https",
				Port:   "443",
			},
			InitialDelay:     1,
			PeriodSeconds:    2,
			TimeoutSeconds:   3,
			SuccessThreshold: 4,
			FailureThreshold: 5,
		},
		ShutDownParams: types.ShutDownParams{
			ShutDownCommand: "override command",
			ShutDownTimeout: 1,
			Signal:          2,
		},
		DisableAnsiColors: true,
		WorkingDir:        "working/dir",
		Extensions:        nil,
	}
}

func Test_mergeProcess(t *testing.T) {
	type args struct {
		baseProcess     *types.ProcessConfig
		overrideProcess *types.ProcessConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *types.ProcessConfig
		wantErr bool
	}{
		{
			name: "command",
			args: args{
				baseProcess:     getBaseProcess(),
				overrideProcess: getOverrideProcess(),
			},
			want:    getMergedProcess(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeProcess(tt.args.baseProcess, tt.args.overrideProcess)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeProcess() error = %+v, wantErr %+v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeProcess() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mergeProcess_NamespaceOverridesNotAppends(t *testing.T) {
	tests := []struct {
		name     string
		base     types.Namespaces
		override types.Namespaces
		want     types.Namespaces
	}{
		{name: "override replaces", base: types.Namespaces{"a"}, override: types.Namespaces{"b"}, want: types.Namespaces{"b"}},
		{name: "override replaces list", base: types.Namespaces{"a", "b"}, override: types.Namespaces{"c"}, want: types.Namespaces{"c"}},
		{name: "empty override keeps base", base: types.Namespaces{"a", "b"}, override: nil, want: types.Namespaces{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &types.ProcessConfig{Name: "p", Namespace: tt.base}
			override := &types.ProcessConfig{Name: "p", Namespace: tt.override}
			got, err := mergeProcess(base, override)
			if err != nil {
				t.Fatalf("mergeProcess() error = %v", err)
			}
			if !reflect.DeepEqual(got.Namespace, tt.want) {
				t.Errorf("mergeProcess() namespace = %#v, want %#v", got.Namespace, tt.want)
			}
		})
	}
}

// Test_mergeProcess_WatchPathsOverrideNotAppend pins the watch path transformer.
// mergeProcess merges with WithAppendSlice, so without it an override file
// would add its paths to the base ones - leaving no way to narrow a watch in an
// override, only to widen it.
func Test_mergeProcess_WatchPathsOverrideNotAppend(t *testing.T) {
	watch := func(paths ...types.WatchPath) *types.WatchConfig {
		if paths == nil {
			return nil
		}
		return &types.WatchConfig{Paths: paths}
	}
	src := types.WatchPath{Path: "./src"}
	pkg := types.WatchPath{Path: "./pkg"}
	web := types.WatchPath{Path: "./web"}

	tests := []struct {
		name     string
		base     *types.WatchConfig
		override *types.WatchConfig
		want     []types.WatchPath
	}{
		{
			name:     "override replaces",
			base:     watch(src),
			override: watch(web),
			want:     []types.WatchPath{web},
		},
		{
			name:     "override replaces a list",
			base:     watch(src, pkg),
			override: watch(web),
			want:     []types.WatchPath{web},
		},
		{
			name:     "empty override keeps base",
			base:     watch(src, pkg),
			override: nil,
			want:     []types.WatchPath{src, pkg},
		},
		{
			name:     "override adds a watch where there was none",
			base:     nil,
			override: watch(web),
			want:     []types.WatchPath{web},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &types.ProcessConfig{Name: "p", Watch: tt.base}
			override := &types.ProcessConfig{Name: "p", Watch: tt.override}
			got, err := mergeProcess(base, override)
			if err != nil {
				t.Fatalf("mergeProcess() error = %v", err)
			}
			if got.Watch == nil {
				t.Fatalf("mergeProcess() watch = nil, want %#v", tt.want)
			}
			if !reflect.DeepEqual(got.Watch.Paths, tt.want) {
				t.Errorf("mergeProcess() watch paths = %#v, want %#v", got.Watch.Paths, tt.want)
			}
		})
	}
}

// Test_mergeProcess_WatchScalarsOverride covers the non-slice watch fields,
// which take the ordinary mergo path rather than the transformer.
func Test_mergeProcess_WatchScalarsOverride(t *testing.T) {
	base := &types.ProcessConfig{Name: "p", Watch: &types.WatchConfig{
		Paths:    []types.WatchPath{{Path: "./src"}},
		Debounce: "300ms",
	}}
	override := &types.ProcessConfig{Name: "p", Watch: &types.WatchConfig{
		Debounce: "1s",
		Cascade:  true,
	}}

	got, err := mergeProcess(base, override)
	if err != nil {
		t.Fatalf("mergeProcess() error = %v", err)
	}
	if got.Watch.Debounce != "1s" {
		t.Errorf("debounce = %v, want 1s", got.Watch.Debounce)
	}
	if !got.Watch.Cascade {
		t.Error("cascade = false, want true")
	}
	if len(got.Watch.Paths) != 1 || got.Watch.Paths[0].Path != "./src" {
		t.Errorf("watch paths = %#v, want the base paths to survive", got.Watch.Paths)
	}
}

func Test_merge(t *testing.T) {
	type args struct {
		opts *LoaderOptions
	}
	tests := []struct {
		name    string
		args    args
		want    *types.Project
		wantErr bool
	}{
		{
			name: "No Processes",
			args: args{
				opts: &LoaderOptions{
					workingDir: "",
					FileNames:  nil,
					projects: []*types.Project{
						{
							Version:     "0.5",
							LogLocation: "loc1",
							LogLevel:    "Debug",
							LogLength:   100,
							Processes:   nil,
							Environment: types.Environment{
								"k1=v1",
								"k2=v2",
							},
							ShellConfig: nil,
						},
						{
							Version:     "0.6",
							LogLocation: "loc2",
							LogLevel:    "Info",
							LogLength:   200,
							Processes:   nil,
							Environment: types.Environment{
								"k0=v0",
								"k1=override",
								"k3=v3",
								"k4=v4",
							},
							ShellConfig: nil,
						},
					},
				},
			},
			want: &types.Project{
				Version:     "0.6",
				LogLocation: "loc2",
				LogLevel:    "Info",
				LogLength:   200,
				Processes:   nil,
				Environment: types.Environment{
					"k0=v0",
					"k1=override",
					"k2=v2",
					"k3=v3",
					"k4=v4",
				},
				ShellConfig: nil,
			},
			wantErr: false,
		},
		{
			name: "With Single Process",
			args: args{
				opts: &LoaderOptions{
					workingDir: "",
					FileNames:  nil,
					projects: []*types.Project{
						{
							Version:     "",
							LogLocation: "",
							LogLevel:    "",
							LogLength:   0,
							Processes: types.Processes{
								"proc1": *getBaseProcess(),
							},
							Environment: nil,
							ShellConfig: nil,
						},
						{
							Version:     "",
							LogLocation: "",
							LogLevel:    "",
							LogLength:   0,
							Processes: types.Processes{
								"proc1": *getOverrideProcess(),
							},
							Environment: nil,
							ShellConfig: nil,
						},
					},
				},
			},
			want: &types.Project{
				Version:     "",
				LogLocation: "",
				LogLevel:    "",
				LogLength:   0,
				Processes: types.Processes{
					"proc1": *getMergedProcess(),
				},
				Environment: nil,
				ShellConfig: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := merge(tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("merge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("merge() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_mergeProcesses(t *testing.T) {
	type args struct {
		base     types.Processes
		override types.Processes
	}
	tests := []struct {
		name    string
		args    args
		want    types.Processes
		wantErr bool
	}{
		{
			name: "Single Process",
			args: args{
				base: types.Processes{
					"proc1": *getBaseProcess(),
				},
				override: types.Processes{
					"proc1": *getOverrideProcess(),
				},
			},
			want: types.Processes{
				"proc1": *getMergedProcess(),
			},
			wantErr: false,
		},
		{
			name: "No Override",
			args: args{
				base: types.Processes{
					"proc1": *getBaseProcess(),
				},
				override: types.Processes{},
			},
			want: types.Processes{
				"proc1": *getBaseProcess(),
			},
			wantErr: false,
		},
		{
			name: "Multiple Processes",
			args: args{
				base: types.Processes{
					"proc1": *getBaseProcess(),
					"proc2": *getBaseProcess(),
				},
				override: types.Processes{
					"proc1": *getOverrideProcess(),
					"proc3": *getBaseProcess(),
				},
			},
			want: types.Processes{
				"proc1": *getMergedProcess(),
				"proc2": *getBaseProcess(),
				"proc3": *getBaseProcess(),
			},
			wantErr: false,
		},
		{
			name: "Disabled Process",
			args: args{
				base: types.Processes{
					"test": types.ProcessConfig{
						Disabled: true,
						Command:  "echo foo",
					},
				},
				override: types.Processes{
					"test": types.ProcessConfig{
						Disabled: false,
						Command:  "echo bar",
					},
				},
			},
			want: types.Processes{
				"test": types.ProcessConfig{
					Disabled: true,
					Command:  "echo bar",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeProcesses(tt.args.base, tt.args.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeProcesses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeProcesses() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}
