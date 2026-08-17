package watcher

import "time"

// flapDetector catches a self-sustaining restart loop - the classic case being
// a process that builds into a directory it also watches, so every restart
// writes the files that trigger the next one.
//
// No causality rule can break that loop, because each restart genuinely does
// produce new changes. Rate is the only signal that distinguishes it from a
// developer saving quickly, so the loop is caught by counting restarts in a
// sliding window.
//
// It is owned exclusively by the dispatcher goroutine and therefore holds no
// lock. Do not call it from anywhere else.
type flapDetector struct {
	window    time.Duration
	threshold int
	restarts  map[string][]time.Time
}

func newFlapDetector(window time.Duration, threshold int) *flapDetector {
	return &flapDetector{
		window:    window,
		threshold: threshold,
		restarts:  make(map[string][]time.Time),
	}
}

// record notes a restart and reports whether the process is now flapping.
func (f *flapDetector) record(name string, at time.Time) bool {
	cutoff := at.Add(-f.window)
	kept := f.restarts[name][:0]
	for _, stamp := range f.restarts[name] {
		if stamp.After(cutoff) {
			kept = append(kept, stamp)
		}
	}
	kept = append(kept, at)
	f.restarts[name] = kept
	return len(kept) >= f.threshold
}

// forget clears a process's history, so a watch resumed after suspension or
// reconfigured by a reload starts from a clean slate.
func (f *flapDetector) forget(name string) {
	delete(f.restarts, name)
}
