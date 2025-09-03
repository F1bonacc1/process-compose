package pclog

import (
	"sync"
)

const (
	slack = 100
)

type ProcessLogBuffer struct {
	mxBuf     sync.Mutex
	buffer    []string
	size      int
	mxObs     sync.Mutex
	observers map[string]LogObserver
}

func NewLogBuffer(size int) *ProcessLogBuffer {
	return &ProcessLogBuffer{
		size:      size,
		buffer:    make([]string, 0, size+slack),
		observers: map[string]LogObserver{},
	}
}

func (b *ProcessLogBuffer) Write(message string) {
	b.mxBuf.Lock()
	b.buffer = append(b.buffer, message)
	if len(b.buffer) > b.size+slack {
		b.buffer = b.buffer[slack:]
	}
	b.mxBuf.Unlock()
	b.mxObs.Lock()
	defer b.mxObs.Unlock()
	for _, observer := range b.observers {
		_, _ = observer.WriteString(message)
	}

}

func (b *ProcessLogBuffer) GetLogRange(offsetFromEnd, limit int) []string {
	b.mxBuf.Lock()
	defer b.mxBuf.Unlock()
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
	b.mxObs.Lock()
	defer b.mxObs.Unlock()
	observer.SetLines(b.GetLogRange(observer.GetTailLength(), 0))
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
	b.buffer = b.buffer[:0]
}
