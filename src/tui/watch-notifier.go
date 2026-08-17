package tui

import (
	"fmt"
	"sync"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
)

const (
	// watchMessageDuration is how long a watch message stays on screen.
	watchMessageDuration = 3 * time.Second
	// watchMessageMinInterval is the minimum gap between two watch messages.
	//
	// The status bar cancels the previous message the instant a new one
	// arrives, with no minimum display time, so a burst of restarts would
	// otherwise flash past unreadably. Anything arriving inside this window is
	// folded into a single summary instead.
	watchMessageMinInterval = 2 * time.Second
)

// watchNotifier turns watch-triggered restarts into readable status messages.
//
// Two things make the output legible: the dispatcher already emits one trigger
// per cascade, and this aggregator folds whatever still overlaps into a count.
type watchNotifier struct {
	mtx sync.Mutex
	// seen records the newest trigger already reported per process, so the
	// same restart is not announced on every refresh tick.
	seen map[string]time.Time
	// primed guards the first pass: triggers that predate the TUI attaching are
	// history, not news.
	primed bool

	lastShown time.Time
	// pending accumulates triggers that arrived inside the quiet interval.
	pendingCount int
	pendingName  string
	pendingPath  string
}

func newWatchNotifier() *watchNotifier {
	return &watchNotifier{seen: make(map[string]time.Time)}
}

// observe folds a refresh tick's states into zero or one message. It returns
// the message to display, or "" when there is nothing new to say.
func (n *watchNotifier) observe(states []types.ProcessState, now time.Time) string {
	// The table refresh goroutines can outlive or precede full view setup;
	// a nil notifier simply reports nothing.
	if n == nil {
		return ""
	}
	n.mtx.Lock()
	defer n.mtx.Unlock()

	fresh := 0
	name, path := "", ""
	var newest time.Time
	for _, state := range states {
		if state.WatchTriggerTime == nil {
			continue
		}
		at := *state.WatchTriggerTime
		if prev, ok := n.seen[state.Name]; ok && !at.After(prev) {
			continue
		}
		n.seen[state.Name] = at
		if !n.primed {
			continue
		}
		fresh++
		// Report the most recent trigger of the batch, which is the one the
		// user just caused.
		if name == "" || at.After(newest) {
			name, path, newest = state.Name, state.WatchTriggerPath, at
		}
	}
	if !n.primed {
		n.primed = true
		return ""
	}

	n.pendingCount += fresh
	if fresh > 0 {
		n.pendingName, n.pendingPath = name, path
	}
	if n.pendingCount == 0 {
		return ""
	}
	// Hold the current message for its minimum interval before replacing it.
	if !n.lastShown.IsZero() && now.Sub(n.lastShown) < watchMessageMinInterval {
		return ""
	}

	count := n.pendingCount
	message := fmt.Sprintf("watch: restarting %s ← %s", n.pendingName, n.pendingPath)
	if count > 1 {
		message = fmt.Sprintf("watch: %d restarts (latest: %s ← %s)", count, n.pendingName, n.pendingPath)
	}

	n.pendingCount = 0
	n.lastShown = now
	return message
}
