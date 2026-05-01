package app

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
)

// captureObserver is a test-only observer that records every event it
// receives. It buffers everything so that synchronous Notify under the
// broadcaster's lock cannot block.
type captureObserver struct {
	id   string
	mu   sync.Mutex
	recv []types.ProcessStateEvent
}

func newCaptureObserver(id string) *captureObserver {
	return &captureObserver{id: id}
}

func (o *captureObserver) UniqueID() string { return o.id }

func (o *captureObserver) Notify(ev types.ProcessStateEvent) {
	o.mu.Lock()
	o.recv = append(o.recv, ev)
	o.mu.Unlock()
}

func (o *captureObserver) events() []types.ProcessStateEvent {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]types.ProcessStateEvent, len(o.recv))
	copy(out, o.recv)
	return out
}

func makeState(name, status string) types.ProcessState {
	return types.ProcessState{Name: name, Status: status}
}

func TestBroadcaster_SubscribeAndPublish(t *testing.T) {
	b := NewProcessStateBroadcaster(nil)
	o := newCaptureObserver("a")
	b.Subscribe(o)

	ev := types.ProcessStateEvent{State: makeState("p1", types.ProcessStateRunning)}
	b.Publish(ev)

	got := o.events()
	if len(got) != 1 {
		t.Fatalf("want 1 event, got %d", len(got))
	}
	if got[0].State.Name != "p1" || got[0].State.Status != types.ProcessStateRunning {
		t.Fatalf("unexpected event payload: %+v", got[0])
	}
	if got[0].Snapshot {
		t.Fatalf("Subscribe (without snapshot) should not produce snapshot events")
	}
}

func TestBroadcaster_SubscribeWithSnapshot(t *testing.T) {
	snap := []types.ProcessState{
		makeState("p1", types.ProcessStateRunning),
		makeState("p2", types.ProcessStatePending),
	}
	b := NewProcessStateBroadcaster(func() []types.ProcessState { return snap })

	o := newCaptureObserver("a")
	b.SubscribeWithSnapshot(o)

	got := o.events()
	if len(got) != 2 {
		t.Fatalf("want 2 snapshot events, got %d", len(got))
	}
	for _, ev := range got {
		if !ev.Snapshot {
			t.Fatalf("snapshot events must have Snapshot=true: %+v", ev)
		}
	}

	// Subsequent publishes should still arrive, and not be marked as snapshot.
	b.Publish(types.ProcessStateEvent{State: makeState("p1", types.ProcessStateCompleted)})
	got = o.events()
	if len(got) != 3 {
		t.Fatalf("want 3 events after publish, got %d", len(got))
	}
	if got[2].Snapshot {
		t.Fatalf("live event after snapshot must not be marked snapshot")
	}
}

func TestBroadcaster_Unsubscribe(t *testing.T) {
	b := NewProcessStateBroadcaster(nil)
	o := newCaptureObserver("a")
	b.Subscribe(o)
	b.Unsubscribe(o)

	b.Publish(types.ProcessStateEvent{State: makeState("p1", types.ProcessStateRunning)})

	if got := o.events(); len(got) != 0 {
		t.Fatalf("unsubscribed observer should receive nothing, got %d events", len(got))
	}
}

func TestBroadcaster_SnapshotAtomicWithSubscribe(t *testing.T) {
	// Verify that no Publish call can interleave between the snapshot delivery
	// and the subscription registration. We do this by hammering Publish from a
	// background goroutine and checking the snapshot is ordered before any
	// post-subscribe deltas, with no duplicate.
	const publishers = 8
	const events = 200

	var counter atomic.Int64
	snap := func() []types.ProcessState {
		// Snapshot as a single fixed state.
		return []types.ProcessState{makeState("p1", types.ProcessStatePending)}
	}
	b := NewProcessStateBroadcaster(snap)

	var wg sync.WaitGroup
	wg.Add(publishers)
	for i := 0; i < publishers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < events; j++ {
				n := counter.Add(1)
				b.Publish(types.ProcessStateEvent{
					State: makeState("p1", strconv.FormatInt(n, 10)),
				})
			}
		}()
	}

	o := newCaptureObserver("a")
	b.SubscribeWithSnapshot(o)
	wg.Wait()
	// Drain anything still queued; Publish is synchronous so by this point all
	// events that happened *after* SubscribeWithSnapshot are in o.recv.

	got := o.events()
	if len(got) == 0 || !got[0].Snapshot {
		t.Fatalf("first event must be a snapshot, got: %+v", got)
	}
	for i, ev := range got[1:] {
		if ev.Snapshot {
			t.Fatalf("event #%d after snapshot is unexpectedly a snapshot: %+v", i+1, ev)
		}
	}
}

func TestBroadcaster_ConcurrentPublish(t *testing.T) {
	b := NewProcessStateBroadcaster(nil)
	const observers = 4
	const publishers = 4
	const eventsPer = 100

	obs := make([]*captureObserver, observers)
	for i := range obs {
		obs[i] = newCaptureObserver(strconv.Itoa(i))
		b.Subscribe(obs[i])
	}

	var wg sync.WaitGroup
	wg.Add(publishers)
	for p := 0; p < publishers; p++ {
		go func() {
			defer wg.Done()
			for i := 0; i < eventsPer; i++ {
				b.Publish(types.ProcessStateEvent{
					State: makeState("p", types.ProcessStateRunning),
				})
			}
		}()
	}
	wg.Wait()

	want := publishers * eventsPer
	for i, o := range obs {
		if got := len(o.events()); got != want {
			t.Errorf("observer %d: want %d events, got %d", i, want, got)
		}
	}
}

func TestBroadcaster_NilSnapshotFunc(t *testing.T) {
	// SubscribeWithSnapshot must not panic when the snapshot func is nil.
	b := NewProcessStateBroadcaster(nil)
	o := newCaptureObserver("a")
	b.SubscribeWithSnapshot(o)
	b.Publish(types.ProcessStateEvent{State: makeState("p1", types.ProcessStateRunning)})

	got := o.events()
	if len(got) != 1 || got[0].Snapshot {
		t.Fatalf("nil snapshot should yield only the live event: %+v", got)
	}
}

func TestBroadcaster_Close(t *testing.T) {
	b := NewProcessStateBroadcaster(nil)
	o := newCaptureObserver("a")
	b.Subscribe(o)
	b.Close()
	b.Publish(types.ProcessStateEvent{State: makeState("p1", types.ProcessStateRunning)})

	if got := o.events(); len(got) != 0 {
		t.Fatalf("Close must drop subscribers: got %d events", len(got))
	}
}
