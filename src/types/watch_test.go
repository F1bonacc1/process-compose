package types

import (
	"testing"
	"time"
)

func TestWatchConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config *WatchConfig
		want   bool
	}{
		{
			name:   "nil config",
			config: nil,
			want:   false,
		},
		{
			name:   "empty config",
			config: &WatchConfig{},
			want:   false,
		},
		{
			name:   "cascade but no paths",
			config: &WatchConfig{Cascade: true},
			want:   false,
		},
		{
			name:   "one path",
			config: &WatchConfig{Paths: []WatchPath{{Path: "./src"}}},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWatchConfig_GetDebounce(t *testing.T) {
	tests := []struct {
		name   string
		config *WatchConfig
		want   time.Duration
	}{
		{
			name:   "nil config falls back to default",
			config: nil,
			want:   DefaultWatchDebounce,
		},
		{
			name:   "empty value falls back to default",
			config: &WatchConfig{},
			want:   DefaultWatchDebounce,
		},
		{
			name:   "milliseconds",
			config: &WatchConfig{Debounce: "50ms"},
			want:   50 * time.Millisecond,
		},
		{
			name:   "seconds",
			config: &WatchConfig{Debounce: "2s"},
			want:   2 * time.Second,
		},
		{
			// GetDebounce must never return a zero or negative settle time; the
			// loader reports the malformed value separately.
			name:   "malformed value falls back to default",
			config: &WatchConfig{Debounce: "300"},
			want:   DefaultWatchDebounce,
		},
		{
			name:   "zero falls back to default",
			config: &WatchConfig{Debounce: "0s"},
			want:   DefaultWatchDebounce,
		},
		{
			name:   "negative falls back to default",
			config: &WatchConfig{Debounce: "-1s"},
			want:   DefaultWatchDebounce,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetDebounce(); got != tt.want {
				t.Errorf("GetDebounce() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWatchConfig_GetDebounceDuration_ReportsParseError(t *testing.T) {
	if _, err := (&WatchConfig{Debounce: "300"}).GetDebounceDuration(); err == nil {
		t.Error("GetDebounceDuration() error = nil, want a parse error for a unitless value")
	}
	if _, err := (&WatchConfig{Debounce: "300ms"}).GetDebounceDuration(); err != nil {
		t.Errorf("GetDebounceDuration() error = %v, want nil", err)
	}
}

func TestWatchConfig_GetMaxEntries(t *testing.T) {
	tests := []struct {
		name   string
		config *WatchConfig
		want   int
	}{
		{name: "nil config", config: nil, want: DefaultWatchMaxEntries},
		{name: "unset", config: &WatchConfig{}, want: DefaultWatchMaxEntries},
		{name: "negative", config: &WatchConfig{MaxEntries: -1}, want: DefaultWatchMaxEntries},
		{name: "explicit", config: &WatchConfig{MaxEntries: 16}, want: 16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetMaxEntries(); got != tt.want {
				t.Errorf("GetMaxEntries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWatchConfig_GetBufferSize(t *testing.T) {
	if got := (*WatchConfig)(nil).GetBufferSize(); got != DefaultWatchBufferSize {
		t.Errorf("GetBufferSize() = %v, want %v", got, DefaultWatchBufferSize)
	}
	if got := (&WatchConfig{BufferSize: 1024}).GetBufferSize(); got != 1024 {
		t.Errorf("GetBufferSize() = %v, want %v", got, 1024)
	}
}

// TestWatchConfig_Clone pins the deep copy that keeps replicas independent.
// cloneProcess copies ProcessConfig by value, so a shallow copy here would let
// one replica's normalization mutate every other replica.
func TestWatchConfig_Clone(t *testing.T) {
	if got := (*WatchConfig)(nil).Clone(); got != nil {
		t.Errorf("Clone() = %v, want nil for a nil config", got)
	}

	original := &WatchConfig{
		Cascade:  true,
		Debounce: "300ms",
		Paths: []WatchPath{
			{
				Path:    "./src",
				Include: []string{"**/*.go"},
				Exclude: []string{"**/*_test.go"},
			},
		},
	}
	clone := original.Clone()

	clone.Paths[0].Path = "./other"
	clone.Paths[0].Include[0] = "**/*.rs"
	clone.Paths[0].Exclude = append(clone.Paths[0].Exclude, "**/testdata/**")
	clone.Cascade = false

	if original.Paths[0].Path != "./src" {
		t.Errorf("Clone() aliased Path: original = %v, want ./src", original.Paths[0].Path)
	}
	if original.Paths[0].Include[0] != "**/*.go" {
		t.Errorf("Clone() aliased Include: original = %v, want **/*.go", original.Paths[0].Include[0])
	}
	if len(original.Paths[0].Exclude) != 1 {
		t.Errorf("Clone() aliased Exclude: original len = %v, want 1", len(original.Paths[0].Exclude))
	}
	if !original.Cascade {
		t.Error("Clone() aliased Cascade")
	}
}

// TestProcessConfig_Compare_DetectsWatchChange pins the Compare entry. Without
// it, UpdateProject treats a process whose only change is its watch block as
// "up to date" and silently drops the change on reload.
func TestProcessConfig_Compare_DetectsWatchChange(t *testing.T) {
	base := func() *ProcessConfig {
		return &ProcessConfig{
			Name:  "api",
			Watch: &WatchConfig{Paths: []WatchPath{{Path: "./src"}}},
		}
	}

	tests := []struct {
		name     string
		mutate   func(*ProcessConfig)
		wantSame bool
	}{
		{
			name:     "identical",
			mutate:   func(*ProcessConfig) {},
			wantSame: true,
		},
		{
			name:     "path changed",
			mutate:   func(p *ProcessConfig) { p.Watch.Paths[0].Path = "./cmd" },
			wantSame: false,
		},
		{
			name:     "cascade toggled",
			mutate:   func(p *ProcessConfig) { p.Watch.Cascade = true },
			wantSame: false,
		},
		{
			name:     "debounce changed",
			mutate:   func(p *ProcessConfig) { p.Watch.Debounce = "1s" },
			wantSame: false,
		},
		{
			name:     "exclude added",
			mutate:   func(p *ProcessConfig) { p.Watch.Paths[0].Exclude = []string{"**/*_test.go"} },
			wantSame: false,
		},
		{
			name:     "watch removed",
			mutate:   func(p *ProcessConfig) { p.Watch = nil },
			wantSame: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current, updated := base(), base()
			tt.mutate(updated)
			if got := current.Compare(updated); got != tt.wantSame {
				t.Errorf("Compare() = %v, want %v", got, tt.wantSame)
			}
		})
	}
}

// TestDisplayProcessStatus_Watching pins the derived state. An exited process
// with an armed watcher must not read "Completed" - that would suggest nothing
// more can happen and leave the project's continued running unexplained.
func TestDisplayProcessStatus_Watching(t *testing.T) {
	tests := []struct {
		name  string
		state ProcessState
		want  string
	}{
		{
			name:  "exited and watched",
			state: ProcessState{Status: ProcessStateCompleted, IsWatched: true},
			want:  ProcessStateWatching,
		},
		{
			name:  "exited, not watched",
			state: ProcessState{Status: ProcessStateCompleted},
			want:  ProcessStateCompleted,
		},
		{
			name:  "running and watched keeps its real status",
			state: ProcessState{Status: ProcessStateRunning, IsWatched: true, IsRunning: true},
			want:  ProcessStateRunning,
		},
		{
			// Watching must not mask a failure: a broken build is the more
			// important signal, so a watched process that failed reads Failed.
			name:  "failed and watched still reads Failed",
			state: ProcessState{Status: ProcessStateCompleted, ExitCode: 1, IsWatched: true},
			want:  "Failed",
		},
		{
			name:  "failed with an allowed exit code is idle, so it reads Watching",
			state: ProcessState{Status: ProcessStateCompleted, ExitCode: 130, SuccessExitCodes: []int{130}, IsWatched: true},
			want:  ProcessStateWatching,
		},
		{
			name:  "errored and watched keeps Error",
			state: ProcessState{Status: ProcessStateError, IsWatched: true},
			want:  ProcessStateError,
		},
		{
			name:  "skipped and watched keeps Skipped",
			state: ProcessState{Status: ProcessStateSkipped, IsWatched: true},
			want:  ProcessStateSkipped,
		},
		{
			// A watched process waiting on its dependencies has not exited, so
			// Watching would be a lie about what it is waiting for.
			name:  "pending and watched keeps Pending",
			state: ProcessState{Status: ProcessStatePending, IsWatched: true},
			want:  ProcessStatePending,
		},
		{
			// GetProcessState promotes the status before the TUI ever sees it;
			// running the display derivation over its own output must be a no-op.
			name:  "an already promoted status passes through",
			state: ProcessState{Status: ProcessStateWatching, IsWatched: true},
			want:  ProcessStateWatching,
		},
		{
			name: "scheduled takes precedence over watching",
			state: ProcessState{
				Status:      ProcessStateCompleted,
				IsWatched:   true,
				NextRunTime: func() *time.Time { now := time.Now(); return &now }(),
			},
			want: ProcessStateScheduled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DisplayProcessStatus(tt.state); got != tt.want {
				t.Errorf("DisplayProcessStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
