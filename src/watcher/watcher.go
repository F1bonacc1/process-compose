package watcher

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// ProjectController is the narrow slice of the project runner the watcher
// needs. Keeping it this small mirrors scheduler.ProcessStarter and is what
// lets the whole subsystem be tested without a ProjectRunner.
type ProjectController interface {
	// RestartProcesses restarts names in the given order, suppressing project
	// completion for the duration of the whole batch.
	RestartProcesses(names []string) error

	// TransitiveDependents returns every process that depends on name, directly
	// or transitively, in dependency order. name itself is excluded.
	TransitiveDependents(name string) []string

	// GetProcessState is used to skip processes that have gone away.
	GetProcessState(name string) (*types.ProcessState, error)
}

// Watcher restarts processes when their watched files change.
//
// A single fsnotify.Watcher backs the whole project rather than one per
// process: on kqueue every watched entry costs a file descriptor, so two
// processes watching the same tree must share registrations. That sharing is
// what dirRefs tracks.
//
// The watcher deliberately lives above Process, which is destroyed and
// recreated on every restart - a watcher owned by Process would tear down and
// re-walk its tree on every restart it had just triggered.
type Watcher struct {
	ctrl ProjectController
	opts Options
	fsw  *fsnotify.Watcher

	mtx     sync.RWMutex
	procs   map[string]*procWatch
	dirRefs map[string]map[string]struct{}
	// dirBufSize records the event buffer each directory was registered with,
	// so that a process asking for a larger one upgrades the existing
	// registration rather than silently inheriting a smaller buffer from
	// whichever process happened to claim the directory first. Windows only in
	// effect; every other backend ignores the size.
	dirBufSize map[string]int

	// trigger is buffered and NEVER closed. That single property is what makes
	// shutdown unable to panic: a timer that slips past the stopped check does
	// a non-blocking send into a live channel and is harmlessly discarded.
	trigger chan trigger

	stopCh   chan struct{}
	stopOnce sync.Once
	started  atomic.Bool
	wg       sync.WaitGroup
}

// procWatch is one process's registration.
type procWatch struct {
	name    string
	cascade bool
	roots   []rootSpec
	deb     *debouncer
	// bufferSize is this process's requested ReadDirectoryChangesW buffer,
	// carried here because directories adopted after startup are registered
	// long after the config that asked for it.
	bufferSize int
	// maxEntries and entries bound how many directories this process may watch.
	// The initial scan is capped by scanTree, but directories created while the
	// project runs are adopted one at a time and need their own budget, or a
	// code generator or an untarred dependency tree walks straight past the cap
	// and exhausts the inotify limit. Both are guarded by Watcher.mtx.
	maxEntries int
	entries    int
	// capWarned keeps the cap from logging on every subsequent mkdir.
	capWarned bool

	// paused is set while the process is stopped, so a change does not restart
	// something the user deliberately stopped. The tree stays registered.
	paused atomic.Bool
	// suspended is set when flap detection fires. Unlike paused it is not a
	// user action, and it clears on reload.
	suspended atomic.Bool
	// watchFrom (UnixNano) drops events caused by churn that predates
	// registration. Atomic because ResumeProcess rewrites it from an API or TUI
	// goroutine while the event loop is reading it.
	watchFrom atomic.Int64
	// quiesceUntil (UnixNano) suppresses events for a moment after a restart.
	// Off by default - see Options.Quiesce for why.
	quiesceUntil atomic.Int64

	// lastTrigger records the change that last restarted this process, for
	// display. Guarded by triggerMtx.
	triggerMtx  sync.Mutex
	triggerPath string
	triggerAt   time.Time
}

// TriggerInfo describes the file change that last restarted a process.
type TriggerInfo struct {
	Path string
	At   time.Time
}

// LastTrigger returns the change that last restarted name, if any. Used to tell
// the user which save caused a restart.
func (w *Watcher) LastTrigger(name string) (TriggerInfo, bool) {
	w.mtx.RLock()
	pw, ok := w.procs[name]
	w.mtx.RUnlock()
	if !ok {
		return TriggerInfo{}, false
	}
	pw.triggerMtx.Lock()
	defer pw.triggerMtx.Unlock()
	if pw.triggerAt.IsZero() {
		return TriggerInfo{}, false
	}
	return TriggerInfo{Path: pw.triggerPath, At: pw.triggerAt}, true
}

func (pw *procWatch) recordTrigger(path string, at time.Time) {
	pw.triggerMtx.Lock()
	defer pw.triggerMtx.Unlock()
	pw.triggerPath = path
	pw.triggerAt = at
}

// rootSpec is a single watched root and its compiled matcher.
type rootSpec struct {
	root    string
	matcher *matcher
	// configured is the path as the user wrote it, for messages.
	configured string
}

// New creates a watcher. It does not register anything or start any goroutine
// until AddProcess and Start are called.
func New(ctrl ProjectController, opts Options) (*Watcher, error) {
	opts = opts.withDefaults()
	fsw, err := fsnotify.NewBufferedWatcher(opts.EventBuffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}
	return &Watcher{
		ctrl:       ctrl,
		opts:       opts,
		fsw:        fsw,
		procs:      make(map[string]*procWatch),
		dirRefs:    make(map[string]map[string]struct{}),
		dirBufSize: make(map[string]int),
		trigger:    make(chan trigger, triggerBufferSize),
		stopCh:     make(chan struct{}),
	}, nil
}

// AddProcess registers a process's watch. The config is deep-copied and its
// matchers compiled once, so a later project reload cannot race the event loop
// through a shared pointer.
func (w *Watcher) AddProcess(name string, cfg *types.WatchConfig) error {
	if !cfg.IsEnabled() {
		return nil
	}
	cfg = cfg.Clone()

	// Phase 1: resolve and walk lock-free. Walking a large tree under the lock
	// would stall every event for its duration.
	specs := make([]rootSpec, 0, len(cfg.Paths))
	dirs := make([]string, 0, 64)
	remaining := cfg.GetMaxEntries()

	for _, watchPath := range cfg.Paths {
		dir, onlyBase, err := resolveRoot(watchPath.Path)
		if err != nil {
			return fmt.Errorf("process %q: watch path %q: %w", name, watchPath.Path, err)
		}
		m, err := newMatcher(dir, matcherOpts{
			include:       watchPath.Include,
			exclude:       watchPath.Exclude,
			useDefaults:   !cfg.DisableDefaultExcludes,
			alwaysExclude: w.opts.AlwaysExclude,
			onlyBase:      onlyBase,
		})
		if err != nil {
			return fmt.Errorf("process %q: %w", name, err)
		}
		found, err := scanTree(name, dir, m, remaining)
		if err != nil {
			return err
		}
		remaining -= len(found)
		if remaining < 0 {
			return &TooManyEntriesError{
				Process: name, Root: dir,
				Scanned: cfg.GetMaxEntries() - remaining, Max: cfg.GetMaxEntries(),
			}
		}
		specs = append(specs, rootSpec{root: dir, matcher: m, configured: watchPath.Path})
		dirs = append(dirs, found...)
	}

	debounce := cfg.GetDebounce()
	if cfg.Debounce == "" {
		debounce = w.opts.DefaultDebounce
	}

	pw := &procWatch{
		name:       name,
		cascade:    cfg.Cascade,
		roots:      specs,
		bufferSize: cfg.GetBufferSize(),
		maxEntries: cfg.GetMaxEntries(),
	}
	pw.watchFrom.Store(time.Now().UnixNano())
	pw.deb = newDebouncer(name, debounce, w.fire)

	// Phase 2: publish and register under the lock. Only map operations and
	// fast syscalls happen here.
	w.mtx.Lock()
	if existing, ok := w.procs[name]; ok {
		w.mtx.Unlock()
		// Replacing an existing registration: drop the old one first so its
		// directory references are released, then retry.
		existing.deb.stop()
		_ = w.RemoveProcess(name)
		w.mtx.Lock()
	}
	w.procs[name] = pw
	for _, dir := range dirs {
		w.retainDirLocked(dir, pw)
	}
	count := len(dirs)
	w.mtx.Unlock()

	log.Info().Msgf("Watching %d path(s) for process %s (%d directories)", len(specs), name, count)
	return nil
}

// RemoveProcess deregisters a process and releases any directory watch no
// longer referenced by another process.
func (w *Watcher) RemoveProcess(name string) error {
	w.mtx.Lock()
	pw, ok := w.procs[name]
	if !ok {
		w.mtx.Unlock()
		return nil
	}
	delete(w.procs, name)

	released := make([]string, 0)
	for dir, refs := range w.dirRefs {
		if _, referenced := refs[name]; !referenced {
			continue
		}
		delete(refs, name)
		if len(refs) == 0 {
			delete(w.dirRefs, dir)
			delete(w.dirBufSize, dir)
			released = append(released, dir)
		}
	}
	w.mtx.Unlock()

	pw.deb.stop()
	for _, dir := range released {
		// A directory that has already gone away takes its watch with it.
		if err := w.fsw.Remove(dir); err != nil && !errors.Is(err, fsnotify.ErrNonExistentWatch) {
			log.Debug().Err(err).Msgf("failed to remove watch on %s", dir)
		}
	}
	return nil
}

// PauseProcess suspends triggering without releasing the tree, so a stopped
// process can be restarted later without paying for a full re-walk.
func (w *Watcher) PauseProcess(name string) error {
	w.mtx.RLock()
	pw, ok := w.procs[name]
	w.mtx.RUnlock()
	if !ok {
		return nil
	}
	pw.paused.Store(true)
	return nil
}

// ResumeProcess re-enables triggering and clears any flap suspension.
func (w *Watcher) ResumeProcess(name string) error {
	w.mtx.RLock()
	pw, ok := w.procs[name]
	w.mtx.RUnlock()
	if !ok {
		return nil
	}
	pw.paused.Store(false)
	pw.suspended.Store(false)
	pw.watchFrom.Store(time.Now().UnixNano())
	return nil
}

// IsRegistered reports whether a watch registration exists for name, armed or
// not.
//
// This is the counterpart of Scheduler.IsScheduled, and the distinction from
// IsWatched matters: a paused registration is not armed but is still fully
// built, so resuming it must not be confused with never having registered it.
// Re-adding instead would re-walk the whole tree for nothing.
func (w *Watcher) IsRegistered(name string) bool {
	w.mtx.RLock()
	defer w.mtx.RUnlock()
	_, ok := w.procs[name]
	return ok
}

// IsWatched reports whether an active watch is armed for name. A paused or
// suspended watch is not active - it cannot restart anything, so it must not
// make the process read as Watching nor hold the project open.
func (w *Watcher) IsWatched(name string) bool {
	w.mtx.RLock()
	defer w.mtx.RUnlock()
	pw, ok := w.procs[name]
	return ok && !pw.paused.Load() && !pw.suspended.Load()
}

// GetWatchedProcesses returns the actively watched process names.
func (w *Watcher) GetWatchedProcesses() []string {
	w.mtx.RLock()
	defer w.mtx.RUnlock()
	names := make([]string, 0, len(w.procs))
	for name, pw := range w.procs {
		if !pw.paused.Load() && !pw.suspended.Load() {
			names = append(names, name)
		}
	}
	return names
}

// WatchedEntryCount reports how many directories carry a watch.
func (w *Watcher) WatchedEntryCount() int {
	w.mtx.RLock()
	defer w.mtx.RUnlock()
	return len(w.dirRefs)
}

// bufSizeOf reports the event buffer a directory was registered with. Test
// helper - the size is only observable on Windows otherwise.
func (w *Watcher) bufSizeOf(dir string) int {
	w.mtx.RLock()
	defer w.mtx.RUnlock()
	return w.dirBufSize[dir]
}

// Start launches the event and dispatch loops. It is safe to call once.
func (w *Watcher) Start() {
	if !w.started.CompareAndSwap(false, true) {
		return
	}
	w.wg.Add(2)
	go w.eventLoop()
	go w.dispatchLoop()
}

// isStopped reports whether shutdown has begun. Work that outlives an event -
// a deferred rescan, say - checks it before touching the fsnotify watcher,
// which Stop has by then closed.
func (w *Watcher) isStopped() bool {
	select {
	case <-w.stopCh:
		return true
	default:
		return false
	}
}

// Stop shuts the watcher down. It is idempotent, and must never be called from
// the event or dispatch loop - wg.Wait would deadlock on itself.
func (w *Watcher) Stop() error {
	var err error
	w.stopOnce.Do(func() {
		close(w.stopCh)
		err = w.fsw.Close()

		w.mtx.Lock()
		for _, pw := range w.procs {
			pw.deb.stop()
		}
		w.mtx.Unlock()

		w.wg.Wait()
	})
	return err
}

// retainDirLocked adds a reference to dir for pw, registering the watch the
// first time anyone asks for it and charging it against pw's entry budget.
// Caller holds mtx.
func (w *Watcher) retainDirLocked(dir string, pw *procWatch) {
	bufSize := pw.bufferSize
	if bufSize <= 0 {
		bufSize = w.opts.WindowsBufferSize
	}

	refs, exists := w.dirRefs[dir]
	if exists {
		if _, held := refs[pw.name]; held {
			w.resizeBufferLocked(dir, bufSize)
			return
		}
	}
	if pw.entries >= pw.maxEntries {
		if !pw.capWarned {
			pw.capWarned = true
			log.Error().Msgf("process %q has reached its watch limit of %d directories; %s "+
				"and anything created under it will not be watched. Add an exclude for it, "+
				"or raise watch.max_entries", pw.name, pw.maxEntries, dir)
		}
		return
	}

	if !exists {
		refs = make(map[string]struct{})
		w.dirRefs[dir] = refs
		if err := w.addWatch(dir, bufSize); err != nil {
			delete(w.dirRefs, dir)
			log.Error().Err(err).Msgf("failed to watch directory %s", dir)
			return
		}
		w.dirBufSize[dir] = bufSize
	} else {
		w.resizeBufferLocked(dir, bufSize)
	}
	refs[pw.name] = struct{}{}
	pw.entries++
}

// resizeBufferLocked raises a directory's event buffer when a process needs a
// larger one than it was registered with. The larger request wins, since the
// buffer only ever guards against dropped events. Caller holds mtx.
func (w *Watcher) resizeBufferLocked(dir string, bufSize int) {
	if bufSize <= w.dirBufSize[dir] {
		return
	}
	// Re-registering an existing path replaces its buffer in place.
	if err := w.addWatch(dir, bufSize); err != nil {
		log.Error().Err(err).Msgf("failed to resize the event buffer for %s", dir)
		return
	}
	w.dirBufSize[dir] = bufSize
}

// addWatch registers a directory with the given event buffer size. The buffer
// matters on Windows, where ReadDirectoryChangesW silently drops events once
// its fixed buffer overflows under churn - a git checkout or an npm install
// will do it. Every other backend ignores the size.
func (w *Watcher) addWatch(dir string, bufSize int) error {
	if err := w.fsw.AddWith(dir, fsnotify.WithBufferSize(bufSize)); err != nil {
		return translateWatchError(err, dir)
	}
	return nil
}

// fire is the debouncer callback. The send is non-blocking so a slow restart
// can never stall the event loop; a dropped trigger is superseded by the next
// burst, which is the correct trade for a watcher.
func (w *Watcher) fire(t trigger) {
	select {
	case w.trigger <- t:
	default:
		log.Debug().Msgf("watch trigger queue full, dropping trigger for %s", t.proc)
	}
}
