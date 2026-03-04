package pclog

import (
	"sync"
)

type ProcessLogBuffer struct {
	mxBuf     sync.Mutex
	buffer    []string // fixed-size ring buffer
	size      int      // capacity
	head      int      // next write index
	count     int      // items stored (0..size)
	mxObs     sync.Mutex
	observers map[string]LogObserver
}

const defaultSize = 1000

func NewLogBuffer(size int) *ProcessLogBuffer {
	if size <= 0 {
		size = defaultSize
	}
	return &ProcessLogBuffer{
		size:      size,
		buffer:    make([]string, size),
		observers: map[string]LogObserver{},
	}
}

func (b *ProcessLogBuffer) Write(message string) {
	b.mxBuf.Lock()
	b.buffer[b.head] = message
	b.head = (b.head + 1) % b.size
	if b.count < b.size {
		b.count++
	}
	b.mxBuf.Unlock()

	b.mxObs.Lock()
	defer b.mxObs.Unlock()
	for _, observer := range b.observers {
		_, _ = observer.WriteString(message)
	}
}

func (b *ProcessLogBuffer) GetLogRange(endOffset, limit int) []string {
	b.mxBuf.Lock()
	defer b.mxBuf.Unlock()

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

func (b *ProcessLogBuffer) GetLogsAndSubscribe(observer LogObserver) {
	lines := b.GetLogRange(0, observer.GetTailLength())
	b.mxObs.Lock()
	defer b.mxObs.Unlock()
	observer.SetLines(lines)
	b.observers[observer.GetUniqueID()] = observer
}

func (b *ProcessLogBuffer) Subscribe(observer LogObserver) {
	b.mxObs.Lock()
	defer b.mxObs.Unlock()
	b.observers[observer.GetUniqueID()] = observer
}

func (b *ProcessLogBuffer) UnSubscribe(observer LogObserver) {
	b.mxObs.Lock()
	defer b.mxObs.Unlock()
	delete(b.observers, observer.GetUniqueID())
}

func (b *ProcessLogBuffer) Close() {
	b.mxObs.Lock()
	defer b.mxObs.Unlock()
	b.observers = map[string]LogObserver{}
}

func (b *ProcessLogBuffer) Truncate() {
	b.mxBuf.Lock()
	defer b.mxBuf.Unlock()
	b.head = 0
	b.count = 0
}
