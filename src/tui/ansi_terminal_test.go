package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// TestDeferredTerminalReplaysAtActivatedWidth is a unit-level regression test
// for issue #512. A deferred terminal must not parse output at its 80-column
// placeholder; it must buffer the raw bytes and replay them only once Activate
// supplies the real width, so wrapping reflects the actual pane.
func TestDeferredTerminalReplaysAtActivatedWidth(t *testing.T) {
	const line = 120 // longer than the 80-column placeholder, shorter than 200

	// Wide activation: a 120-char line fits on one row, so it must NOT wrap.
	wide := NewDeferredAnsiTerminal()
	wide.Write([]byte(strings.Repeat("X", line) + "\n"))
	// Nothing is parsed while deferred.
	if c := wide.GetCell(0, 0).Char; c != ' ' {
		t.Fatalf("deferred terminal parsed output before Activate: cell(0,0)=%q", c)
	}
	wide.Activate(200, 24)
	for x := range line {
		if c := wide.GetCell(x, 0).Char; c != 'X' {
			t.Fatalf("at 200 cols col %d of row 0 = %q, want 'X' (line wrapped though it fit)", x, c)
		}
	}
	if c := wide.GetCell(0, 1).Char; c != ' ' {
		t.Fatalf("at 200 cols row 1 is not empty (line wrapped though it fit): %q", c)
	}

	// Narrow activation: the same line must wrap at the activated width (100),
	// not at the 80-column placeholder.
	narrow := NewDeferredAnsiTerminal()
	narrow.Write([]byte(strings.Repeat("X", line) + "\n"))
	narrow.Activate(100, 24)
	if c := narrow.GetCell(99, 0).Char; c != 'X' {
		t.Fatalf("at 100 cols col 99 of row 0 = %q, want 'X' (wrapped before the real width)", c)
	}
	if c := narrow.GetCell(line-100-1, 1).Char; c != 'X' {
		t.Fatalf("at 100 cols the wrapped remainder is missing on row 1")
	}
}

// TestCSIIntermediateBytesNotPrinted guards against a CSI parser bug where
// intermediate bytes (0x20-0x2F) were misclassified as the final byte, ending
// the sequence early and printing its trailing final byte as a literal
// character. The concrete symptom was helix's DECSCUSR cursor-shape sequence
// (ESC [ <n> SP q) leaking a stray 'q' onto the screen on every ':'.
func TestCSIIntermediateBytesNotPrinted(t *testing.T) {
	cases := []struct {
		name string
		seq  string
	}{
		{"DECSCUSR steady bar", "\x1b[6 q"},
		{"DECSCUSR blinking block", "\x1b[1 q"},
		{"DECSCUSR default", "\x1b[ q"},
		{"DECSTR soft reset", "\x1b[!p"},
		{"DECSCA", "\x1b[\"q"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			term := NewAnsiTerminal(80, 24)
			term.Write([]byte(tc.seq))
			// The sequence must be fully consumed: nothing printed and the
			// parser back in the normal state so following text lands at (0,0).
			if c := term.GetCell(0, 0).Char; c != ' ' {
				t.Fatalf("%s leaked %q onto the screen at (0,0)", tc.name, c)
			}
			term.Write([]byte("Z"))
			if c := term.GetCell(0, 0).Char; c != 'Z' {
				t.Fatalf("%s left the parser in a bad state: text landed as %q, want 'Z'", tc.name, c)
			}
		})
	}
}

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

// TestWideUtf8Rendering verifies the CJK output in the interactive-process
// fixture for issue #535. These runes occupy two terminal cells; storing each
// one in a single cell lets the following rune overwrite its right half during
// rendering.
func TestWideUtf8Rendering(t *testing.T) {
	const input = "中文测试：系统连接正常，今日2026年8月27日" //nolint:gosmopolitan
	term := NewAnsiTerminal(80, 24)

	// PTY reads can split a multi-byte UTF-8 rune at arbitrary byte boundaries.
	bytes := []byte(input)
	for _, split := range []int{1, 8, 19, len(bytes)} {
		if len(bytes) == 0 {
			break
		}
		if split > len(bytes) {
			split = len(bytes)
		}
		term.Write(bytes[:split])
		bytes = bytes[split:]
	}

	col := 0
	for _, r := range input {
		if got := term.GetCell(col, 0); got.Char != r || got.Continuation {
			t.Fatalf("cell %d = %#v, want leading cell for %q", col, got, r)
		}
		width := uniseg.StringWidth(string(r))
		for continuation := 1; continuation < width; continuation++ {
			if got := term.GetCell(col+continuation, 0); !got.Continuation {
				t.Fatalf("cell %d is not marked as the continuation of %q", col+continuation, r)
			}
		}
		col += width
	}

	if got := term.GetText(0, 0, col-1, 0); got != input {
		t.Fatalf("copied terminal text = %q, want %q", got, input)
	}
	if x, y := term.GetCursor(); x != col || y != 0 {
		t.Fatalf("cursor = (%d, %d), want (%d, 0)", x, y, col)
	}
}

// TestWideRuneWrapsBeforeFinalColumn ensures that a wide rune that cannot fit in the
// remaining column wraps to the next line instead of overflowing or splitting.
func TestWideRuneWrapsBeforeFinalColumn(t *testing.T) {
	term := NewAnsiTerminal(5, 2)
	term.Write([]byte("abcd中")) //nolint:gosmopolitan

	const want = "abcd\n中" //nolint:gosmopolitan
	if got := term.GetText(0, 0, 4, 1); got != want {
		t.Fatalf("terminal text = %q, want %q", got, want)
	}
	if x, y := term.GetCursor(); x != 2 || y != 1 {
		t.Fatalf("cursor = (%d, %d), want (2, 1)", x, y)
	}
}

// TestWideRuneOverwrite verifies that overwriting parts of a wide rune cleanly
// resets orphaned continuation cells and orphaned leading halves.
func TestWideRuneOverwrite(t *testing.T) {
	t.Run("Overwrite leading cell with narrow character clears continuation", func(t *testing.T) {
		term := NewAnsiTerminal(10, 2)
		// Write a wide character followed by another character: "中B"
		term.Write([]byte("中B")) //nolint:gosmopolitan
		if cell := term.GetCell(1, 0); !cell.Continuation {
			t.Fatalf("cell 1 should be continuation cell")
		}

		// Move cursor back to col 0 and overwrite with 'A'
		term.Write([]byte("\rA"))

		// Col 0 should be 'A', Col 1 should now be a blank space with Continuation=false
		if got := term.GetCell(0, 0); got.Char != 'A' || got.Continuation {
			t.Fatalf("cell 0 = %#v, want 'A' without continuation", got)
		}
		if got := term.GetCell(1, 0); got.Char != ' ' || got.Continuation {
			t.Fatalf("cell 1 = %#v, want ' ' without continuation", got)
		}
		if got := term.GetCell(2, 0); got.Char != 'B' || got.Continuation {
			t.Fatalf("cell 2 = %#v, want 'B' without continuation", got)
		}

		// GetText should preserve the space between A and B
		if got := term.GetText(0, 0, 2, 0); got != "A B" {
			t.Fatalf("copied terminal text = %q, want %q", got, "A B")
		}
	})

	t.Run("Overwrite continuation cell with narrow character clears leading cell", func(t *testing.T) {
		term := NewAnsiTerminal(10, 2)
		// Write a wide character: "中" at col 0, continuation at col 1
		term.Write([]byte("中")) //nolint:gosmopolitan

		// Backspace once to land on col 1 (the continuation cell) and write 'X'
		term.Write([]byte("\bX"))

		// Col 0 should have been cleared to ' ', Col 1 should be 'X'
		if got := term.GetCell(0, 0); got.Char != ' ' || got.Continuation {
			t.Fatalf("cell 0 = %#v, want ' ' without continuation", got)
		}
		if got := term.GetCell(1, 0); got.Char != 'X' || got.Continuation {
			t.Fatalf("cell 1 = %#v, want 'X' without continuation", got)
		}
		if got := term.GetText(0, 0, 1, 0); got != " X" {
			t.Fatalf("copied terminal text = %q, want %q", got, " X")
		}
	})
}

