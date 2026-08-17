package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
)

func stateWithTrigger(name, path string, at time.Time) types.ProcessState {
	stamp := at
	return types.ProcessState{
		Name:             name,
		WatchTriggerPath: path,
		WatchTriggerTime: &stamp,
	}
}

// TestWatchNotifier_IgnoresHistoryOnFirstPass - triggers that predate the TUI
// attaching are history, not news. Announcing them would spam the status bar
// the moment a user attaches to a long-running project.
func TestWatchNotifier_IgnoresHistoryOnFirstPass(t *testing.T) {
	n := newWatchNotifier()
	now := time.Now()
	states := []types.ProcessState{stateWithTrigger("api", "src/main.go", now.Add(-time.Hour))}

	if msg := n.observe(states, now); msg != "" {
		t.Errorf("observe() = %q on the first pass, want no message", msg)
	}
}

func TestWatchNotifier_ReportsNewTrigger(t *testing.T) {
	n := newWatchNotifier()
	now := time.Now()

	n.observe(nil, now) // prime
	msg := n.observe([]types.ProcessState{stateWithTrigger("api", "src/main.go", now)}, now)

	if msg == "" {
		t.Fatal("observe() returned no message for a new trigger")
	}
	for _, want := range []string{"api", "src/main.go"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q does not mention %q", msg, want)
		}
	}
}

// TestWatchNotifier_DoesNotRepeatSameTrigger - the table refreshes on a timer,
// so the same trigger is seen on every tick and must be announced only once.
func TestWatchNotifier_DoesNotRepeatSameTrigger(t *testing.T) {
	n := newWatchNotifier()
	now := time.Now()
	states := []types.ProcessState{stateWithTrigger("api", "src/main.go", now)}

	n.observe(nil, now)
	if msg := n.observe(states, now); msg == "" {
		t.Fatal("first observe returned no message")
	}
	for i := range 5 {
		later := now.Add(time.Duration(i+1) * watchMessageMinInterval)
		if msg := n.observe(states, later); msg != "" {
			t.Errorf("observe() = %q on repeat tick %d, want no message", msg, i)
		}
	}
}

// TestWatchNotifier_CoalescesBurst is the behavior the status bar cannot
// provide on its own: it cancels the previous message the instant a new one
// arrives, so without folding, a burst would flash past unreadably.
func TestWatchNotifier_CoalescesBurst(t *testing.T) {
	n := newWatchNotifier()
	base := time.Now()
	n.observe(nil, base)

	// First trigger is reported immediately.
	first := n.observe([]types.ProcessState{stateWithTrigger("api", "src/a.go", base)}, base)
	if first == "" {
		t.Fatal("first trigger produced no message")
	}

	// Three more arrive inside the quiet interval - all must be withheld.
	for i := range 3 {
		at := base.Add(time.Duration(i+1) * 100 * time.Millisecond)
		states := []types.ProcessState{
			stateWithTrigger("api", "src/a.go", base),
			stateWithTrigger("worker", "src/b.go", at),
		}
		if msg := n.observe(states, at); msg != "" && i < 2 {
			t.Errorf("observe() = %q inside the quiet interval, want it withheld", msg)
		}
		// Each tick reports a distinct new trigger for worker.
		states[1] = stateWithTrigger("worker", "src/b.go", at.Add(time.Millisecond))
	}

	// Once the interval passes, the withheld triggers surface as one summary.
	summary := n.observe(
		[]types.ProcessState{stateWithTrigger("db", "src/c.go", base.Add(time.Second))},
		base.Add(watchMessageMinInterval+time.Second),
	)
	if summary == "" {
		t.Fatal("withheld triggers never surfaced")
	}
	if !strings.Contains(summary, "restarts") {
		t.Errorf("summary %q does not report a count; a burst must not be shown as a single restart", summary)
	}
}

func TestWatchNotifier_NilIsSafe(t *testing.T) {
	var n *watchNotifier
	if msg := n.observe([]types.ProcessState{stateWithTrigger("api", "x", time.Now())}, time.Now()); msg != "" {
		t.Errorf("observe() on a nil notifier = %q, want empty", msg)
	}
}

func TestWatchNotifier_IgnoresStatesWithoutTriggers(t *testing.T) {
	n := newWatchNotifier()
	now := time.Now()
	n.observe(nil, now)

	states := []types.ProcessState{{Name: "api", Status: types.ProcessStateRunning}}
	if msg := n.observe(states, now); msg != "" {
		t.Errorf("observe() = %q for a process with no trigger, want empty", msg)
	}
}
