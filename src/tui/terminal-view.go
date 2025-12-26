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

var specialKeyMap = map[tcell.Key][]byte{
	tcell.KeyEnter:         []byte("\r"),
	tcell.KeyCtrlA:         []byte("\x01"),
	tcell.KeyCtrlB:         []byte("\x02"),
	tcell.KeyCtrlC:         []byte("\x03"),
	tcell.KeyCtrlD:         []byte("\x04"),
	tcell.KeyCtrlE:         []byte("\x05"),
	tcell.KeyCtrlF:         []byte("\x06"),
	tcell.KeyCtrlK:         []byte("\x0B"),
	tcell.KeyCtrlL:         []byte("\x0C"),
	tcell.KeyCtrlN:         []byte("\x0E"),
	tcell.KeyCtrlP:         []byte("\x10"),
	tcell.KeyCtrlQ:         []byte("\x11"),
	tcell.KeyCtrlR:         []byte("\x12"),
	tcell.KeyCtrlS:         []byte("\x13"),
	tcell.KeyCtrlT:         []byte("\x14"),
	tcell.KeyCtrlU:         []byte("\x15"),
	tcell.KeyCtrlV:         []byte("\x16"),
	tcell.KeyCtrlW:         []byte("\x17"),
	tcell.KeyCtrlX:         []byte("\x18"),
	tcell.KeyCtrlY:         []byte("\x19"),
	tcell.KeyCtrlZ:         []byte("\x1A"),
	tcell.KeyCtrlBackslash: []byte("\x1C"), // SIGQUIT
	tcell.KeyCtrlRightSq:   []byte("\x1D"),
	tcell.KeyBackspace:     []byte("\b"),
	tcell.KeyBackspace2:    []byte("\b"),
	tcell.KeyTab:           []byte("\t"),
	tcell.KeyEsc:           []byte("\x1b"),
	tcell.KeyUp:            []byte("\x1b[A"),
	tcell.KeyDown:          []byte("\x1b[B"),
	tcell.KeyRight:         []byte("\x1b[C"),
	tcell.KeyLeft:          []byte("\x1b[D"),
	tcell.KeyPgUp:          []byte("\x1b[5~"),
	tcell.KeyPgDn:          []byte("\x1b[6~"),
	tcell.KeyHome:          []byte("\x1b[H"),
	tcell.KeyEnd:           []byte("\x1b[F"),
	tcell.KeyDelete:        []byte("\x1b[3~"),
	tcell.KeyInsert:        []byte("\x1b[2~"),
}

type TerminalView struct {
	*tview.Box
	app          *tview.Application
	pty          *os.File
	term         *AnsiTerminal
	terminals    map[*os.File]*AnsiTerminal
	lock         sync.Mutex
	isRunning    bool
	width        int
	height       int
	firstDraw    bool // Track if we've had the first draw with proper dimensions
	inEscapeMode bool
	exitKey      tcell.Key
	onEscape     func()
}

func NewTerminalView(app *tview.Application) *TerminalView {
	tv := &TerminalView{
		Box:       tview.NewBox().SetBorder(true),
		app:       app,
		term:      NewAnsiTerminal(80, 24),
		terminals: make(map[*os.File]*AnsiTerminal),
		width:     80,
		height:    24,
		exitKey:   tcell.KeyCtrlA,
	}
	tv.SetTitle("Terminal")
	tv.SetTitleAlign(tview.AlignCenter)
	return tv
}

func (t *TerminalView) SetOnEscape(handler func()) {
	t.onEscape = handler
}

func (t *TerminalView) SetExitKey(key tcell.Key) {
	t.exitKey = key
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

		t.handleKeyInput(event)
	}
}

func (t *TerminalView) handleKeyInput(event *tcell.EventKey) {
	if event.Key() == t.exitKey {
		if !t.inEscapeMode {
			t.inEscapeMode = true
			return
		}
		// If already in escape mode, exitKey means send it literally
		t.inEscapeMode = false
	}

	if t.inEscapeMode {
		if event.Key() == tcell.KeyEsc {
			t.inEscapeMode = false
			if t.onEscape != nil {
				t.onEscape()
			}
			return
		}
		// If key is not Esc and not exitKey (handled above),
		// we treat it as a normal sequence preceded by exitKey.
		// So we first send the buffered exitKey.
		t.inEscapeMode = false

		// Try to send the corresponding control character for the exitKey
		if t.pty != nil {
			if exitBytes := t.getSpecialKeySequence(t.exitKey); len(exitBytes) > 0 {
				_, err := t.pty.Write(exitBytes)
				if err != nil {
					log.Error().Err(err).Msg("Error writing to PTY")
				}
			} else {
				// Fallback if no mapping (though tcell generally maps KeyCtrlX)
				log.Warn().Msgf("No byte sequence found for configured exit key: %v", t.exitKey)
			}
		}
	}

	var data []byte
	if event.Key() == tcell.KeyRune {
		data = []byte(string(event.Rune()))
	} else {
		data = t.getSpecialKeySequence(event.Key())
	}

	if len(data) > 0 && t.pty != nil {
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

func (t *TerminalView) getSpecialKeySequence(key tcell.Key) []byte {
	if seq, ok := specialKeyMap[key]; ok {
		return seq
	}
	return nil
}

func (t *TerminalView) Stop() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.isRunning = false
	// Do NOT clear terminal state here, as we want to persist it in the map
	// t.term = NewAnsiTerminal(t.width, t.height)
	t.pty = nil
}

func (t *TerminalView) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		t.lock.Lock()
		defer t.lock.Unlock()

		if !t.isRunning || t.pty == nil {
			return t.Box.MouseHandler()(action, event, setFocus)
		}

		if !t.InRect(event.Position()) {
			return false, nil
		}

		// Calculate coordinates relative to the terminal view
		x, y, _, _ := t.GetInnerRect()
		mx, my := event.Position()
		relX := mx - x
		relY := my - y

		// Bounds check
		if relX < 0 || relY < 0 || relX >= t.width || relY >= t.height {
			return false, nil
		}

		// Forward mouse event to PTY
		// We use X10 mouse encoding (simple encoding)
		// Format: CSI M <button+32> <x+33> <y+33>
		// Note: coordinates are 1-based in Xterm protocol

		btn := event.Buttons()
		var buttonCode byte

		switch {
		case btn&tcell.Button1 != 0: // Left click
			buttonCode = 0
		case btn&tcell.Button2 != 0: // Middle click
			buttonCode = 1
		case btn&tcell.Button3 != 0: // Right click
			buttonCode = 2
		case btn&tcell.WheelUp != 0:
			buttonCode = 64
		case btn&tcell.WheelDown != 0:
			buttonCode = 65

		default:
			// For movements or unsupported buttons, we consume but don't send (for now)
			// unless we implement full motion tracking
			return true, nil
		}

		// Add modifiers
		if event.Modifiers()&tcell.ModShift != 0 {
			buttonCode |= 4
		}
		if event.Modifiers()&tcell.ModAlt != 0 {
			buttonCode |= 8
		}
		if event.Modifiers()&tcell.ModCtrl != 0 {
			buttonCode |= 16
		}

		// Encode and write
		// <button+32> <x+33> <y+33>
		encoded := []byte{
			'\x1b', '[', 'M',
			buttonCode + 32,
			byte(relX) + 33,
			byte(relY) + 33,
		}

		_, err := t.pty.Write(encoded)
		if err != nil {
			log.Error().Err(err).Msg("Error writing mouse event to PTY")
		}

		// Ensure we get focus on click (especially left click)
		if action == tview.MouseLeftClick || action == tview.MouseLeftDown {
			setFocus(t)
		}

		return true, nil
	}
}
