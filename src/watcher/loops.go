package watcher

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// eventLoop drains fsnotify and routes matching changes to per-process
// debouncers. It exits when the watcher is closed, which closes both channels.
func (w *Watcher) eventLoop() {
	defer w.wg.Done()
	for {
		select {
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			w.handleWatchError(err)
		case <-w.stopCh:
			return
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	if !interesting(event.Op) {
		return
	}
	// fsnotify has no recursive watching on any platform, so a directory
	// created after startup has to be picked up by hand.
	if event.Op.Has(fsnotify.Create) {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			w.adoptNewDir(event.Name)
		}
	}
	if event.Op.Has(fsnotify.Remove) || event.Op.Has(fsnotify.Rename) {
		w.releaseGoneDir(event.Name)
	}
	w.notifyMatching(event.Name, time.Now())
}

// notifyMatching feeds a changed path to every process whose matcher accepts it.
func (w *Watcher) notifyMatching(changed string, at time.Time) {
	w.mtx.RLock()
	targets := make([]*procWatch, 0, 4)
	for _, pw := range w.procs {
		if pw.paused.Load() || pw.suspended.Load() {
			continue
		}
		for _, spec := range pw.roots {
			if spec.matcher.MatchFile(changed) {
				targets = append(targets, pw)
				break
			}
		}
	}
	w.mtx.RUnlock()

	for _, pw := range targets {
		// Absorb the burst a rebuild writes into its own output directory.
		// Off unless the user opts in: this drops events outright, so an edit
		// saved inside the window would be lost.
		if quiesce := pw.quiesceUntil.Load(); quiesce > 0 && at.UnixNano() < quiesce {
			continue
		}
		if at.UnixNano() < pw.watchFrom.Load() {
			continue
		}
		pw.deb.notify(changed, at)
	}
}

// adoptNewDir registers a directory created after startup, then rescans it.
//
// The rescan is what closes the race window between mkdir and the watch being
// registered: anything created in that gap produced no event we could see, so
// without it a `git checkout` or a code generator that creates a directory and
// immediately fills it would be missed.
func (w *Watcher) adoptNewDir(dir string) {
	if w.isStopped() {
		return
	}
	w.mtx.RLock()
	interested := make([]*procWatch, 0, 2)
	for _, pw := range w.procs {
		for _, spec := range pw.roots {
			if under(spec.root, dir) && spec.matcher.MatchDir(dir) {
				interested = append(interested, pw)
				break
			}
		}
	}
	w.mtx.RUnlock()
	if len(interested) == 0 {
		return
	}

	w.mtx.Lock()
	for _, pw := range interested {
		w.retainDirLocked(dir, pw)
	}
	w.mtx.Unlock()

	// Tracked, so Stop cannot return while a rescan is still registering
	// watches. The Add happens before the spawning goroutine's own Done, which
	// is what makes the nesting below safe.
	w.wg.Add(1)
	go w.rescanAfterSettle(dir, interested)
}

func (w *Watcher) rescanAfterSettle(dir string, interested []*procWatch) {
	defer w.wg.Done()
	select {
	case <-time.After(w.opts.RescanSettle):
	case <-w.stopCh:
		return
	}
	if w.isStopped() {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	now := time.Now()
	for _, entry := range entries {
		full := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			w.adoptNewDir(full)
			continue
		}
		for _, pw := range interested {
			if pw.paused.Load() || pw.suspended.Load() {
				continue
			}
			for _, spec := range pw.roots {
				if spec.matcher.MatchFile(full) {
					pw.deb.notify(full, now)
					break
				}
			}
		}
	}
}

// releaseGoneDir drops bookkeeping for a directory that disappeared, along with
// everything beneath it, and refunds the entry budget it was charged against.
// fsnotify drops its own watch, but dirRefs would otherwise drift and leak
// entries against the cap.
func (w *Watcher) releaseGoneDir(dir string) {
	// Every Remove and Rename lands here, the overwhelming majority of them for
	// files, so the miss has to be cheap: a read-locked lookup rather than a
	// write-locked scan of every watched directory.
	//
	// Only a directory we registered can have registered descendants - the walk
	// adds every directory on the way down, and a pruned subtree contributes
	// none - so a path that is not itself watched has nothing beneath it to
	// release.
	w.mtx.RLock()
	_, watched := w.dirRefs[dir]
	w.mtx.RUnlock()
	if !watched {
		return
	}

	w.mtx.Lock()
	defer w.mtx.Unlock()
	for candidate, refs := range w.dirRefs {
		if candidate != dir && !under(dir, candidate) {
			continue
		}
		for name := range refs {
			if pw, ok := w.procs[name]; ok && pw.entries > 0 {
				pw.entries--
			}
		}
		delete(w.dirRefs, candidate)
		delete(w.dirBufSize, candidate)
	}
}

// handleWatchError reports a backend error. An overflow is special: it means we
// cannot know what changed, so every active watch is conservatively triggered
// rather than silently missing a change.
func (w *Watcher) handleWatchError(err error) {
	if err == nil {
		return
	}
	if errors.Is(err, fsnotify.ErrEventOverflow) {
		log.Warn().Msg("File watch event overflow; some changes may have been missed. " +
			"Narrow the watch with 'exclude:', or raise 'watch.buffer_size' on Windows")
		w.triggerAllActive()
		return
	}
	if errors.Is(err, fsnotify.ErrClosed) {
		return
	}
	log.Error().Err(err).Msg("File watch error")
}

func (w *Watcher) triggerAllActive() {
	now := time.Now()
	w.mtx.RLock()
	targets := make([]*procWatch, 0, len(w.procs))
	for _, pw := range w.procs {
		if !pw.paused.Load() && !pw.suspended.Load() {
			targets = append(targets, pw)
		}
	}
	w.mtx.RUnlock()

	for _, pw := range targets {
		pw.deb.notify("(event overflow)", now)
	}
}

// dispatchLoop turns debounced triggers into restarts, one batch at a time.
//
// Serializing here is what makes storm control tractable: the causality map and
// flap detector are owned solely by this goroutine and need no lock.
func (w *Watcher) dispatchLoop() {
	defer w.wg.Done()
	restartedAt := make(map[string]time.Time)
	flap := newFlapDetector(w.opts.FlapWindow, w.opts.FlapThreshold)

	for {
		select {
		case t := <-w.trigger:
			select {
			case <-w.stopCh:
				// Shutting down: do not begin new work.
				return
			default:
			}
			w.handleTrigger(t, restartedAt, flap)
		case <-w.stopCh:
			return
		}
	}
}

func (w *Watcher) handleTrigger(t trigger, restartedAt map[string]time.Time, flap *flapDetector) {
	w.mtx.RLock()
	pw, ok := w.procs[t.proc]
	w.mtx.RUnlock()
	if !ok || pw.paused.Load() || pw.suspended.Load() {
		return
	}

	// Causality: if this process was already restarted after the newest event
	// in the burst, the running incarnation has necessarily observed the change
	// and restarting again would be pure churn. This is what collapses
	// overlapping watches into a single restart.
	if !shouldRestart(t.proc, t.at, restartedAt, time.Unix(0, pw.watchFrom.Load())) {
		return
	}

	names := []string{t.proc}
	if pw.cascade {
		// Dependents are deliberately not filtered by causality. A dependent
		// does not watch this file, so "has it restarted since the file
		// changed?" answers the wrong question - what matters is that it
		// restarts after the process whose output it consumes, which is exactly
		// what this batch is about to do. Filtering here dropped a dependent
		// that its own watch had restarted moments earlier, leaving it running
		// against output that was about to be rebuilt.
		//
		// Nothing is lost by trusting the batch: TransitiveDependents already
		// returns each process once.
		names = append(names, w.ctrl.TransitiveDependents(t.proc)...)
	}

	// Stamp before restarting, never after. The earlier bound can only cause an
	// extra restart; a later one could swallow a real change.
	batchStart := time.Now()
	for _, name := range names {
		restartedAt[name] = batchStart
	}

	reason := t.path
	for _, spec := range pw.roots {
		if under(spec.root, t.path) {
			reason = relativeTo(spec.root, t.path)
			break
		}
	}
	// Checked before announcing the restart, not after: a suspension returns
	// without restarting anything, and a log that says "Restarting api" and
	// then does not is worse than no log at all.
	if flap.record(t.proc, batchStart) {
		pw.suspended.Store(true)
		flap.forget(t.proc)
		log.Error().Msgf("process %q restarted %d times in %v; the last trigger was %s, "+
			"which looks like its own output. Watching is suspended for this process until the "+
			"project is reloaded - add an exclude (e.g. exclude: [\"bin/**\"]) or narrow watch.paths",
			t.proc, w.opts.FlapThreshold, w.opts.FlapWindow, t.path)
		return
	}

	if len(names) > 1 {
		log.Info().Msgf("Restarting %s (+%d dependent(s)): watch triggered by %s",
			t.proc, len(names)-1, reason)
	} else {
		log.Info().Msgf("Restarting %s: watch triggered by %s", t.proc, reason)
	}

	pw.recordTrigger(reason, batchStart)

	if err := w.ctrl.RestartProcesses(names); err != nil {
		log.Error().Err(err).Msgf("failed to restart %s after a file change", t.proc)
	}

	// Suppress the build output the restart is about to produce.
	if w.opts.Quiesce > 0 {
		pw.quiesceUntil.Store(time.Now().Add(w.opts.Quiesce).UnixNano())
	}
}

// shouldRestart applies the causality rule: drop a trigger whose newest event
// predates the target's last restart or the moment its watch was registered.
//
// It applies only to the process that owns the watch that fired. Cascade
// dependents are exempt - see handleTrigger for why.
func shouldRestart(name string, eventAt time.Time, restartedAt map[string]time.Time, watchFrom time.Time) bool {
	effective := restartedAt[name]
	if watchFrom.After(effective) {
		effective = watchFrom
	}
	return !effective.After(eventAt)
}

// under reports whether target lies inside dir.
func under(dir, target string) bool {
	if dir == target {
		return false
	}
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel != ".." && !strings.HasPrefix(rel, "../")
}
