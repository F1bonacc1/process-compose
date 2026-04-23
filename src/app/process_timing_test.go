package app

import (
	"sync"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
)

// newBareProcessForTimingTests builds the minimum Process needed to exercise
// the timing-tracking paths (setProcHealth, onProcessEnd bookkeeping) in
// isolation -- i.e. without actually spawning a child or starting probes.
func newBareProcessForTimingTests() *Process {
	return &Process{
		procConf:  &types.ProcessConfig{Name: "p", ReplicaName: "p"},
		procState: &types.ProcessState{},
	}
}

func TestSetProcHealth_SetsReadyTimeOnce(t *testing.T) {
	p := newBareProcessForTimingTests()

	before := time.Now()
	p.setProcHealth(types.ProcessHealthReady)
	after := time.Now()

	if p.procState.Health != types.ProcessHealthReady {
		t.Fatalf("Health = %q, want %q", p.procState.Health, types.ProcessHealthReady)
	}
	if p.procState.ProcessReadyTime == nil {
		t.Fatal("ProcessReadyTime was not set")
	}
	got := *p.procState.ProcessReadyTime
	if got.Before(before) || got.After(after) {
		t.Errorf("ProcessReadyTime = %v, want between %v and %v", got, before, after)
	}

	// Calling setProcHealth(Ready) again must not overwrite the first timestamp.
	first := *p.procState.ProcessReadyTime
	time.Sleep(2 * time.Millisecond)
	p.setProcHealth(types.ProcessHealthReady)
	if !p.procState.ProcessReadyTime.Equal(first) {
		t.Errorf("ProcessReadyTime changed on second Ready call: first=%v second=%v",
			first, *p.procState.ProcessReadyTime)
	}
}

func TestSetProcHealth_NotReadyDoesNotSetTime(t *testing.T) {
	p := newBareProcessForTimingTests()
	p.setProcHealth(types.ProcessHealthNotReady)
	if p.procState.ProcessReadyTime != nil {
		t.Errorf("ProcessReadyTime unexpectedly set for NotReady: %v",
			*p.procState.ProcessReadyTime)
	}
}

// TestSetProcHealth_ConcurrentReadyNoRace exercises the mutex protection the
// maintainer asked for. Running under `go test -race` / `make testrace` should
// flag any unsynchronised access to procState.
func TestSetProcHealth_ConcurrentReadyNoRace(t *testing.T) {
	p := newBareProcessForTimingTests()

	const workers = 32
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			p.setProcHealth(types.ProcessHealthReady)
		}()
	}
	wg.Wait()

	if p.procState.ProcessReadyTime == nil {
		t.Fatal("ProcessReadyTime not set after concurrent Ready calls")
	}
	if p.procState.Health != types.ProcessHealthReady {
		t.Errorf("Health = %q, want Ready", p.procState.Health)
	}
}
