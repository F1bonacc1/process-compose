package tui

import (
	"testing"

	"github.com/rivo/tview"
)

func TestFilterByText(t *testing.T) {
	cp := &commandPalette{
		filtered: make([]int, 0),
	}

	items := []listItem{
		{main: "Start Process"},
		{main: "Stop Process"},
		{main: "Restart Process"},
		{main: "Scale Process"},
		{main: "Create Process"},
		{main: "Delete Process"},
	}

	tests := []struct {
		name      string
		text      string
		wantCount int
	}{
		{name: "empty filter returns all", text: "", wantCount: 6},
		{name: "case insensitive", text: "RESTART", wantCount: 1},
		{name: "no match", text: "nonexistent", wantCount: 0},
		{name: "partial overlap", text: "re", wantCount: 2}, // Restart, Create
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp.list = newTestList()
			cp.filterByText(tt.text, items)
			if len(cp.filtered) != tt.wantCount {
				t.Errorf("filterByText(%q) filtered count = %d, want %d", tt.text, len(cp.filtered), tt.wantCount)
			}
		})
	}
}

func TestHeightClamping(t *testing.T) {
	tests := []struct {
		name     string
		filtered []int
		phase    palettePhase
		want     int
	}{
		{name: "empty list uses minimum", filtered: []int{}, phase: palettePhaseCommand, want: 8},
		{name: "many items capped at max", filtered: make([]int, 20), phase: palettePhaseCommand, want: 18},
		{name: "process select single row items", filtered: []int{0, 1, 2}, phase: palettePhaseProcessSelect, want: 8},
		{name: "command double row items", filtered: []int{0, 1, 2}, phase: palettePhaseCommand, want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := &commandPalette{filtered: tt.filtered, phase: tt.phase}
			if got := cp.height(); got != tt.want {
				t.Errorf("height() = %d, want %d", got, tt.want)
			}
		})
	}
}

// newTestList creates a minimal tview.List for testing filter logic.
func newTestList() *tview.List {
	return tview.NewList()
}
