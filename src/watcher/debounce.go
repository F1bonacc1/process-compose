package watcher

import (
	"sync"
	"time"
)

// trigger is one debounced burst of filesystem activity, ready to be turned
// into a restart by the dispatcher.
type trigger struct {
	// proc is the process whose watch fired.
	proc string
	// path is the newest matching path in the burst, used to tell the user why
	// their process restarted.
	path string
	// at is the timestamp of the newest event in the burst. The dispatcher
	// compares it against the target's last restart to drop triggers whose
	// change a running incarnation has already observed.
	at time.Time
	// count is how many matching events were coalesced into this trigger.
	count int
}

// debouncer collapses a burst of filesystem events into a single trailing-edge
// trigger, so that an editor save storm, a `git checkout` or a build writing a
// tree of files produces one restart rather than dozens.
//
// It uses a single time.AfterFunc timer rather than a goroutine per process:
// "fire delay after the last event" is exactly timer.Reset(delay), and a
// coalescing channel would still need a timer to express the trailing edge.
type debouncer struct {
	name  string
	delay time.Duration
	// fire is invoked without the lock held, so the callback may take other
	// locks without inverting this package's lock order.
	fire func(trigger)
	// now is the package's only clock seam. It exists so the deadline recheck
	// below can be tested without sleeping; production always uses time.Now.
	now func() time.Time

	mtx       sync.Mutex
	timer     *time.Timer
	lastEvent time.Time
	lastPath  string
	count     int
	stopped   bool
}

func newDebouncer(name string, delay time.Duration, fire func(trigger)) *debouncer {
	if delay <= 0 {
		delay = defaultDebounce
	}
	return &debouncer{
		name:  name,
		delay: delay,
		fire:  fire,
		now:   time.Now,
	}
}

// notify records a matching event and (re)arms the trailing-edge timer.
func (d *debouncer) notify(path string, at time.Time) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if d.stopped {
		return
	}

	d.lastEvent = at
	d.lastPath = path
	d.count++

	if d.timer == nil {
		d.timer = time.AfterFunc(d.delay, d.onElapsed)
		return
	}
	d.timer.Reset(d.delay)
}

// onElapsed runs when the timer fires. It rechecks the deadline before
// triggering: a Reset racing with an already-elapsed timer can let one stale
// wakeup through, and re-arming for the remainder makes the trailing edge exact
// rather than merely approximate.
func (d *debouncer) onElapsed() {
	d.mtx.Lock()
	if d.stopped || d.count == 0 {
		d.timer = nil
		d.mtx.Unlock()
		return
	}
	if remaining := d.delay - d.now().Sub(d.lastEvent); remaining > 0 {
		d.timer.Reset(remaining)
		d.mtx.Unlock()
		return
	}

	t := trigger{proc: d.name, path: d.lastPath, at: d.lastEvent, count: d.count}
	d.count = 0
	d.lastPath = ""
	d.timer = nil
	d.mtx.Unlock()

	d.fire(t)
}

// stop disarms the debouncer. It is idempotent, and a timer that has already
// slipped past the stopped check does no harm: fire performs a non-blocking
// send on a channel that is never closed.
func (d *debouncer) stop() {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.stopped = true
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	d.count = 0
	d.lastPath = ""
}

// pending reports whether a burst is currently being coalesced. Test helper.
func (d *debouncer) pending() int {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	return d.count
}
