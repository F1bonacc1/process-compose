package pclog

import (
	"sync"
)

const (
	slack = 100
)

type ProcessLogBuffer struct {
	buffer    []string
	size      int
	observers map[string]LogObserver
	mx        sync.Mutex
}

func NewLogBuffer(size int) *ProcessLogBuffer {
	return &ProcessLogBuffer{
		size:      size,
		buffer:    make([]string, 0, size+slack),
		observers: map[string]LogObserver{},
	}
}

func (b *ProcessLogBuffer) Write(message string) {
	b.mx.Lock()
	defer b.mx.Unlock()
	b.buffer = append(b.buffer, message)
	if len(b.buffer) > b.size+slack {
		b.buffer = b.buffer[slack:]
	}
	for _, observer := range b.observers {
		_, _ = observer.WriteString(message)
	}

}

func (b *ProcessLogBuffer) GetLogRange(offsetFromEnd, limit int) []string {
	if len(b.buffer) == 0 {
		return []string{}
	}
	if offsetFromEnd < 0 {
		offsetFromEnd = 0
	}
	if offsetFromEnd > len(b.buffer) {
		offsetFromEnd = len(b.buffer)
	}

	if limit < 1 {
		limit = 0
	}
	if limit > len(b.buffer) {
		limit = len(b.buffer)
	}
	if offsetFromEnd+limit > len(b.buffer) {
		limit = len(b.buffer) - offsetFromEnd
	}
	if limit == 0 {
		return b.buffer[len(b.buffer)-offsetFromEnd:]
	}
	return b.buffer[len(b.buffer)-offsetFromEnd : offsetFromEnd+limit]
}

func (b *ProcessLogBuffer) GetLogLength() int {
	return len(b.buffer)
}

func (b *ProcessLogBuffer) GetLogsAndSubscribe(observer LogObserver) {
	b.mx.Lock()
	defer b.mx.Unlock()
	observer.SetLines(b.GetLogRange(observer.GetTailLength(), 0))
	b.observers[observer.GetUniqueID()] = observer
}

func (b *ProcessLogBuffer) Subscribe(observer LogObserver) {
	b.mx.Lock()
	defer b.mx.Unlock()
	b.observers[observer.GetUniqueID()] = observer
}

func (b *ProcessLogBuffer) UnSubscribe(observer LogObserver) {
	b.mx.Lock()
	defer b.mx.Unlock()
	delete(b.observers, observer.GetUniqueID())
}

func (b *ProcessLogBuffer) Close() {
	b.mx.Lock()
	defer b.mx.Unlock()
	b.observers = map[string]LogObserver{}
}
