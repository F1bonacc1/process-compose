package pclog

import (
	"fmt"
	"testing"
)

func createLogBufferLines(start int, count int) []string {
	lines := make([]string, count)
	for i := range count {
		lines[i] = fmt.Sprintf("line %d", start+i)
	}
	return lines
}

type mockObserver struct {
	id       string
	tailLen  int
	lines    []string
	setLines []string
}

func (m *mockObserver) WriteString(line string) (int, error) {
	m.lines = append(m.lines, line)
	return len(line), nil
}
func (m *mockObserver) SetLines(lines []string) { m.setLines = lines }
func (m *mockObserver) GetTailLength() int      { return m.tailLen }
func (m *mockObserver) GetUniqueID() string     { return m.id }

func TestGetLogRange(t *testing.T) {
	buffer := NewLogBuffer(100)
	for i := range 20 {
		buffer.Write(fmt.Sprintf("line %d", i))
	}

	tests := []struct {
		name          string
		endOffset     int
		limit         int
		expectedLines []string
	}{
		{
			name:          "get last 10 lines",
			endOffset:     0,
			limit:         10,
			expectedLines: createLogBufferLines(10, 10),
		},
		{
			name:          "get last 10 lines with offset",
			endOffset:     5,
			limit:         10,
			expectedLines: createLogBufferLines(5, 10),
		},
		{
			name:          "limit greater than available lines",
			endOffset:     0,
			limit:         30,
			expectedLines: createLogBufferLines(0, 20),
		},
		{
			name:          "offset greater than available lines",
			endOffset:     30,
			limit:         10,
			expectedLines: []string{},
		},
		{
			name:          "zero limit",
			endOffset:     0,
			limit:         0,
			expectedLines: createLogBufferLines(0, 20),
		},
		{
			name:          "negative offset",
			endOffset:     -5,
			limit:         10,
			expectedLines: createLogBufferLines(10, 10),
		},
		{
			name:          "negative limit",
			endOffset:     0,
			limit:         -1,
			expectedLines: createLogBufferLines(0, 20),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := buffer.GetLogRange(tt.endOffset, tt.limit)
			if len(logs) != len(tt.expectedLines) {
				t.Fatalf("expected %d lines, got %d", len(tt.expectedLines), len(logs))
			}
			for i, line := range logs {
				if line != tt.expectedLines[i] {
					t.Errorf("expected line %d to be '%s', got '%s'", i, tt.expectedLines[i], line)
				}
			}
		})
	}
}

func TestWriteOverflow(t *testing.T) {
	size := 10
	buffer := NewLogBuffer(size)

	// Write more than size lines to trigger overflow
	totalLines := size*2 + 1
	for i := range totalLines {
		buffer.Write(fmt.Sprintf("line %d", i))
	}

	length := buffer.GetLogLength()
	if length != size {
		t.Fatalf("expected length %d after overflow, got %d", size, length)
	}

	logs := buffer.GetLogRange(0, 0)
	expected := createLogBufferLines(totalLines-size, size)
	if len(logs) != len(expected) {
		t.Fatalf("expected %d lines, got %d", len(expected), len(logs))
	}
	for i, line := range logs {
		if line != expected[i] {
			t.Errorf("line %d: expected '%s', got '%s'", i, expected[i], line)
		}
	}
}

func TestWriteExactCapacity(t *testing.T) {
	size := 10
	buffer := NewLogBuffer(size)

	for i := range size {
		buffer.Write(fmt.Sprintf("line %d", i))
	}

	length := buffer.GetLogLength()
	if length != size {
		t.Fatalf("expected length %d, got %d", size, length)
	}

	logs := buffer.GetLogRange(0, 0)
	expected := createLogBufferLines(0, size)
	if len(logs) != len(expected) {
		t.Fatalf("expected %d lines, got %d", len(expected), len(logs))
	}
	for i, line := range logs {
		if line != expected[i] {
			t.Errorf("line %d: expected '%s', got '%s'", i, expected[i], line)
		}
	}
}

func TestGetLogLength(t *testing.T) {
	buffer := NewLogBuffer(10)

	if l := buffer.GetLogLength(); l != 0 {
		t.Fatalf("empty buffer: expected 0, got %d", l)
	}

	for i := range 5 {
		buffer.Write(fmt.Sprintf("line %d", i))
	}
	if l := buffer.GetLogLength(); l != 5 {
		t.Fatalf("after 5 writes: expected 5, got %d", l)
	}

	// Overflow: write enough to exceed capacity
	for i := 5; i < 25; i++ {
		buffer.Write(fmt.Sprintf("line %d", i))
	}
	if l := buffer.GetLogLength(); l != 10 {
		t.Fatalf("after overflow: expected 10, got %d", l)
	}
}

func TestTruncate(t *testing.T) {
	buffer := NewLogBuffer(10)
	for i := range 5 {
		buffer.Write(fmt.Sprintf("line %d", i))
	}

	buffer.Truncate()

	if l := buffer.GetLogLength(); l != 0 {
		t.Fatalf("after truncate: expected 0, got %d", l)
	}
	logs := buffer.GetLogRange(0, 0)
	if len(logs) != 0 {
		t.Fatalf("after truncate: expected empty, got %d lines", len(logs))
	}

	// Write after truncate should work
	buffer.Write("after truncate")
	if l := buffer.GetLogLength(); l != 1 {
		t.Fatalf("after write post-truncate: expected 1, got %d", l)
	}
	logs = buffer.GetLogRange(0, 0)
	if len(logs) != 1 || logs[0] != "after truncate" {
		t.Fatalf("unexpected logs after truncate+write: %v", logs)
	}
}

func TestGetLogRangeAfterOverflow(t *testing.T) {
	size := 10
	buffer := NewLogBuffer(size)

	// Write enough to overflow
	total := size*2 + 5
	for i := range total {
		buffer.Write(fmt.Sprintf("line %d", i))
	}

	tests := []struct {
		name          string
		endOffset     int
		limit         int
		expectedLines []string
	}{
		{
			name:          "all lines after overflow",
			endOffset:     0,
			limit:         0,
			expectedLines: createLogBufferLines(total-size, size),
		},
		{
			name:          "last 5 lines after overflow",
			endOffset:     0,
			limit:         5,
			expectedLines: createLogBufferLines(total-5, 5),
		},
		{
			name:          "5 lines with offset 3 after overflow",
			endOffset:     3,
			limit:         5,
			expectedLines: createLogBufferLines(total-size+2, 5),
		},
		{
			name:          "limit exceeds available after overflow",
			endOffset:     0,
			limit:         20,
			expectedLines: createLogBufferLines(total-size, size),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := buffer.GetLogRange(tt.endOffset, tt.limit)
			if len(logs) != len(tt.expectedLines) {
				t.Fatalf("expected %d lines, got %d", len(tt.expectedLines), len(logs))
			}
			for i, line := range logs {
				if line != tt.expectedLines[i] {
					t.Errorf("line %d: expected '%s', got '%s'", i, tt.expectedLines[i], line)
				}
			}
		})
	}
}

func TestSubscribeAndUnSubscribe(t *testing.T) {
	buffer := NewLogBuffer(10)
	obs := &mockObserver{id: "obs1", tailLen: 10}

	buffer.Subscribe(obs)
	buffer.Write("hello")

	if len(obs.lines) != 1 || obs.lines[0] != "hello" {
		t.Fatalf("observer should have received 'hello', got %v", obs.lines)
	}

	buffer.UnSubscribe(obs)
	buffer.Write("world")

	if len(obs.lines) != 1 {
		t.Fatalf("observer should not receive after unsubscribe, got %v", obs.lines)
	}
}

func TestGetLogsAndSubscribe(t *testing.T) {
	buffer := NewLogBuffer(10)
	for i := range 5 {
		buffer.Write(fmt.Sprintf("line %d", i))
	}

	obs := &mockObserver{id: "obs1", tailLen: 3}
	buffer.GetLogsAndSubscribe(obs)

	// Should have received last 3 lines via SetLines
	if len(obs.setLines) != 3 {
		t.Fatalf("expected 3 setLines, got %d: %v", len(obs.setLines), obs.setLines)
	}
	expected := createLogBufferLines(2, 3)
	for i, line := range obs.setLines {
		if line != expected[i] {
			t.Errorf("setLines[%d]: expected '%s', got '%s'", i, expected[i], line)
		}
	}

	// Subsequent writes should go to observer
	buffer.Write("new line")
	if len(obs.lines) != 1 || obs.lines[0] != "new line" {
		t.Fatalf("observer should receive subsequent writes, got %v", obs.lines)
	}
}

func TestClose(t *testing.T) {
	buffer := NewLogBuffer(10)
	obs := &mockObserver{id: "obs1", tailLen: 10}
	buffer.Subscribe(obs)

	buffer.Close()
	buffer.Write("after close")

	if len(obs.lines) != 0 {
		t.Fatalf("observer should not receive after Close, got %v", obs.lines)
	}
}

// Benchmarks

func BenchmarkWrite(b *testing.B) {
	buffer := NewLogBuffer(1000)
	line := "benchmark log line content here"
	b.ResetTimer()
	for i := range b.N {
		_ = i
		buffer.Write(line)
	}
}

func BenchmarkWriteOverflow(b *testing.B) {
	buffer := NewLogBuffer(1000)
	line := "benchmark log line content here"
	// Pre-fill to capacity
	for range 1000 {
		buffer.Write(line)
	}
	b.ResetTimer()
	for i := range b.N {
		_ = i
		buffer.Write(line)
	}
}

func BenchmarkGetLogRange(b *testing.B) {
	buffer := NewLogBuffer(1000)
	for range 1000 {
		buffer.Write("benchmark log line content here")
	}
	b.ResetTimer()
	for i := range b.N {
		_ = i
		buffer.GetLogRange(0, 100)
	}
}
