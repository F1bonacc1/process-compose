package health

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/f1bonacc1/go-health/v2"
	"github.com/f1bonacc1/process-compose/src/command"
)

// tolerance absorbs scheduler jitter on loaded CI runners while staying well
// below the 1s minimum probe period, so a wrong interval choice is still caught.
const tolerance = 500 * time.Millisecond

type fakeChecker struct {
	mu    sync.Mutex
	calls []time.Time
	// fail reports whether the n-th (1-based) check should fail.
	fail func(n int) bool
}

func (f *fakeChecker) Status() (any, error) {
	f.mu.Lock()
	f.calls = append(f.calls, time.Now())
	n := len(f.calls)
	f.mu.Unlock()

	if f.fail(n) {
		return map[string]string{"exit_code": "1"}, errors.New("check failed")
	}
	return map[string]string{"exit_code": "0"}, nil
}

func (f *fakeChecker) snapshot() []time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]time.Time(nil), f.calls...)
}

type blockingChecker struct {
	mu           sync.Mutex
	calls        int
	firstStarted chan struct{}
	releaseFirst chan struct{}
}

func (c *blockingChecker) Status() (any, error) {
	c.mu.Lock()
	c.calls++
	n := c.calls
	c.mu.Unlock()

	if n == 1 {
		close(c.firstStarted)
		<-c.releaseFirst
		return nil, errors.New("stale failure")
	}
	return nil, nil
}

func newTestProber(t *testing.T, probe Probe, checker health.ICheckable, onEnd func(bool, bool, string, any)) *Prober {
	t.Helper()
	probe.Exec = &ExecProbe{Command: "true"}
	p, err := New(t.Name(), probe, nil, *command.DefaultShellConfig(), onEnd)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	p.checker = checker
	return p
}

// assertSchedule asserts the exact number of checks and when each one ran,
// measured from the first check. Offsets are absolute rather than gap-to-gap:
// time.Ticker schedules against a fixed origin, so one late delivery must not
// cascade into a failure for every later check.
func assertSchedule(t *testing.T, calls []time.Time, want []time.Duration) {
	t.Helper()
	if len(calls) != len(want) {
		t.Fatalf("got %d checks, want %d", len(calls), len(want))
	}
	for i, w := range want {
		got := calls[i].Sub(calls[0])
		if got < w-tolerance || got > w+tolerance {
			t.Errorf("check %d ran at +%v, want ~+%v", i+1, got.Round(time.Millisecond), w)
		}
	}
}

func TestProber_StartupPeriodAppliesUntilFirstSuccess(t *testing.T) {
	t.Parallel()
	// Fail twice, then succeed. The first success must switch the poll interval
	// to period_seconds immediately, not one startup interval later.
	checker := &fakeChecker{fail: func(n int) bool { return n < 3 }}
	p := newTestProber(t, Probe{
		StartupPeriodSeconds: 1,
		PeriodSeconds:        5,
		FailureThreshold:     100,
	}, checker, func(bool, bool, string, any) {})

	p.Start()
	// t=0 fail, t=1 fail, t=2 ok, next check not before t=7.
	time.Sleep(3500 * time.Millisecond)
	p.Stop()

	assertSchedule(t, checker.snapshot(), []time.Duration{0, time.Second, 2 * time.Second})
}

func TestProber_SwitchesToPeriodAfterImmediateSuccess(t *testing.T) {
	t.Parallel()
	checker := &fakeChecker{fail: func(int) bool { return false }}
	p := newTestProber(t, Probe{
		StartupPeriodSeconds: 1,
		PeriodSeconds:        4,
		FailureThreshold:     100,
	}, checker, func(bool, bool, string, any) {})

	p.Start()
	// The check at t=0 succeeds, so the next one is due at t=4, not at t=1:
	// the startup interval must not survive the check that ends startup.
	time.Sleep(2500 * time.Millisecond)
	p.Stop()

	if got := len(checker.snapshot()); got != 1 {
		t.Errorf("got %d checks in 2.5s, want 1", got)
	}
}

func TestProber_UnhealthyPeriodAppliesAfterFailure(t *testing.T) {
	t.Parallel()
	// Succeed once, then fail forever.
	checker := &fakeChecker{fail: func(n int) bool { return n > 1 }}
	p := newTestProber(t, Probe{
		StartupPeriodSeconds:   1,
		PeriodSeconds:          1,
		UnhealthyPeriodSeconds: 2,
		FailureThreshold:       100,
	}, checker, func(bool, bool, string, any) {})

	p.Start()
	// t=0 ok, t=1 fail, t=3 fail, next check not before t=5.
	time.Sleep(4 * time.Second)
	p.Stop()

	assertSchedule(t, checker.snapshot(), []time.Duration{0, time.Second, 3 * time.Second})
}

func TestProber_UnhealthyPeriodDefaultsToPeriod(t *testing.T) {
	t.Parallel()
	checker := &fakeChecker{fail: func(n int) bool { return n > 1 }}
	p := newTestProber(t, Probe{
		StartupPeriodSeconds: 1,
		PeriodSeconds:        1,
		FailureThreshold:     100,
	}, checker, func(bool, bool, string, any) {})

	p.Start()
	time.Sleep(2500 * time.Millisecond)
	p.Stop()

	assertSchedule(t, checker.snapshot(), []time.Duration{0, time.Second, 2 * time.Second})
}

func TestProber_FatalReportedOncePerFailureStreak(t *testing.T) {
	t.Parallel()
	checker := &fakeChecker{fail: func(int) bool { return true }}

	var mu sync.Mutex
	fatals := 0
	p := newTestProber(t, Probe{
		StartupPeriodSeconds: 1,
		PeriodSeconds:        1,
		FailureThreshold:     2,
	}, checker, func(_, isFatal bool, _ string, _ any) {
		if isFatal {
			mu.Lock()
			fatals++
			mu.Unlock()
		}
	})

	p.Start()
	// t=0, t=1, t=2 all fail; only the second breaches the threshold. Later
	// checks keep failing without re-reporting, so a longer wait is still valid.
	time.Sleep(3500 * time.Millisecond)
	p.Stop()

	if got := len(checker.snapshot()); got < 3 {
		t.Fatalf("got %d checks, want at least 3", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if fatals != 1 {
		t.Errorf("fatal reported %d times, want 1", fatals)
	}
}

func TestProber_FailureStreakResetsOnSuccess(t *testing.T) {
	t.Parallel()
	// Two failures, a success, then two more -> two distinct streaks, two fatals.
	checker := &fakeChecker{fail: func(n int) bool { return n != 3 }}

	var mu sync.Mutex
	fatals := 0
	p := newTestProber(t, Probe{
		StartupPeriodSeconds: 1,
		PeriodSeconds:        1,
		FailureThreshold:     2,
	}, checker, func(_, isFatal bool, _ string, _ any) {
		if isFatal {
			mu.Lock()
			fatals++
			mu.Unlock()
		}
	})

	p.Start()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := fatals
		mu.Unlock()
		if n >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	p.Stop()

	mu.Lock()
	defer mu.Unlock()
	if fatals != 2 {
		t.Errorf("fatal reported %d times, want 2", fatals)
	}
}

func TestProber_RestartDoesNotLeakMonitoringLoop(t *testing.T) {
	t.Parallel()
	// A restarted process calls Start() again on the same Prober without an
	// intervening Stop(). The previous loop must not survive it.
	checker := &fakeChecker{fail: func(int) bool { return false }}
	p := newTestProber(t, Probe{
		StartupPeriodSeconds: 1,
		PeriodSeconds:        1,
		FailureThreshold:     100,
	}, checker, func(bool, bool, string, any) {})

	p.Start()
	time.Sleep(1500 * time.Millisecond)
	p.Start()
	time.Sleep(500 * time.Millisecond)
	p.Stop()

	before := len(checker.snapshot())
	if before == 0 {
		t.Fatal("prober never ran a check")
	}
	time.Sleep(2500 * time.Millisecond)
	if after := len(checker.snapshot()); after != before {
		t.Errorf("prober kept checking after Stop: %d checks -> %d", before, after)
	}
}

func TestProber_StopIsIdempotentAndRestartable(t *testing.T) {
	t.Parallel()
	checker := &fakeChecker{fail: func(int) bool { return false }}
	p := newTestProber(t, Probe{
		StartupPeriodSeconds: 1,
		PeriodSeconds:        1,
		FailureThreshold:     100,
	}, checker, func(bool, bool, string, any) {})

	p.Stop() // never started
	p.Start()
	time.Sleep(200 * time.Millisecond)
	p.Stop()
	p.Stop()

	before := len(checker.snapshot())
	p.Start()
	time.Sleep(200 * time.Millisecond)
	p.Stop()

	if after := len(checker.snapshot()); after <= before {
		t.Errorf("prober did not restart: %d checks -> %d", before, after)
	}
}

// TestProber_StartDoesNotWaitForBlockedCallback guards the restart path: the
// readiness fatal callback stops the process and can block for the whole
// shutdown timeout. A restarted process calls Start meanwhile, and must be
// probed immediately rather than after the old callback returns.
func TestProber_StartDoesNotWaitForBlockedCallback(t *testing.T) {
	checker := &fakeChecker{fail: func(int) bool { return false }}
	callbackEntered := make(chan struct{})
	releaseCallback := make(chan struct{})
	var once sync.Once
	p := newTestProber(t, Probe{
		StartupPeriodSeconds: 1,
		PeriodSeconds:        1,
	}, checker, func(bool, bool, string, any) {
		blocked := false
		once.Do(func() { blocked = true })
		if blocked {
			close(callbackEntered)
			<-releaseCallback
		}
	})
	defer close(releaseCallback)

	p.Start()
	<-callbackEntered

	restarted := make(chan struct{})
	go func() {
		p.Start()
		close(restarted)
	}()
	select {
	case <-restarted:
	case <-time.After(time.Second):
		t.Fatal("Start blocked on a callback from the superseded loop")
	}

	// The new loop must run its first check without waiting for the callback.
	deadline := time.Now().Add(time.Second)
	for len(checker.snapshot()) < 2 {
		if time.Now().After(deadline) {
			t.Fatal("new loop did not probe while the old callback was blocked")
		}
		time.Sleep(10 * time.Millisecond)
	}
	p.Stop()
}

func TestProber_RestartDiscardsInFlightCheck(t *testing.T) {
	checker := &blockingChecker{
		firstStarted: make(chan struct{}),
		releaseFirst: make(chan struct{}),
	}
	var cbMu sync.Mutex
	failedCallbacks := 0
	p := newTestProber(t, Probe{
		StartupPeriodSeconds:    1,
		PeriodSeconds:           1,
		StartupFailAfterSeconds: 100,
		FailAfterSeconds:        100,
	}, checker, func(isOk, _ bool, _ string, _ any) {
		if !isOk {
			cbMu.Lock()
			failedCallbacks++
			cbMu.Unlock()
		}
	})

	p.Start()
	<-checker.firstStarted

	// Supersede the loop while the first check is blocked.
	p.Start()

	// Wait for the new loop's first (successful) check.
	deadline := time.Now().Add(2 * time.Second)
	for {
		p.mu.Lock()
		healthy := p.wasEverHealthy
		p.mu.Unlock()
		if healthy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("new loop did not complete its first check")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Release the stale check. It must not be counted against the new process
	// instance.
	close(checker.releaseFirst)
	time.Sleep(100 * time.Millisecond)

	p.Stop()
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.wasEverHealthy || !p.isHealthy || p.consecutiveFailures != 0 || p.fatalReported {
		t.Errorf("stale check corrupted state: wasEverHealthy=%v isHealthy=%v failures=%d fatalReported=%v",
			p.wasEverHealthy, p.isHealthy, p.consecutiveFailures, p.fatalReported)
	}
	cbMu.Lock()
	defer cbMu.Unlock()
	if failedCallbacks != 0 {
		t.Errorf("stale check delivered %d failure callbacks, want 0", failedCallbacks)
	}
}

func TestProber_StartupFailAfterBudget(t *testing.T) {
	checker := &fakeChecker{fail: func(int) bool { return true }}

	var mu sync.Mutex
	fatals := 0
	var fatalAt time.Time
	p := newTestProber(t, Probe{
		StartupPeriodSeconds:    1,
		PeriodSeconds:           10,
		StartupFailAfterSeconds: 2,
		FailAfterSeconds:        100,
	}, checker, func(_, isFatal bool, _ string, _ any) {
		if !isFatal {
			return
		}
		mu.Lock()
		fatals++
		fatalAt = time.Now()
		mu.Unlock()
	})

	p.Start()
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := fatals
		mu.Unlock()
		if n >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	p.Stop()

	mu.Lock()
	defer mu.Unlock()
	if fatals != 1 {
		t.Fatalf("fatal reported %d times, want 1", fatals)
	}
	calls := checker.snapshot()
	if len(calls) == 0 {
		t.Fatal("prober never ran a check")
	}
	if elapsed := fatalAt.Sub(calls[0]); elapsed < 2*time.Second-tolerance || elapsed > 2*time.Second+tolerance {
		t.Errorf("gave up after %v, want ~2s from the first failure (startup_fail_after_seconds)", elapsed)
	}
}

func TestProber_DerivedZeroBudgetFatalOnFirstFailure(t *testing.T) {
	checker := &fakeChecker{fail: func(int) bool { return true }}

	var mu sync.Mutex
	fatals := 0
	var fatalAt time.Time
	p := newTestProber(t, Probe{
		StartupPeriodSeconds: 1,
		PeriodSeconds:        10,
		FailureThreshold:     1,
	}, checker, func(_, isFatal bool, _ string, _ any) {
		if !isFatal {
			return
		}
		mu.Lock()
		fatals++
		fatalAt = time.Now()
		mu.Unlock()
	})

	p.Start()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := fatals
		mu.Unlock()
		if n >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	p.Stop()

	calls := checker.snapshot()
	mu.Lock()
	defer mu.Unlock()
	if fatals != 1 {
		t.Fatalf("fatal reported %d times, want 1", fatals)
	}
	if len(calls) == 0 {
		t.Fatal("prober never ran a check")
	}
	if elapsed := fatalAt.Sub(calls[0]); elapsed > tolerance {
		t.Errorf("gave up %v after the first failure, want immediate (failure_threshold: 1)", elapsed)
	}
}

func TestProber_FailAfterBudgetAppliesAfterSuccess(t *testing.T) {
	// Succeed once, then fail forever. The startup budget must not apply after
	// the first success; fail_after_seconds is the deadline from then on.
	checker := &fakeChecker{fail: func(n int) bool { return n > 1 }}

	var mu sync.Mutex
	fatals := 0
	var fatalAt time.Time
	p := newTestProber(t, Probe{
		StartupPeriodSeconds:    1,
		StartupFailAfterSeconds: 100,
		PeriodSeconds:           1,
		UnhealthyPeriodSeconds:  1,
		FailAfterSeconds:        2,
	}, checker, func(_, isFatal bool, _ string, _ any) {
		if !isFatal {
			return
		}
		mu.Lock()
		fatals++
		fatalAt = time.Now()
		mu.Unlock()
	})

	p.Start()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := fatals
		mu.Unlock()
		if n >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	p.Stop()

	mu.Lock()
	defer mu.Unlock()
	if fatals != 1 {
		t.Fatalf("fatal reported %d times, want 1", fatals)
	}
	calls := checker.snapshot()
	if len(calls) < 2 {
		t.Fatalf("got %d checks, want at least 2", len(calls))
	}
	// calls[0] succeeds; calls[1] is the first failure of the steady-state
	// streak. The budget is 2s from that failure.
	if elapsed := fatalAt.Sub(calls[1]); elapsed < 2*time.Second-tolerance || elapsed > 2*time.Second+tolerance {
		t.Errorf("gave up after %v, want ~2s from the first steady-state failure (fail_after_seconds)", elapsed)
	}
}

func TestProber_FastStartupKeepsWallClockGrace(t *testing.T) {
	t.Parallel()
	// period 4s, threshold 2 => give up 4s after the first failure. Polling
	// every second must still give up at ~4s, on the 5th failure, not the 2nd.
	checker := &fakeChecker{fail: func(int) bool { return true }}

	var mu sync.Mutex
	fatals, fatalAtCheck := 0, 0
	var fatalAt time.Time
	p := newTestProber(t, Probe{
		StartupPeriodSeconds: 1,
		PeriodSeconds:        4,
		FailureThreshold:     2,
	}, checker, func(_, isFatal bool, _ string, _ any) {
		if !isFatal {
			return
		}
		n := len(checker.snapshot())
		mu.Lock()
		fatals++
		fatalAtCheck = n
		fatalAt = time.Now()
		mu.Unlock()
	})

	start := time.Now()
	p.Start()
	time.Sleep(5500 * time.Millisecond)
	p.Stop()

	mu.Lock()
	defer mu.Unlock()
	if fatals != 1 {
		t.Fatalf("fatal reported %d times, want 1", fatals)
	}
	if fatalAtCheck != 5 {
		t.Errorf("fatal reported on check %d, want 5", fatalAtCheck)
	}
	if elapsed := fatalAt.Sub(start); elapsed < 4*time.Second-tolerance || elapsed > 4*time.Second+tolerance {
		t.Errorf("gave up after %v, want ~4s (the steady-state grace)", elapsed)
	}
}
