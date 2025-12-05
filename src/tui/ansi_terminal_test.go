package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestAnsiColors(t *testing.T) {
	term := NewAnsiTerminal(80, 24)

	tests := []struct {
		name     string
		input    string
		expected tcell.Color // Expected foreground color
		isBg     bool
	}{
		{"Basic Red", "\x1b[31mX", tcell.PaletteColor(1), false},
		{"Bright Red", "\x1b[91mX", tcell.PaletteColor(9), false},
		{"256 Color 123", "\x1b[38;5;123mX", tcell.PaletteColor(123), false},
		{"Background Blue", "\x1b[44mX", tcell.PaletteColor(4), true},
		{"Bright Background Blue", "\x1b[104mX", tcell.PaletteColor(12), true},
		{"256 Background 200", "\x1b[48;5;200mX", tcell.PaletteColor(200), true},
		{"RGB Foreground", "\x1b[38;2;255;0;0mX", tcell.NewRGBColor(255, 0, 0), false},
		{"RGB Background", "\x1b[48;2;0;255;0mX", tcell.NewRGBColor(0, 255, 0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset terminal state
			term.currentStyle = tcell.StyleDefault
			term.cursorX = 0
			term.cursorY = 0

			term.Write([]byte(tt.input))
			cell := term.GetCell(0, 0)

			var got tcell.Color
			if tt.isBg {
				_, got, _ = cell.Style.Decompose()
			} else {
				got, _, _ = cell.Style.Decompose()
			}

			if got != tt.expected {
				t.Errorf("expected color %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestUtf8Rendering(t *testing.T) {
	term := NewAnsiTerminal(80, 24)

	// Test string with mixed ASCII and multi-byte UTF-8 characters
	// "Hi 🌍" -> 'H', 'i', ' ', '🌍'
	input := "Hi 🌍"
	term.Write([]byte(input))

	// 'H'
	c0 := term.GetCell(0, 0)
	if c0.Char != 'H' {
		t.Errorf("expected 'H', got %c", c0.Char)
	}

	// 'i'
	c1 := term.GetCell(1, 0)
	if c1.Char != 'i' {
		t.Errorf("expected 'i', got %c", c1.Char)
	}

	// ' '
	c2 := term.GetCell(2, 0)
	if c2.Char != ' ' {
		t.Errorf("expected ' ', got %c", c2.Char)
	}

	// '🌍' (Earth Globe Europe-Africa)
	c3 := term.GetCell(3, 0)
	if c3.Char != '🌍' {
		t.Errorf("expected '🌍', got %c", c3.Char)
	}
}
