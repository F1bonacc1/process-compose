package tui

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

type TerminalView struct {
	*tview.Box
	app       *tview.Application
	pty       *os.File
	term      *AnsiTerminal
	terminals map[*os.File]*AnsiTerminal
	lock      sync.Mutex
	isRunning bool
	width     int
	height    int
	firstDraw bool // Track if we've had the first draw with proper dimensions
}

func NewTerminalView(app *tview.Application) *TerminalView {
	tv := &TerminalView{
		Box:       tview.NewBox().SetBorder(true),
		app:       app,
		term:      NewAnsiTerminal(80, 24),
		terminals: make(map[*os.File]*AnsiTerminal),
		width:     80,
		height:    24,
	}
	tv.SetTitle("Terminal")
	return tv
}

func (t *TerminalView) SetPty(ptyFile *os.File) {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Stop any existing PTY reading
	if t.isRunning {
		t.isRunning = false
	}

	t.pty = ptyFile
	if ptyFile != nil {
		// Get current actual dimensions from the Box FIRST
		_, _, width, height := t.GetInnerRect()
		// If dimensions are invalid or too small (likely not yet laid out), use default
		if width < 20 || height < 10 {
			width = 80
			height = 24
		}

		// Force a delayed resize event.
		// We deliberately set the terminal height to (height - 1).
		// This causes the state to mismatch with the actual widget dimensions.
		// The next `Draw` call (which runs on a ticker) will detect this mismatch (h-1 vs h)
		// and trigger a standard resize back to `height`.
		// This ensures that `vim` sees two distinct resize events spaced out in time,
		// guaranteeing a redraw.
		fakeHeight := height
		if height > 1 {
			fakeHeight = height - 1
		}

		// Update stored dimensions to the FAKE height
		t.width = width
		t.height = fakeHeight

		// Check if we already have a terminal for this PTY
		if term, ok := t.terminals[ptyFile]; ok {
			t.term = term
			// Resize existing terminal to FAKE height
			t.term.Resize(width, fakeHeight)
		} else {
			// Create terminal with FAKE dimensions
			t.term = NewAnsiTerminal(width, fakeHeight)
			t.terminals[ptyFile] = t.term
		}

		// Set response callback to write to PTY
		// We use a closure to capture ptyFile
		t.term.SetResponseCallback(func(data []byte) {
			if ptyFile != nil {
				_, err := ptyFile.Write(data)
				if err != nil {
					log.Error().Err(err).Msg("Failed to write response to PTY")
				}
			}
		})

		// Set PTY size to FAKE height
		err := pty.Setsize(ptyFile, &pty.Winsize{
			Rows: uint16(fakeHeight),
			Cols: uint16(width),
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to set initial PTY size")
		}

		// Mark as running, but DON'T start reading yet
		// Wait for first Draw call to ensure proper dimensions
		t.isRunning = true
		t.firstDraw = false
	} else {
		t.isRunning = false
	}
}

func (t *TerminalView) readPty(ptyFile *os.File, term *AnsiTerminal) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Panic in TerminalView.readPty: %v", r)
		}
	}()

	buf := make([]byte, 4096)
	drawTicker := time.NewTicker(16 * time.Millisecond)
	defer drawTicker.Stop()

	for {
		n, err := ptyFile.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Error().Err(err).Msg("Error reading from PTY")
			}
			t.lock.Lock()
			// Only update isRunning if we are still the current PTY
			if t.pty == ptyFile {
				t.isRunning = false
			}
			t.lock.Unlock()
			return
		}

		shouldExit, shouldDraw := func() (bool, bool) {
			t.lock.Lock()
			defer t.lock.Unlock()

			// Check if we are still the active PTY
			if t.pty != ptyFile {
				return true, false
			}

			if n > 0 {
				term.Write(buf[:n])
				return false, true
			}
			// Prevent spin loop on 0-byte reads
			time.Sleep(10 * time.Millisecond)
			return false, false
		}()

		if shouldExit {
			return
		}

		if shouldDraw {
			// Non-blocking draw using ticker
			select {
			case <-drawTicker.C:
				t.app.Draw()
			default:
				// Skip draw if ticker hasn't fired
			}
		}

		// Yield to main thread to allow drawing
		time.Sleep(2 * time.Millisecond)
	}
}

func (t *TerminalView) Draw(screen tcell.Screen) {
	t.Box.Draw(screen)
	x, y, width, height := t.GetInnerRect()

	t.lock.Lock()
	defer t.lock.Unlock()

	// Early exit if dimensions are invalid
	if width <= 0 || height <= 0 {
		return
	}

	// Check if resize is needed BEFORE updating stored dimensions
	if width != t.width || height != t.height {
		t.width = width
		t.height = height
		t.term.Resize(width, height)
		if t.pty != nil {
			err := pty.Setsize(t.pty, &pty.Winsize{
				Rows: uint16(height),
				Cols: uint16(width),
			})
			if err != nil {
				log.Error().Err(err).Msg("Failed to resize PTY")
			}
		}
	}

	// Start reading PTY on first draw (when we have proper dimensions)
	// Only start if we haven't already started AND we're running AND we have a PTY
	if !t.firstDraw && t.isRunning && t.pty != nil {
		t.firstDraw = true
		ptyToRead := t.pty
		termToUse := t.term
		// Release lock before starting goroutine to avoid holding it during startup
		t.lock.Unlock()
		go t.readPty(ptyToRead, termToUse)
		t.lock.Lock()
	}

	if !t.isRunning {
		// Show a message when terminal is not active
		msg := "No active terminal session"
		if width > len(msg) {
			msgX := x + (width-len(msg))/2
			msgY := y + height/2
			for i, ch := range msg {
				screen.SetContent(msgX+i, msgY, ch, nil, tcell.StyleDefault.Dim(true))
			}
		}
		return
	}

	// Iterate over the terminal state and draw cells
	for row := range height {
		for col := range width {
			cell := t.term.GetCell(col, row)
			screen.SetContent(x+col, y+row, cell.Char, nil, cell.Style)
		}
	}

	// Draw cursor
	cursorX, cursorY := t.term.GetCursor()
	if t.term.IsCursorVisible() && cursorY < height && cursorX < width {
		screen.ShowCursor(x+cursorX, y+cursorY)
	}
}

func (t *TerminalView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		t.lock.Lock()
		defer t.lock.Unlock()

		if !t.isRunning || t.pty == nil {
			return
		}

		// Forward input to PTY
		// We might need to convert tcell events to bytes expected by the terminal
		// For now, let's try simple rune forwarding and some keys

		var data []byte
		if event.Key() == tcell.KeyRune {
			data = []byte(string(event.Rune()))
		} else {
			// Handle special keys
			switch event.Key() {
			case tcell.KeyEnter:
				data = []byte("\r")
			case tcell.KeyCtrlA:
				data = []byte("\x01")
			case tcell.KeyCtrlB:
				data = []byte("\x02")
			case tcell.KeyCtrlC:
				data = []byte("\x03")
			case tcell.KeyCtrlD:
				data = []byte("\x04")
			case tcell.KeyCtrlE:
				data = []byte("\x05")
			case tcell.KeyCtrlF:
				data = []byte("\x06")
			case tcell.KeyCtrlK:
				data = []byte("\x0B")
			case tcell.KeyCtrlL:
				data = []byte("\x0C")
			case tcell.KeyCtrlN:
				data = []byte("\x0E")
			case tcell.KeyCtrlP:
				data = []byte("\x10")
			case tcell.KeyCtrlQ: // XON
				data = []byte("\x11")
			case tcell.KeyCtrlR:
				data = []byte("\x12")
			case tcell.KeyCtrlS: // XOFF
				data = []byte("\x13")
			case tcell.KeyCtrlT:
				data = []byte("\x14")
			case tcell.KeyCtrlU:
				data = []byte("\x15")
			case tcell.KeyCtrlV:
				data = []byte("\x16")
			case tcell.KeyCtrlW:
				data = []byte("\x17")
			case tcell.KeyCtrlX:
				data = []byte("\x18")
			case tcell.KeyCtrlY:
				data = []byte("\x19")
			case tcell.KeyCtrlZ:
				data = []byte("\x1A")
			case tcell.KeyCtrlBackslash:
				data = []byte("\x1C") // SIGQUIT
			case tcell.KeyCtrlRightSq:
				data = []byte("\x1D")
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				data = []byte("\b")
			case tcell.KeyTab:
				data = []byte("\t")
			case tcell.KeyEsc:
				data = []byte("\x1b")
			case tcell.KeyUp:
				data = []byte("\x1b[A")
			case tcell.KeyDown:
				data = []byte("\x1b[B")
			case tcell.KeyRight:
				data = []byte("\x1b[C")
			case tcell.KeyLeft:
				data = []byte("\x1b[D")
			case tcell.KeyPgUp:
				data = []byte("\x1b[5~")
			case tcell.KeyPgDn:
				data = []byte("\x1b[6~")
			case tcell.KeyHome:
				data = []byte("\x1b[H")
			case tcell.KeyEnd:
				data = []byte("\x1b[F")
			case tcell.KeyDelete:
				data = []byte("\x1b[3~")
			case tcell.KeyInsert:
				data = []byte("\x1b[2~")
			}
		}

		if len(data) > 0 {
			// Release lock before writing to PTY to avoid deadlock
			// if PTY write blocks
			t.lock.Unlock()
			_, err := t.pty.Write(data)
			if err != nil {
				log.Error().Err(err).Msg("Error writing to PTY")
			}
			// Re-acquire lock to satisfy defer Unlock
			t.lock.Lock()
		}
	}
}

func (t *TerminalView) Stop() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.isRunning = false
	// Do NOT clear terminal state here, as we want to persist it in the map
	// t.term = NewAnsiTerminal(t.width, t.height)
	t.pty = nil
}
