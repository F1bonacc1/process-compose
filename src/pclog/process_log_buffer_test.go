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
