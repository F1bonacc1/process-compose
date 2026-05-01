package app

import (
	"sync"

	"github.com/f1bonacc1/process-compose/src/types"
)

// StateObserver consumes process state events. Implementations must be
// safe for concurrent calls; Notify is invoked while the broadcaster's
// mutex is held, so observers should not block on slow I/O.
type StateObserver interface {
	// Notify is called for every event delivered to this observer. If the
	// observer cannot accept the event (e.g. its buffer is full), it should
	// drop or close itself rather than block.
	Notify(ev types.ProcessStateEvent)
	// UniqueID identifies this observer for subscribe/unsubscribe.
	UniqueID() string
}

// SnapshotFunc returns the current state of all processes for the initial
// replay on subscribe.
type SnapshotFunc func() []types.ProcessState

// ProcessStateBroadcaster fans out ProcessStateEvent values to subscribed
// observers. It mirrors the pclog.ProcessLogBuffer subscribe/fanout pattern.
type ProcessStateBroadcaster struct {
	mx        sync.Mutex
	observers map[string]StateObserver
	snapshot  SnapshotFunc
}

// NewProcessStateBroadcaster constructs a broadcaster. The snapshot func is
// called under the broadcaster's lock during SubscribeWithSnapshot to deliver
// the initial state of all processes; pass nil to disable snapshotting.
func NewProcessStateBroadcaster(snapshot SnapshotFunc) *ProcessStateBroadcaster {
	return &ProcessStateBroadcaster{
		observers: map[string]StateObserver{},
		snapshot:  snapshot,
	}
}

// Subscribe registers an observer for future events only.
func (b *ProcessStateBroadcaster) Subscribe(o StateObserver) {
	b.mx.Lock()
	defer b.mx.Unlock()
	b.observers[o.UniqueID()] = o
}

// SubscribeWithSnapshot delivers a snapshot event for every current process,
// then registers the observer. Both happen under one lock so no live events
// can sneak in between the snapshot and the subscription.
func (b *ProcessStateBroadcaster) SubscribeWithSnapshot(o StateObserver) {
	b.mx.Lock()
	defer b.mx.Unlock()
	if b.snapshot != nil {
		for _, st := range b.snapshot() {
			o.Notify(types.ProcessStateEvent{Snapshot: true, State: st})
		}
	}
	b.observers[o.UniqueID()] = o
}

// Unsubscribe removes an observer. Safe to call multiple times.
func (b *ProcessStateBroadcaster) Unsubscribe(o StateObserver) {
	b.mx.Lock()
	defer b.mx.Unlock()
	delete(b.observers, o.UniqueID())
}

// Publish delivers the event to all current subscribers synchronously.
func (b *ProcessStateBroadcaster) Publish(ev types.ProcessStateEvent) {
	b.mx.Lock()
	defer b.mx.Unlock()
	for _, o := range b.observers {
		o.Notify(ev)
	}
}

// Close drops all subscribers.
func (b *ProcessStateBroadcaster) Close() {
	b.mx.Lock()
	defer b.mx.Unlock()
	b.observers = map[string]StateObserver{}
}
