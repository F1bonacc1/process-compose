package watcher

import (
	"sync"
	"testing"
	"time"
)

// triggerRecorder collects fired triggers under a mutex, returning copies. The
// copy-under-lock accessor is what keeps these tests clean under -race.
type triggerRecorder struct {
	mtx      sync.Mutex
	triggers []trigger
	fired    chan struct{}
}

func newTriggerRecorder() *triggerRecorder {
	return &triggerRecorder{fired: make(chan struct{}, 64)}
}

func (r *triggerRecorder) record(t trigger) {
	r.mtx.Lock()
	r.triggers = append(r.triggers, t)
	r.mtx.Unlock()
	select {
	case r.fired <- struct{}{}:
	default:
	}
}

func (r *triggerRecorder) all() []trigger {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	return append([]trigger(nil), r.triggers...)
}

func (r *triggerRecorder) count() int {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	return len(r.triggers)
}

// waitForTrigger blocks until at least one trigger fires or the deadline passes.
func (r *triggerRecorder) waitForTrigger(t *testing.T, timeout time.Duration) bool {
	t.Helper()
	select {
	case <-r.fired:
		return true
	case <-time.After(timeout):
		return false
	}
}

// TestDebouncer_CoalescesBurst is the headline behavior: a save storm, a git
// checkout or a build writing many files must produce exactly one restart.
func TestDebouncer_CoalescesBurst(t *testing.T) {
	rec := newTriggerRecorder()
	d := newDebouncer("api", 40*time.Millisecond, rec.record)
	t.Cleanup(d.stop)

	now := time.Now()
	for i := range 20 {
		d.notify("/repo/src/file.go", now.Add(time.Duration(i)*time.Millisecond))
	}

	if !rec.waitForTrigger(t, 2*time.Second) {
		t.Fatal("debouncer never fired")
	}
	// Give any stray extra timer a chance to misbehave before asserting.
	time.Sleep(100 * time.Millisecond)

	got := rec.all()
	if len(got) != 1 {
		t.Fatalf("fired %d times, want exactly 1", len(got))
	}
	if got[0].count != 20 {
		t.Errorf("trigger.count = %v, want 20", got[0].count)
	}
	if got[0].proc != "api" {
		t.Errorf("trigger.proc = %v, want api", got[0].proc)
	}
}

// TestDebouncer_TrailingEdge pins that the timer measures from the *last* event,
// not the first - otherwise a continuously-writing build would fire mid-build.
func TestDebouncer_TrailingEdge(t *testing.T) {
	rec := newTriggerRecorder()
	d := newDebouncer("api", 80*time.Millisecond, rec.record)
	t.Cleanup(d.stop)

	// Keep the burst hot for longer than the delay.
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		d.notify("/repo/src/file.go", time.Now())
		time.Sleep(10 * time.Millisecond)
		if rec.count() > 0 {
			t.Fatal("debouncer fired while the burst was still arriving")
		}
	}

	if !rec.waitForTrigger(t, 2*time.Second) {
		t.Fatal("debouncer never fired after the burst settled")
	}
}

func TestDebouncer_SeparateBurstsFireSeparately(t *testing.T) {
	rec := newTriggerRecorder()
	d := newDebouncer("api", 30*time.Millisecond, rec.record)
	t.Cleanup(d.stop)

	d.notify("/repo/src/a.go", time.Now())
	if !rec.waitForTrigger(t, 2*time.Second) {
		t.Fatal("first burst never fired")
	}

	d.notify("/repo/src/b.go", time.Now())
	if !rec.waitForTrigger(t, 2*time.Second) {
		t.Fatal("second burst never fired")
	}

	got := rec.all()
	if len(got) != 2 {
		t.Fatalf("fired %d times, want 2", len(got))
	}
	if got[0].path != "/repo/src/a.go" || got[1].path != "/repo/src/b.go" {
		t.Errorf("paths = %v, %v; want a.go then b.go", got[0].path, got[1].path)
	}
}

// TestDebouncer_ReportsNewestPath - the message shown to the user should name
// the most recent file in the burst, not the first one seen.
func TestDebouncer_ReportsNewestPath(t *testing.T) {
	rec := newTriggerRecorder()
	d := newDebouncer("api", 30*time.Millisecond, rec.record)
	t.Cleanup(d.stop)

	now := time.Now()
	d.notify("/repo/src/first.go", now)
	d.notify("/repo/src/second.go", now.Add(time.Millisecond))
	d.notify("/repo/src/third.go", now.Add(2*time.Millisecond))

	if !rec.waitForTrigger(t, 2*time.Second) {
		t.Fatal("debouncer never fired")
	}
	if got := rec.all()[0].path; got != "/repo/src/third.go" {
		t.Errorf("trigger.path = %v, want /repo/src/third.go", got)
	}
}

func TestDebouncer_StopSuppressesPendingBurst(t *testing.T) {
	rec := newTriggerRecorder()
	d := newDebouncer("api", 50*time.Millisecond, rec.record)

	d.notify("/repo/src/file.go", time.Now())
	d.stop()

	time.Sleep(200 * time.Millisecond)
	if got := rec.count(); got != 0 {
		t.Errorf("fired %d times after stop, want 0", got)
	}
}

func TestDebouncer_StopIsIdempotentAndBlocksLaterNotify(t *testing.T) {
	rec := newTriggerRecorder()
	d := newDebouncer("api", 20*time.Millisecond, rec.record)

	d.stop()
	d.stop()
	d.notify("/repo/src/file.go", time.Now())

	time.Sleep(150 * time.Millisecond)
	if got := rec.count(); got != 0 {
		t.Errorf("fired %d times after stop, want 0", got)
	}
	if got := d.pending(); got != 0 {
		t.Errorf("pending() = %v after stop, want 0", got)
	}
}

// TestDebouncer_ReArmsWhenBurstStillHot exercises the deadline recheck directly.
// A Reset racing an already-elapsed timer can let one stale wakeup through; the
// recheck must re-arm rather than fire early. The injected clock reports a time
// that makes the burst look hot, so onElapsed must not produce a trigger.
func TestDebouncer_ReArmsWhenBurstStillHot(t *testing.T) {
	rec := newTriggerRecorder()
	d := newDebouncer("api", time.Hour, rec.record)
	t.Cleanup(d.stop)

	base := time.Now()
	d.now = func() time.Time { return base }
	d.notify("/repo/src/file.go", base)

	// Simulate the stale wakeup: the timer's callback runs even though only an
	// instant has passed since the last event.
	d.onElapsed()

	if got := rec.count(); got != 0 {
		t.Errorf("fired %d times on a stale wakeup, want 0", got)
	}
	if got := d.pending(); got != 1 {
		t.Errorf("pending() = %v, want the burst to still be held", got)
	}

	// Advance the clock past the delay; now the same wakeup must fire.
	d.now = func() time.Time { return base.Add(2 * time.Hour) }
	d.onElapsed()

	if got := rec.count(); got != 1 {
		t.Errorf("fired %d times once the delay elapsed, want 1", got)
	}
}

func TestNewDebouncer_NonPositiveDelayFallsBack(t *testing.T) {
	d := newDebouncer("api", 0, func(trigger) {})
	if d.delay != defaultDebounce {
		t.Errorf("delay = %v, want %v", d.delay, defaultDebounce)
	}
	d = newDebouncer("api", -time.Second, func(trigger) {})
	if d.delay != defaultDebounce {
		t.Errorf("delay = %v, want %v", d.delay, defaultDebounce)
	}
}
