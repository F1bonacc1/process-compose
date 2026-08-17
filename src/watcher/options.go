package watcher

import (
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
)

const (
	defaultDebounce    = types.DefaultWatchDebounce
	defaultMaxEntries  = types.DefaultWatchMaxEntries
	defaultBufferSize  = types.DefaultWatchBufferSize
	defaultEventBuffer = 4096

	// defaultRescanSettle is how long a newly created directory is given to
	// settle before it is walked. The walk catches sub-directories and files
	// created in the race window between mkdir and watch registration, which
	// fsnotify cannot report because the watch did not exist yet.
	defaultRescanSettle = 20 * time.Millisecond

	// Flap detection bounds. A process rebuilding into its own watched tree is
	// self-sustaining and no causality rule can break it, so the loop is caught
	// by rate instead.
	defaultFlapWindow    = time.Minute
	defaultFlapThreshold = 10

	// triggerBufferSize bounds the dispatcher's queue. Sends are non-blocking,
	// so an overflow drops a trigger that a newer burst will supersede rather
	// than stalling the event loop behind a slow restart.
	triggerBufferSize = 256
)

// Options tunes the watcher. Every interval is a field rather than a constant
// so tests can run in milliseconds - this package has no clock injection, so
// short real durations are the only lever.
type Options struct {
	// MaxEntries caps watched directories per process. 0 uses the default.
	MaxEntries int
	// DefaultDebounce applies to processes that set no debounce of their own.
	DefaultDebounce time.Duration
	// EventBuffer sizes the shared fsnotify event channel.
	EventBuffer uint
	// WindowsBufferSize is the ReadDirectoryChangesW buffer. Windows only.
	WindowsBufferSize int
	// RescanSettle delays the walk of a newly created directory.
	RescanSettle time.Duration
	// Quiesce suppresses events for this long after a restart, absorbing the
	// output a rebuild writes into its own watched tree.
	//
	// Off (0) by default, deliberately: suppression *drops* events rather than
	// deferring them, so an edit saved inside the window would be silently
	// lost. Losing a save is worse than an extra restart, and the two non-lossy
	// defences - default excludes covering build output directories, and flap
	// detection catching a genuine loop loudly - cover the same ground. Set it
	// explicitly if a build writes somewhere the excludes cannot express.
	Quiesce time.Duration
	// FlapWindow and FlapThreshold bound the restart rate before a watch is
	// suspended as a suspected feedback loop.
	FlapWindow    time.Duration
	FlapThreshold int
	// AlwaysExclude holds absolute paths dropped regardless of a process's
	// disable_default_excludes - process-compose's own log files.
	AlwaysExclude []string
}

func (o Options) withDefaults() Options {
	if o.MaxEntries <= 0 {
		o.MaxEntries = defaultMaxEntries
	}
	if o.DefaultDebounce <= 0 {
		o.DefaultDebounce = defaultDebounce
	}
	if o.EventBuffer == 0 {
		o.EventBuffer = defaultEventBuffer
	}
	if o.WindowsBufferSize <= 0 {
		o.WindowsBufferSize = defaultBufferSize
	}
	if o.RescanSettle <= 0 {
		o.RescanSettle = defaultRescanSettle
	}
	if o.Quiesce < 0 {
		o.Quiesce = 0
	}
	if o.FlapWindow <= 0 {
		o.FlapWindow = defaultFlapWindow
	}
	if o.FlapThreshold <= 0 {
		o.FlapThreshold = defaultFlapThreshold
	}
	return o
}
