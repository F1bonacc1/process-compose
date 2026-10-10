package pclog

import (
	"sync"
	"sync/atomic"
	"time"
)

type ProcessLogBuffer struct {
	mxBuf  sync.Mutex
	buffer []string // fixed-size ring buffer
	size   int      // capacity
	head   int      // next write index
	count  int      // items stored (0..size)
	// mxObs guards observers. When both are held, mxBuf is locked first.
	mxObs         sync.Mutex
	observers     map[string]*subscription
	lastWriteNano atomic.Int64 // activity tracking (lock-free)

	// Test hooks, nil outside tests. Each marks a point where a concurrent
	// Write or subscription could interleave.
	hookAfterAppend   func() // Write: line appended, subscribers not yet read
	hookBeforeDeliver func() // Write: subscribers read, line not yet delivered
	hookAfterBacklog  func() // GetLogsAndSubscribe: backlog read, not yet subscribed
}

// subscription delivers lines to one observer. Its mutex serializes the
// observer's SetLines, every delivery to it and its removal, so the observer
// gets its backlog before any live line and nothing once UnSubscribe returns.
// The observer's methods run under that mutex: they must not block or
// unsubscribe, but may call back into the buffer.
type subscription struct {
	mu       sync.Mutex
	observer LogObserver
	active   bool
}

func (s *subscription) deliver(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		_, _ = s.observer.WriteString(message)
	}
}

// deactivate stops deliveries, waiting for one in progress to finish.
func (s *subscription) deactivate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = false
}

const defaultSize = 1000

func NewLogBuffer(size int) *ProcessLogBuffer {
	if size <= 0 {
		size = defaultSize
	}
	return &ProcessLogBuffer{
		size:      size,
		buffer:    make([]string, size),
		observers: map[string]*subscription{},
	}
}

// GetLastWriteTime returns the last time data was written to the log buffer.
// Lock-free: uses atomic load to avoid contention with Write().
func (b *ProcessLogBuffer) GetLastWriteTime() time.Time {
	nano := b.lastWriteNano.Load()
	if nano == 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
}

func (b *ProcessLogBuffer) Write(message string) {
	b.lastWriteNano.Store(time.Now().UnixNano())
	// Append and snapshot the subscriptions under the same mxBuf lock that
	// GetLogsAndSubscribe reads the backlog and subscribes under, so a new
	// subscriber gets this line exactly once: in its backlog or live.
	b.mxBuf.Lock()
	b.buffer[b.head] = message
	b.head = (b.head + 1) % b.size
	if b.count < b.size {
		b.count++
	}
	if b.hookAfterAppend != nil {
		b.hookAfterAppend()
	}
	b.mxObs.Lock()
	snapshot := make([]*subscription, 0, len(b.observers))
	for _, sub := range b.observers {
		snapshot = append(snapshot, sub)
	}
	b.mxObs.Unlock()
	b.mxBuf.Unlock()
	if b.hookBeforeDeliver != nil {
		b.hookBeforeDeliver()
	}

	// Fan out without holding the buffer locks: a slow/blocked observer must
	// not stall every other Write and any new GetLogsAndSubscribe on this buffer.
	for _, sub := range snapshot {
		sub.deliver(message)
	}
}

func (b *ProcessLogBuffer) GetLogRange(endOffset, limit int) []string {
	b.mxBuf.Lock()
	defer b.mxBuf.Unlock()
	return b.getLogRangeLocked(endOffset, limit)
}

func (b *ProcessLogBuffer) getLogRangeLocked(endOffset, limit int) []string {
	if b.count == 0 {
		return []string{}
	}

	if endOffset < 0 {
		endOffset = 0
	}
	if endOffset > b.count {
		endOffset = b.count
	}

	available := b.count - endOffset
	if available <= 0 {
		return []string{}
	}

	if limit <= 0 {
		limit = available
	}
	if limit > available {
		limit = available
	}

	result := make([]string, limit)
	// Start of the logical buffer (oldest element)
	start := (b.head - b.count + b.size) % b.size
	// Skip to the first element we want: offset from end means we skip the last endOffset items,
	// and we want the last `limit` items of the remaining.
	firstIdx := (start + available - limit) % b.size
	for i := range limit {
		result[i] = b.buffer[(firstIdx+i)%b.size]
	}
	return result
}

func (b *ProcessLogBuffer) GetLogLength() int {
	b.mxBuf.Lock()
	defer b.mxBuf.Unlock()
	return b.count
}

// GetLogsAndSubscribe hands the observer the current backlog, then every line
// written after it, each line exactly once.
func (b *ProcessLogBuffer) GetLogsAndSubscribe(observer LogObserver) {
	sub := &subscription{observer: observer}
	// Hold live lines back until the observer has its backlog.
	sub.mu.Lock()
	defer sub.mu.Unlock()

	b.mxBuf.Lock()
	lines := b.getLogRangeLocked(0, observer.GetTailLength())
	if b.hookAfterBacklog != nil {
		b.hookAfterBacklog()
	}
	replaced := b.register(sub)
	b.mxBuf.Unlock()
	if replaced != nil {
		replaced.deactivate()
	}

	// Outside mxBuf: the observer may call back into the buffer, e.g. the TUI
	// log view truncates it when a line clears the screen.
	observer.SetLines(lines)
	sub.active = true
}

func (b *ProcessLogBuffer) Subscribe(observer LogObserver) {
	if replaced := b.register(&subscription{observer: observer, active: true}); replaced != nil {
		replaced.deactivate()
	}
}

// register adds sub, returning the subscription of the same observer it
// replaced, if any.
func (b *ProcessLogBuffer) register(sub *subscription) *subscription {
	b.mxObs.Lock()
	defer b.mxObs.Unlock()
	id := sub.observer.GetUniqueID()
	replaced := b.observers[id]
	b.observers[id] = sub
	return replaced
}

// UnSubscribe stops deliveries to the observer. Once it returns, the observer
// receives nothing more, even from a Write already in progress.
func (b *ProcessLogBuffer) UnSubscribe(observer LogObserver) {
	b.mxObs.Lock()
	id := observer.GetUniqueID()
	sub := b.observers[id]
	delete(b.observers, id)
	b.mxObs.Unlock()
	if sub != nil {
		sub.deactivate()
	}
}

func (b *ProcessLogBuffer) Close() {
	b.mxObs.Lock()
	subs := b.observers
	b.observers = map[string]*subscription{}
	b.mxObs.Unlock()
	for _, sub := range subs {
		sub.deactivate()
	}
}

func (b *ProcessLogBuffer) Truncate() {
	b.mxBuf.Lock()
	defer b.mxBuf.Unlock()
	b.head = 0
	b.count = 0
}
