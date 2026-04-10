package tui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/f1bonacc1/glippy"
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
	activeReaders map[*os.File]bool // tracks PTYs with running readPty goroutines
	lock         sync.Mutex
	isRunning    bool
	width        int
	height       int
	firstDraw    bool // Track if we've had the first draw with proper dimensions
	inEscapeMode bool
	exitKey      tcell.Key
	onEscape     func()
	onFocus      func()
	onBlur       func()
	isScrolling  bool

	// Selection state
	isSelecting  bool
	hasSelection bool
	selStartCol  int
	selStartRow  int
	selEndCol    int
	selEndRow    int
	onSelectionChanged func()
}

func NewTerminalView(app *tview.Application) *TerminalView {
	tv := &TerminalView{
		Box:           tview.NewBox().SetBorder(true),
		app:           app,
		term:          NewAnsiTerminal(80, 24),
		terminals:     make(map[*os.File]*AnsiTerminal),
		activeReaders: make(map[*os.File]bool),
		width:         80,
		height:        24,
		exitKey:       tcell.KeyCtrlA,
	}
	tv.SetTitle("Terminal")
	tv.SetTitleAlign(tview.AlignCenter)
	return tv
}

func (t *TerminalView) SetOnEscape(handler func()) {
	t.onEscape = handler
}

func (t *TerminalView) SetOnFocus(handler func()) {
	t.onFocus = handler
}

func (t *TerminalView) SetOnBlur(handler func()) {
	t.onBlur = handler
}

func (t *TerminalView) Focus(delegate func(p tview.Primitive)) {
	t.Box.Focus(delegate)
	if t.onFocus != nil {
		t.onFocus()
	}
}

func (t *TerminalView) Blur() {
	t.Box.Blur()
	if t.onBlur != nil {
		t.onBlur()
	}
}

func (t *TerminalView) SetOnSelectionChanged(handler func()) {
	t.onSelectionChanged = handler
}

func (t *TerminalView) HasSelection() bool {
	return t.hasSelection
}

func (t *TerminalView) clearSelection() {
	if t.hasSelection || t.isSelecting {
		t.hasSelection = false
		t.isSelecting = false
		if t.onSelectionChanged != nil {
			t.onSelectionChanged()
		}
	}
}

func (t *TerminalView) isInSelection(col, row int) bool {
	if !t.hasSelection && !t.isSelecting {
		return false
	}
	startCol, startRow := t.selStartCol, t.selStartRow
	endCol, endRow := t.selEndCol, t.selEndRow
	// Normalize direction
	startPos := startRow*t.width + startCol
	endPos := endRow*t.width + endCol
	if startPos > endPos {
		startPos, endPos = endPos, startPos
	}
	pos := row*t.width + col
	return pos >= startPos && pos <= endPos
}

func (t *TerminalView) copySelection() {
	if !t.hasSelection || t.term == nil {
		return
	}
	text := t.term.GetText(t.selStartCol, t.selStartRow, t.selEndCol, t.selEndRow)
	if err := glippy.Set(text); err != nil {
		log.Error().Err(err).Msg("Failed to copy to clipboard")
	}
	t.clearSelection()
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
			delete(t.activeReaders, ptyFile)
			t.lock.Unlock()
			return
		}

		shouldExit, shouldDraw := func() (bool, bool) {
			t.lock.Lock()
			defer t.lock.Unlock()

			if n > 0 {
				term.Write(buf[:n])
				// Only trigger draw if this is the active PTY
				return false, t.pty == ptyFile
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
	// Start reading PTY on first draw (when we have proper dimensions).
	// Only start if no reader is already running for this PTY.
	if !t.firstDraw && t.isRunning && t.pty != nil {
		t.firstDraw = true
		ptyToRead := t.pty
		termToUse := t.term
		if !t.activeReaders[ptyToRead] {
			t.activeReaders[ptyToRead] = true
			// Release lock before starting goroutine to avoid holding it during startup
			t.lock.Unlock()
			go t.readPty(ptyToRead, termToUse)
			t.lock.Lock()
		}
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
			style := cell.Style
			if t.isInSelection(col, row) {
				style = style.Reverse(true)
			}
			screen.SetContent(x+col, y+row, cell.Char, nil, style)
		}
	}

	// Draw cursor
	cursorX, cursorY := t.term.GetCursor()
	if t.term.IsCursorVisible() && cursorY < height && cursorX < width && t.term.viewOffset == 0 {
		screen.ShowCursor(x+cursorX, y+cursorY)
	}

	// Draw scrolling indicator
	if t.term.viewOffset > 0 {
		viewOffset, historySize := t.term.GetViewportStatus()
		msg := fmt.Sprintf("[%d/%d]", viewOffset, historySize)
		// Draw at top right
		msgX := x + width - len(msg) - 1
		msgY := y
		if msgX > x {
			for i, ch := range msg {
				screen.SetContent(msgX+i, msgY, ch, nil, tcell.StyleDefault.Reverse(true))
			}
		}
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
	// Handle selection: Enter copies, any other key clears
	if t.hasSelection {
		if event.Key() == tcell.KeyEnter {
			t.copySelection()
			return
		}
		t.clearSelection()
	}

	if event.Key() == t.exitKey {
		if !t.inEscapeMode {
			t.inEscapeMode = true
			return
		}
		// If already in escape mode, exitKey means send it literally
		t.inEscapeMode = false
	}

	if t.inEscapeMode {
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyHome, tcell.KeyEnd:
			t.inEscapeMode = false
			t.isScrolling = true
			// Fallthrough to scrolling handler
		case tcell.KeyEsc:
			t.inEscapeMode = false
			if t.onEscape != nil {
				t.onEscape()
			}
			return
		default:
			// Not a scrolling key, treat as normal sequence preceded by exitKey
			t.inEscapeMode = false

			// Try to send the corresponding control character for the exitKey
			if t.pty != nil {
				if exitBytes := t.getSpecialKeySequence(t.exitKey); len(exitBytes) > 0 {
					_, err := t.pty.Write(exitBytes)
					if err != nil {
						log.Error().Err(err).Msg("Error writing to PTY")
					}
				} else {
					log.Warn().Msgf("No byte sequence found for configured exit key: %v", t.exitKey)
				}
			}
		}
	}

	if t.isScrolling {
		switch event.Key() {
		case tcell.KeyEsc:
			t.isScrolling = false
			t.term.ResetViewport()
			return
		case tcell.KeyUp:
			t.term.ScrollViewport(1)
			return
		case tcell.KeyDown:
			t.term.ScrollViewport(-1)
			return
		case tcell.KeyPgUp:
			t.term.ScrollViewport(t.height)
			return
		case tcell.KeyPgDn:
			t.term.ScrollViewport(-t.height)
			return
		case tcell.KeyHome:
			t.term.ScrollViewport(1000000) // All the way up
			return
		case tcell.KeyEnd:
			t.term.ResetViewport()
			return
		default:
			// Any other key exits scrolling mode and is processed normally
			t.isScrolling = false
			t.term.ResetViewport()
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

// Application cursor key mode sequences (DECCKM / SS3)
var applicationKeyMap = map[tcell.Key][]byte{
	tcell.KeyUp:    []byte("\x1bOA"),
	tcell.KeyDown:  []byte("\x1bOB"),
	tcell.KeyRight: []byte("\x1bOC"),
	tcell.KeyLeft:  []byte("\x1bOD"),
	tcell.KeyHome:  []byte("\x1bOH"),
	tcell.KeyEnd:   []byte("\x1bOF"),
}

func (t *TerminalView) getSpecialKeySequence(key tcell.Key) []byte {
	if t.term != nil && t.term.IsApplicationCursorKeys() {
		if seq, ok := applicationKeyMap[key]; ok {
			return seq
		}
	}
	if seq, ok := specialKeyMap[key]; ok {
		return seq
	}
	return nil
}

// GetLastActivityTime returns the last write time for a terminal associated with the given PTY.
// Returns zero time if the PTY has no associated terminal.
func (t *TerminalView) GetLastActivityTime(ptyFile *os.File) time.Time {
	if ptyFile == nil {
		return time.Time{}
	}
	t.lock.Lock()
	term, ok := t.terminals[ptyFile]
	t.lock.Unlock()
	if !ok {
		return time.Time{}
	}
	return term.GetLastWriteTime()
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

		// When PTY mouse mode is OFF, handle text selection
		if !t.term.IsMouseModeEnabled() {
			return t.handleSelectionMouse(action, relX, relY, setFocus)
		}

		// PTY mouse mode is ON: forward to PTY
		return t.handlePtyMouse(action, event, relX, relY, setFocus)
	}
}

func (t *TerminalView) handleSelectionMouse(action tview.MouseAction, relX, relY int, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	switch action {
	case tview.MouseLeftDown:
		t.clearSelection()
		t.isSelecting = true
		t.selStartCol = relX
		t.selStartRow = relY
		t.selEndCol = relX
		t.selEndRow = relY
		setFocus(t)
		return true, t // Capture subsequent mouse events
	case tview.MouseMove:
		if t.isSelecting {
			t.selEndCol = relX
			t.selEndRow = relY
			return true, t // Keep capturing
		}
	case tview.MouseLeftUp:
		if t.isSelecting {
			t.selEndCol = relX
			t.selEndRow = relY
			t.isSelecting = false
			// Only mark as having selection if start != end
			if t.selStartCol != t.selEndCol || t.selStartRow != t.selEndRow {
				t.hasSelection = true
				if t.onSelectionChanged != nil {
					t.onSelectionChanged()
				}
			}
			return true, nil // Release capture
		}
	case tview.MouseLeftClick:
		t.clearSelection()
		setFocus(t)
		return true, nil
	case tview.MouseScrollUp:
		t.isScrolling = true
		t.term.ScrollViewport(1)
		return true, nil
	case tview.MouseScrollDown:
		t.isScrolling = true
		t.term.ScrollViewport(-1)
		return true, nil
	}
	return true, nil
}

func (t *TerminalView) handlePtyMouse(action tview.MouseAction, event *tcell.EventMouse, relX, relY int, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	btn := event.Buttons()
	var buttonCode byte

	switch {
	case btn&tcell.Button1 != 0:
		buttonCode = 0
	case btn&tcell.Button2 != 0:
		buttonCode = 1
	case btn&tcell.Button3 != 0:
		buttonCode = 2
	case btn&tcell.WheelUp != 0:
		buttonCode = 64
	case btn&tcell.WheelDown != 0:
		buttonCode = 65
	default:
		return true, nil
	}

	if event.Modifiers()&tcell.ModShift != 0 {
		buttonCode |= 4
	}
	if event.Modifiers()&tcell.ModAlt != 0 {
		buttonCode |= 8
	}
	if event.Modifiers()&tcell.ModCtrl != 0 {
		buttonCode |= 16
	}

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

	if action == tview.MouseLeftClick || action == tview.MouseLeftDown {
		setFocus(t)
	}

	return true, nil
}
