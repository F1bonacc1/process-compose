package api

import (
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
)

func TestStateWsObserver_NilFilterDeliversAll(t *testing.T) {
	o := newStateWsObserver(4, nil)
	o.Notify(types.ProcessStateEvent{State: types.ProcessState{Name: "a"}})
	o.Notify(types.ProcessStateEvent{State: types.ProcessState{Name: "b"}})

	if got := len(o.events); got != 2 {
		t.Fatalf("nil filter should deliver everything, got %d events", got)
	}
}

func TestStateWsObserver_FilterDropsUnmatched(t *testing.T) {
	filter := map[string]struct{}{"a": {}, "c": {}}
	o := newStateWsObserver(4, filter)
	o.Notify(types.ProcessStateEvent{State: types.ProcessState{Name: "a"}})
	o.Notify(types.ProcessStateEvent{State: types.ProcessState{Name: "b"}})
	o.Notify(types.ProcessStateEvent{State: types.ProcessState{Name: "c"}})

	if got := len(o.events); got != 2 {
		t.Fatalf("filter should drop 'b', got %d events", got)
	}

	gotNames := []string{}
	for range 2 {
		ev := <-o.events
		gotNames = append(gotNames, ev.State.Name)
	}
	if gotNames[0] != "a" || gotNames[1] != "c" {
		t.Fatalf("filter should preserve order of matching events, got %v", gotNames)
	}
}

func TestStateWsObserver_BackpressureClosesObserver(t *testing.T) {
	o := newStateWsObserver(1, nil)
	o.Notify(types.ProcessStateEvent{State: types.ProcessState{Name: "a"}})
	// Second Notify should fail to enqueue and signal close.
	o.Notify(types.ProcessStateEvent{State: types.ProcessState{Name: "b"}})

	select {
	case <-o.closeCh:
	default:
		t.Fatal("expected closeCh to be signalled on backpressure")
	}

	// Subsequent Notify is a no-op (closed observer must not panic).
	o.Notify(types.ProcessStateEvent{State: types.ProcessState{Name: "c"}})
}
