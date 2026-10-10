package pclog

import (
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// syncObserver is a LogObserver that is safe for concurrent use.
type syncObserver struct {
	mu       sync.Mutex
	id       string
	tailLen  int
	setLines []string
	lines    []string
	// onCall, if set, runs on every SetLines and WriteString call, outside mu.
	onCall func()
}

func (o *syncObserver) WriteString(line string) (int, error) {
	o.mu.Lock()
	o.lines = append(o.lines, line)
	o.mu.Unlock()
	if o.onCall != nil {
		o.onCall()
	}
	return len(line), nil
}

func (o *syncObserver) SetLines(lines []string) {
	o.mu.Lock()
	o.setLines = append([]string(nil), lines...)
	o.mu.Unlock()
	if o.onCall != nil {
		o.onCall()
	}
}

func (o *syncObserver) GetTailLength() int  { return o.tailLen }
func (o *syncObserver) GetUniqueID() string { return o.id }

// received returns the backlog followed by the live lines.
func (o *syncObserver) received() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append(append([]string(nil), o.setLines...), o.lines...)
}

// waitBriefly waits for ch to close, giving up after a short while. Hooks use
// it where, once fixed, the other goroutine is expected to be blocked on the
// buffer's lock.
func waitBriefly(ch <-chan struct{}) {
	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
	}
}

func waitOrFail(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func assertReceivedOnce(t *testing.T, obs *syncObserver, line string) {
	t.Helper()
	count := 0
	for _, l := range obs.received() {
		if l == line {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected %q exactly once, got it %d times in %v", line, count, obs.received())
	}
}

// A line written after the backlog was read, but before the observer was
// subscribed, must still be delivered.
func TestGetLogsAndSubscribe_WriteAfterBacklogRead(t *testing.T) {
	buffer := NewLogBuffer(10)
	buffer.Write("line 0")
	obs := &syncObserver{id: "obs1", tailLen: 10}
	written := make(chan struct{})
	buffer.hookAfterBacklog = func() {
		go func() {
			buffer.Write("line 1")
			close(written)
		}()
		waitBriefly(written)
	}

	buffer.GetLogsAndSubscribe(obs)
	waitOrFail(t, written, "the write")

	assertReceivedOnce(t, obs, "line 0")
	assertReceivedOnce(t, obs, "line 1")
}

// A line appended before the backlog was read must not also be delivered live.
func TestGetLogsAndSubscribe_SubscribeWhileWriteInProgress(t *testing.T) {
	buffer := NewLogBuffer(10)
	obs := &syncObserver{id: "obs1", tailLen: 10}
	appended := make(chan struct{})
	subscribed := make(chan struct{})
	written := make(chan struct{})
	buffer.hookAfterAppend = func() {
		close(appended)
		waitBriefly(subscribed)
	}

	go func() {
		buffer.Write("line 0")
		close(written)
	}()
	waitOrFail(t, appended, "the append")
	go func() {
		buffer.GetLogsAndSubscribe(obs)
		close(subscribed)
	}()
	waitOrFail(t, written, "the write")
	waitOrFail(t, subscribed, "the subscription")

	assertReceivedOnce(t, obs, "line 0")
}

// Nothing may reach an observer after UnSubscribe returns, even from a Write
// that started before it.
func TestUnSubscribe_NoDeliveryAfterReturn(t *testing.T) {
	buffer := NewLogBuffer(10)
	obs := &syncObserver{id: "obs1", tailLen: 10}
	buffer.GetLogsAndSubscribe(obs)
	subscribersRead := make(chan struct{})
	unsubscribed := make(chan struct{})
	written := make(chan struct{})
	buffer.hookBeforeDeliver = func() {
		close(subscribersRead)
		waitBriefly(unsubscribed)
	}

	go func() {
		buffer.Write("late")
		close(written)
	}()
	waitOrFail(t, subscribersRead, "the write to read its subscribers")
	buffer.UnSubscribe(obs)
	atUnsubscribe := len(obs.received())
	close(unsubscribed)
	waitOrFail(t, written, "the write")

	if got := obs.received(); len(got) != atUnsubscribe {
		t.Fatalf("observer received %v after UnSubscribe returned", got[atUnsubscribe:])
	}
}

// Observers may call back into the buffer, like the TUI log view truncating it
// when a line clears the screen.
func TestGetLogsAndSubscribe_ObserverCallsBackIntoBuffer(t *testing.T) {
	buffer := NewLogBuffer(10)
	buffer.Write("line 0")
	obs := &syncObserver{id: "obs1", tailLen: 10, onCall: buffer.Truncate}
	done := make(chan struct{})

	go func() {
		buffer.GetLogsAndSubscribe(obs)
		buffer.Write("line 1")
		buffer.UnSubscribe(obs)
		close(done)
	}()

	waitOrFail(t, done, "subscribe, write and unsubscribe (deadlock?)")
}

// Every subscriber sees one gapless, duplicate-free sequence across its
// backlog and live lines while a writer keeps appending.
func TestGetLogsAndSubscribe_ConcurrentWriterSequence(t *testing.T) {
	buffer := NewLogBuffer(1000)
	var stop atomic.Bool
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for i := 0; !stop.Load(); i++ {
			buffer.Write(strconv.Itoa(i))
		}
	}()
	defer func() {
		stop.Store(true)
		<-writerDone
	}()

	for i := range 2000 {
		obs := &syncObserver{id: "obs" + strconv.Itoa(i), tailLen: 20}
		buffer.GetLogsAndSubscribe(obs)
		for len(obs.received()) < 25 {
			runtime.Gosched()
		}
		buffer.UnSubscribe(obs)

		got := obs.received()
		for j := 1; j < len(got); j++ {
			prev, _ := strconv.Atoi(got[j-1])
			cur, _ := strconv.Atoi(got[j])
			if cur != prev+1 {
				t.Fatalf("subscriber %d: line %d follows line %d", i, cur, prev)
			}
		}
	}
}
