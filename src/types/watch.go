package types

import (
	"slices"
	"time"
)

const (
	// DefaultWatchDebounce is how long a burst of filesystem events is allowed
	// to settle before a restart is triggered.
	DefaultWatchDebounce = 300 * time.Millisecond

	// DefaultWatchMaxEntries caps how many directories a single process may
	// watch. The cap exists because kqueue (macOS, BSD) opens a file descriptor
	// per watched entry, and process-compose already spends descriptors on
	// process pipes, PTYs and log files - an unbounded watch can exhaust the
	// limit and break process launching itself.
	DefaultWatchMaxEntries = 8192

	// DefaultWatchBufferSize is the ReadDirectoryChangesW buffer size used on
	// Windows, where a fixed buffer silently drops events under churn. Ignored
	// on every other platform.
	DefaultWatchBufferSize = 64 * 1024

	// MinWatchBufferSize is the smallest buffer fsnotify accepts. Below it the
	// Windows backend refuses to register the watch at all, so the value is
	// rejected at load time on every platform - a config that would fail on
	// Windows should not quietly pass on Linux.
	MinWatchBufferSize = 4096
)

// WatchConfig defines the file watching configuration of a process. A change to
// a watched path restarts the process, and - with Cascade - its dependents.
type WatchConfig struct {
	// Paths to watch. A config with no paths watches nothing.
	Paths []WatchPath `yaml:"paths,omitempty" json:"paths,omitempty"`

	// Cascade also restarts the transitive dependents of this process, in
	// dependency order. Restarts never propagate to dependencies.
	Cascade bool `yaml:"cascade,omitempty" json:"cascade,omitempty"`

	// Debounce is how long to wait for changes to settle before restarting,
	// e.g. "300ms" or "1s". Defaults to DefaultWatchDebounce.
	//
	// This is a string rather than a time.Duration so that it survives the JSON
	// round trip used by the REST API and by OriginalConfig (which ScaleProcess
	// replays to build replicas), and so that the generated JSON schema types it
	// as a string. ScheduleConfig.Interval is a string for the same reasons.
	Debounce string `yaml:"debounce,omitempty" json:"debounce,omitempty"`

	// MaxEntries overrides DefaultWatchMaxEntries for this process.
	MaxEntries int `yaml:"max_entries,omitempty" json:"max_entries,omitempty"`

	// BufferSize overrides DefaultWatchBufferSize. Windows only.
	BufferSize int `yaml:"buffer_size,omitempty" json:"buffer_size,omitempty"`

	// DisableDefaultExcludes turns off the built-in ignore list (.git,
	// node_modules, build output directories, log files and editor swap files).
	DisableDefaultExcludes bool `yaml:"disable_default_excludes,omitempty" json:"disable_default_excludes,omitempty"`
}

// WatchPath is a single watched root and the filters applied beneath it.
type WatchPath struct {
	// Path is the directory to watch, resolved against the process working_dir.
	// The whole subtree is watched; Exclude prunes it.
	Path string `yaml:"path" json:"path"`

	// Include, when non-empty, restricts triggering to matching files.
	// A pattern containing '/' matches the root-relative path; a pattern
	// without one matches the base name.
	Include []string `yaml:"include,omitempty" json:"include,omitempty"`

	// Exclude drops matching files and prunes matching directories during the
	// walk. Applied on top of the default ignores.
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// IsEnabled reports whether this config watches anything.
func (w *WatchConfig) IsEnabled() bool {
	return w != nil && len(w.Paths) > 0
}

// GetDebounce returns the debounce duration, falling back to the default for a
// missing or malformed value. Use GetDebounceDuration to surface a parse error;
// the loader validates it so that a bad value is reported at load time.
func (w *WatchConfig) GetDebounce() time.Duration {
	d, err := w.GetDebounceDuration()
	if err != nil || d <= 0 {
		return DefaultWatchDebounce
	}
	return d
}

// GetDebounceDuration parses Debounce, reporting a malformed value as an error.
func (w *WatchConfig) GetDebounceDuration() (time.Duration, error) {
	if w == nil || w.Debounce == "" {
		return DefaultWatchDebounce, nil
	}
	return time.ParseDuration(w.Debounce)
}

// GetMaxEntries returns the watched entry cap, defaulting to
// DefaultWatchMaxEntries.
func (w *WatchConfig) GetMaxEntries() int {
	if w == nil || w.MaxEntries <= 0 {
		return DefaultWatchMaxEntries
	}
	return w.MaxEntries
}

// GetBufferSize returns the Windows event buffer size, defaulting to
// DefaultWatchBufferSize.
func (w *WatchConfig) GetBufferSize() int {
	if w == nil || w.BufferSize <= 0 {
		return DefaultWatchBufferSize
	}
	return w.BufferSize
}

// Clone returns a deep copy. Replicas share a ProcessConfig by value, so a
// shallow copy would let per-replica normalization mutate every replica - see
// cloneProcess in the loader.
func (w *WatchConfig) Clone() *WatchConfig {
	if w == nil {
		return nil
	}
	newWatch := *w
	if w.Paths != nil {
		newWatch.Paths = make([]WatchPath, len(w.Paths))
		for i, p := range w.Paths {
			newWatch.Paths[i] = WatchPath{
				Path:    p.Path,
				Include: slices.Clone(p.Include),
				Exclude: slices.Clone(p.Exclude),
			}
		}
	}
	return &newWatch
}
