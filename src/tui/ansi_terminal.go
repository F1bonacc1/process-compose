package tui

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog/log"
)

// Cell represents a single character cell in the terminal
type Cell struct {
	Char  rune
	Style tcell.Style
}

// AnsiTerminal is a simple ANSI escape sequence parser and terminal emulator
type AnsiTerminal struct {
	width         int
	height        int
	cells         [][]Cell
	cursorX       int
	cursorY       int
	cursorVisible bool
	currentStyle  tcell.Style
	lock          sync.Mutex

	// Alternate screen buffer
	mainScreen   [][]Cell
	altScreen    [][]Cell
	useAltScreen bool

	// Saved cursor positions
	savedCursorX int
	savedCursorY int
	savedStyle   tcell.Style

	// Character set designation (for ESC ( sequence)
	charSetMode byte

	// Parser state
	parseState       int
	escapeSeq        bytes.Buffer
	expectingCharSet bool // For ESC ( B sequences
	latentWrap       bool // Pending wrap state
	utf8Buffer       []byte

	// Scrolling region
	scrollTop    int
	scrollBottom int

	// Callback for sending responses back to the PTY
	responseCallback func([]byte)
}

const (
	stateNormal = iota
	stateEscape
	stateCSI
	stateDCS
	stateOSC
)

// NewAnsiTerminal creates a new ANSI terminal emulator
func NewAnsiTerminal(width, height int) *AnsiTerminal {
	t := &AnsiTerminal{
		width:         width,
		height:        height,
		cursorVisible: true,
		currentStyle:  tcell.StyleDefault,
		parseState:    stateNormal,
		scrollTop:     0,
		scrollBottom:  height - 1,
		utf8Buffer:    make([]byte, 0, 4),
	}
	t.cells = make([][]Cell, height)
	for i := range t.cells {
		t.cells[i] = make([]Cell, width)
		for j := range t.cells[i] {
			t.cells[i][j] = Cell{Char: ' ', Style: tcell.StyleDefault}
		}
	}
	return t
}

// SetResponseCallback sets the callback for sending responses back to the PTY
func (t *AnsiTerminal) SetResponseCallback(callback func([]byte)) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.responseCallback = callback
}

// Resize changes the terminal dimensions
func (t *AnsiTerminal) Resize(width, height int) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if width == t.width && height == t.height {
		return
	}

	// Helper to resize a screen buffer
	resizeBuffer := func(oldBuf [][]Cell) [][]Cell {
		newBuf := make([][]Cell, height)
		for i := range newBuf {
			newBuf[i] = make([]Cell, width)
			for j := range newBuf[i] {
				newBuf[i][j] = Cell{Char: ' ', Style: t.currentStyle}
			}
		}

		// Copy content
		for i := 0; i < height && i < len(oldBuf); i++ {
			for j := 0; j < width && j < len(oldBuf[i]); j++ {
				newBuf[i][j] = oldBuf[i][j]
			}
		}
		return newBuf
	}

	if t.useAltScreen {
		// Resize alt screen (current t.cells)
		t.cells = resizeBuffer(t.cells)
		t.altScreen = t.cells // Update reference

		// Resize main screen if it exists
		if t.mainScreen != nil {
			t.mainScreen = resizeBuffer(t.mainScreen)
		}
	} else {
		// Resize main screen (current t.cells)
		t.cells = resizeBuffer(t.cells)
		// t.mainScreen is nil in this mode

		// Also resize altScreen if it exists, to keep it in sync
		if t.altScreen != nil {
			t.altScreen = resizeBuffer(t.altScreen)
		}
	}

	t.width = width
	t.height = height

	// Adjust cursor if needed
	if t.cursorX >= width {
		t.cursorX = width - 1
	}
	if t.cursorY >= height {
		t.cursorY = height - 1
	}

	// Reset scrolling region to full screen on resize
	t.scrollTop = 0
	t.scrollBottom = height - 1
}

// Write processes incoming data and updates terminal state
func (t *AnsiTerminal) Write(data []byte) {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, b := range data {
		if t.parseState == stateNormal {
			// Handle UTF-8 buffering
			if len(t.utf8Buffer) > 0 || b >= 128 {
				t.utf8Buffer = append(t.utf8Buffer, b)

				// Determine if we have a complete rune
				if utf8.FullRune(t.utf8Buffer) {
					r, width := utf8.DecodeRune(t.utf8Buffer)
					t.utf8Buffer = t.utf8Buffer[:0]

					if r == utf8.RuneError && width == 1 {
						// Invalid UTF-8, print replacement character or ignore?
						// Usually terminals print specific replacement char
						t.putChar(r)
					} else {
						t.putChar(r)
					}
				}
				continue
			}
		}

		switch t.parseState {
		case stateNormal:
			t.handleNormal(b)
		case stateEscape:
			t.handleEscape(b)
		case stateCSI:
			t.handleCSI(b)
		case stateDCS:
			t.handleDCS(b)
		case stateOSC:
			t.handleOSC(b)
		}
	}
}

func (t *AnsiTerminal) handleNormal(b byte) {
	// Handle character set designation continuation
	if t.expectingCharSet {
		t.charSetMode = b // Store it but we don't really use it
		t.expectingCharSet = false
		return
	}

	switch b {
	case '\x1b': // ESC
		t.parseState = stateEscape
		t.escapeSeq.Reset()
	case '\n': // Line feed - move down one line and to column 0 (LF+CR behavior)
		t.latentWrap = false
		t.cursorX = 0
		t.cursorY++
		if t.cursorY > t.scrollBottom {
			t.scrollUp()
			t.cursorY = t.scrollBottom
		}
	case '\r': // Carriage return - move to start of line
		t.latentWrap = false
		t.cursorX = 0
	case '\t': // Tab
		t.latentWrap = false
		t.cursorX = (t.cursorX + 8) & ^7
		if t.cursorX >= t.width {
			t.cursorX = t.width - 1
		}
	case '\b': // Backspace
		t.latentWrap = false
		if t.cursorX > 0 {
			t.cursorX--
		}
	default:
		if b >= 32 { // Printable character
			t.putChar(rune(b))
		}
	}
}

func (t *AnsiTerminal) handleEscape(b byte) {
	switch b {
	case '[':
		t.parseState = stateCSI
		t.escapeSeq.Reset()
	case '7':
		// Save cursor (ESC 7)
		t.saveCursor()
		t.parseState = stateNormal
	case '8':
		// Restore cursor (ESC 8)
		t.restoreCursor()
		t.parseState = stateNormal
	case 'M':
		// Reverse Index (ESC M) - move cursor up, scroll down if at top margin
		if t.cursorY == t.scrollTop {
			t.scrollDown()
		} else {
			t.cursorY--
			if t.cursorY < 0 {
				t.cursorY = 0
			}
		}
		t.parseState = stateNormal
	case '=':
		// Application keypad mode (ESC =) - ignore
		t.parseState = stateNormal
	case '>':
		// Normal keypad mode (ESC >) - ignore
		t.parseState = stateNormal
	case '(':
		// Designate G0 character set (ESC ( X) - expect one more character
		t.expectingCharSet = true
		t.parseState = stateNormal
	case 'P':
		// DCS - Device Control String (ESC P ... ST)
		// Used by vim for various control sequences
		// We'll consume until we see ST (ESC \)
		t.parseState = stateDCS
		t.escapeSeq.Reset()
	case ']':
		// OSC - Operating System Command (ESC ] ... ST or BEL)
		// Used for setting window title, etc.
		t.parseState = stateOSC
		t.escapeSeq.Reset()
	case '\\':
		// ST - String Terminator (ESC \)
		// End of DCS or OSC sequence
		t.parseState = stateNormal
	default:
		// Other escape sequences - log and ignore
		log.Debug().Msgf("Unhandled ESC sequence: ESC %c (0x%02x)", b, b)
		t.parseState = stateNormal
	}
}

func (t *AnsiTerminal) handleCSI(b byte) {
	if b >= '0' && b <= '9' || b == ';' || b == '?' {
		t.escapeSeq.WriteByte(b)
	} else {
		// Terminal byte for CSI sequence
		t.executeCSI(b)
		t.parseState = stateNormal
	}
}

func (t *AnsiTerminal) executeCSI(command byte) {
	params := t.parseCSIParams()

	switch command {
	case '@', 'L', 'M', 'P', 'X':
		t.handleTextModification(command, params)

	case 'A', 'B', 'C', 'D', 'G', 'H', 'f', 'd', 'e':
		t.handleCursorMovement(command, params)

	case 'J', 'K':
		t.handleErase(command, params)

	case 'S', 'T':
		t.handleScrolling(command, params)

	case 'm': // SGR - Select Graphic Rendition
		t.handleSGR(params)

	case 'r': // Set scrolling region
		top, bottom := 1, t.height
		if len(params) > 0 {
			top = params[0]
		}
		if len(params) > 1 {
			bottom = params[1]
		}
		if top < 1 {
			top = 1
		}
		if bottom > t.height {
			bottom = t.height
		}
		if top > bottom {
			top = 1
			bottom = t.height
		}

		t.scrollTop = top - 1
		t.scrollBottom = bottom - 1
		// Cursor moves to home position
		t.cursorX = 0
		t.cursorY = 0

	case 's': // Save cursor position (CSI s)
		t.saveCursor()

	case 'u': // Restore cursor position (CSI u)
		t.restoreCursor()

	case 'h': // Set mode
		t.handleSetMode(params)

	case 'l': // Reset mode
		t.handleResetMode(params)

	case 'n': // Device Status Report (DSR)
		t.handleDSR(params)

	default:
		// Log unhandled CSI sequences
	}
}

func (t *AnsiTerminal) handleTextModification(command byte, params []int) {
	n := 1
	if len(params) > 0 {
		n = params[0]
	}

	switch command {
	case '@': // Insert character (ICH)
		t.insertChar(n)
	case 'L': // Insert line
		t.insertLine(n)
	case 'M': // Delete line
		t.deleteLine(n)
	case 'P': // Delete character (DCH)
		t.deleteChar(n)
	case 'X': // Erase character (ECH)
		t.eraseChar(n)
	}
}

func (t *AnsiTerminal) handleCursorMovement(command byte, params []int) {
	n := 1
	if len(params) > 0 {
		n = params[0]
	}

	switch command {
	case 'A': // Cursor up
		t.cursorY -= n
		if t.cursorY < 0 {
			t.cursorY = 0
		}
		t.latentWrap = false

	case 'B': // Cursor down
		t.cursorY += n
		if t.cursorY >= t.height {
			t.cursorY = t.height - 1
		}
		t.latentWrap = false

	case 'C': // Cursor forward
		t.cursorX += n
		if t.cursorX >= t.width {
			t.cursorX = t.width - 1
		}
		t.latentWrap = false

	case 'D': // Cursor back
		t.cursorX -= n
		if t.cursorX < 0 {
			t.cursorX = 0
		}
		t.latentWrap = false

	case 'G': // Cursor Horizontal Absolute (CHA)
		t.cursorX = n - 1
		if t.cursorX < 0 {
			t.cursorX = 0
		}
		if t.cursorX >= t.width {
			t.cursorX = t.width - 1
		}
		t.latentWrap = false

	case 'H', 'f': // Cursor position
		row, col := 1, 1
		if len(params) > 0 {
			row = params[0]
		}
		if len(params) > 1 {
			col = params[1]
		}
		t.cursorY = row - 1
		t.cursorX = col - 1
		t.latentWrap = false

		if t.cursorY < 0 {
			t.cursorY = 0
		}
		if t.cursorY >= t.height {
			t.cursorY = t.height - 1
		}
		if t.cursorX < 0 {
			t.cursorX = 0
		}
		if t.cursorX >= t.width {
			t.cursorX = t.width - 1
		}

	case 'd': // Line Position Absolute (VPA)
		t.cursorY = n - 1
		if t.cursorY < 0 {
			t.cursorY = 0
		}
		if t.cursorY >= t.height {
			t.cursorY = t.height - 1
		}
		t.latentWrap = false

	case 'e': // Line Position Relative (VPR)
		t.cursorY += n
		if t.cursorY >= t.height {
			t.cursorY = t.height - 1
		}
		t.latentWrap = false
	}
}

func (t *AnsiTerminal) handleErase(command byte, params []int) {
	mode := 0
	if len(params) > 0 {
		mode = params[0]
	}
	switch command {
	case 'J': // Erase in display
		t.eraseDisplay(mode)
	case 'K': // Erase in line
		t.eraseLine(mode)
	}
}

func (t *AnsiTerminal) handleScrolling(command byte, params []int) {
	n := 1
	if len(params) > 0 {
		n = params[0]
	}
	for i := 0; i < n; i++ {
		if command == 'S' { // Scroll Up
			t.scrollUp()
		} else { // Scroll Down
			t.scrollDown()
		}
	}
}

func (t *AnsiTerminal) parseCSIParams() []int {
	if t.escapeSeq.Len() == 0 {
		return nil
	}

	paramsStr := t.escapeSeq.String()
	// Strip '?' prefix if present (for private modes like ?1049)
	if len(paramsStr) > 0 && paramsStr[0] == '?' {
		paramsStr = paramsStr[1:]
	}

	parts := bytes.Split([]byte(paramsStr), []byte(";"))
	params := make([]int, 0, len(parts))

	for _, part := range parts {
		if len(part) == 0 {
			params = append(params, 0)
			continue
		}
		n, err := strconv.Atoi(string(part))
		if err == nil {
			params = append(params, n)
		}
	}
	return params
}

func (t *AnsiTerminal) handleSGR(params []int) {
	if len(params) == 0 {
		params = []int{0}
	}

	for i := 0; i < len(params); i++ {
		param := params[i]
		switch {
		case param == 0: // Reset
			t.currentStyle = tcell.StyleDefault
		case param == 1: // Bold
			t.currentStyle = t.currentStyle.Bold(true)
		case param == 4: // Underline
			t.currentStyle = t.currentStyle.Underline(true)
		case param == 7: // Reverse
			t.currentStyle = t.currentStyle.Reverse(true)
		case param >= 30 && param <= 37: // Foreground color
			t.currentStyle = t.currentStyle.Foreground(ansiColor256(param - 30))
		case param >= 90 && param <= 97: // Bright Foreground color
			t.currentStyle = t.currentStyle.Foreground(ansiColor256(param - 90 + 8))
		case param == 38: // Extended Foreground
			if i+2 < len(params) && params[i+1] == 5 {
				n := params[i+2]
				t.currentStyle = t.currentStyle.Foreground(ansiColor256(n))
				i += 2
			} else if i+4 < len(params) && params[i+1] == 2 {
				// RGB color: 38;2;r;g;b
				r := int32(params[i+2])
				g := int32(params[i+3])
				b := int32(params[i+4])
				t.currentStyle = t.currentStyle.Foreground(tcell.NewRGBColor(r, g, b))
				i += 4
			}
		case param >= 40 && param <= 47: // Background color
			t.currentStyle = t.currentStyle.Background(ansiColor256(param - 40))
		case param >= 100 && param <= 107: // Bright Background color
			t.currentStyle = t.currentStyle.Background(ansiColor256(param - 100 + 8))
		case param == 48: // Extended Background
			if i+2 < len(params) && params[i+1] == 5 {
				n := params[i+2]
				t.currentStyle = t.currentStyle.Background(ansiColor256(n))
				i += 2
			} else if i+4 < len(params) && params[i+1] == 2 {
				// RGB color: 48;2;r;g;b
				r := int32(params[i+2])
				g := int32(params[i+3])
				b := int32(params[i+4])
				t.currentStyle = t.currentStyle.Background(tcell.NewRGBColor(r, g, b))
				i += 4
			}
		case param == 39: // Default foreground
			t.currentStyle = t.currentStyle.Foreground(tcell.ColorDefault)
		case param == 49: // Default background
			t.currentStyle = t.currentStyle.Background(tcell.ColorDefault)
		}
	}
}

func ansiColor256(n int) tcell.Color {
	return tcell.PaletteColor(n)
}

func (t *AnsiTerminal) putChar(ch rune) {
	if t.latentWrap {
		t.cursorX = 0
		t.cursorY++
		if t.cursorY > t.scrollBottom {
			t.scrollUp()
			t.cursorY = t.scrollBottom
		}
		t.latentWrap = false
	}

	if t.cursorY >= 0 && t.cursorY < t.height && t.cursorX >= 0 && t.cursorX < t.width {
		t.cells[t.cursorY][t.cursorX] = Cell{Char: ch, Style: t.currentStyle}
	}

	t.cursorX++
	if t.cursorX >= t.width {
		t.cursorX = t.width - 1
		t.latentWrap = true
	}
}

func (t *AnsiTerminal) insertLine(n int) {
	// Only insert within scrolling region
	if t.cursorY < t.scrollTop || t.cursorY > t.scrollBottom {
		return
	}

	for i := 0; i < n; i++ {
		// Move lines down from cursor to bottom of region
		for y := t.scrollBottom; y > t.cursorY; y-- {
			copy(t.cells[y], t.cells[y-1])
		}
		// Clear current line
		for x := 0; x < t.width; x++ {
			t.cells[t.cursorY][x] = Cell{Char: ' ', Style: t.currentStyle}
		}
	}
}

func (t *AnsiTerminal) deleteLine(n int) {
	// Only delete within scrolling region
	if t.cursorY < t.scrollTop || t.cursorY > t.scrollBottom {
		return
	}

	for i := 0; i < n; i++ {
		// Move lines up from cursor+1 to bottom of region
		for y := t.cursorY; y < t.scrollBottom; y++ {
			copy(t.cells[y], t.cells[y+1])
		}
		// Clear bottom line of region
		for x := 0; x < t.width; x++ {
			t.cells[t.scrollBottom][x] = Cell{Char: ' ', Style: t.currentStyle}
		}
	}
}

func (t *AnsiTerminal) scrollUp() {
	// Only scroll within the scrolling region
	if t.scrollTop >= t.scrollBottom {
		return
	}

	// Move lines up
	for i := t.scrollTop; i < t.scrollBottom; i++ {
		copy(t.cells[i], t.cells[i+1])
	}

	// Clear bottom line of the region
	for i := 0; i < t.width; i++ {
		t.cells[t.scrollBottom][i] = Cell{Char: ' ', Style: t.currentStyle}
	}
}

func (t *AnsiTerminal) scrollDown() {
	// Only scroll within the scrolling region
	if t.scrollTop >= t.scrollBottom {
		return
	}

	// Move lines down
	for i := t.scrollBottom; i > t.scrollTop; i-- {
		copy(t.cells[i], t.cells[i-1])
	}

	// Clear top line of the region
	for i := 0; i < t.width; i++ {
		t.cells[t.scrollTop][i] = Cell{Char: ' ', Style: t.currentStyle}
	}
}

func (t *AnsiTerminal) eraseDisplay(mode int) {
	switch mode {
	case 0: // Erase from cursor to end of screen
		t.eraseLine(0)
		for i := t.cursorY + 1; i < t.height; i++ {
			for j := 0; j < t.width; j++ {
				t.cells[i][j] = Cell{Char: ' ', Style: t.currentStyle}
			}
		}
	case 1: // Erase from start to cursor
		for i := 0; i < t.cursorY; i++ {
			for j := 0; j < t.width; j++ {
				t.cells[i][j] = Cell{Char: ' ', Style: t.currentStyle}
			}
		}
		t.eraseLine(1)
	case 2, 3: // Erase entire screen
		for i := 0; i < t.height; i++ {
			for j := 0; j < t.width; j++ {
				t.cells[i][j] = Cell{Char: ' ', Style: t.currentStyle}
			}
		}
	}
}

func (t *AnsiTerminal) eraseLine(mode int) {
	switch mode {
	case 0: // Erase from cursor to end of line
		for j := t.cursorX; j < t.width; j++ {
			t.cells[t.cursorY][j] = Cell{Char: ' ', Style: t.currentStyle}
		}
	case 1: // Erase from start of line to cursor
		for j := 0; j <= t.cursorX && j < t.width; j++ {
			t.cells[t.cursorY][j] = Cell{Char: ' ', Style: t.currentStyle}
		}
	case 2: // Erase entire line
		for j := 0; j < t.width; j++ {
			t.cells[t.cursorY][j] = Cell{Char: ' ', Style: t.currentStyle}
		}
	}
}

// GetCell returns the cell at the given position
func (t *AnsiTerminal) GetCell(x, y int) Cell {
	t.lock.Lock()
	defer t.lock.Unlock()

	if y >= 0 && y < t.height && x >= 0 && x < t.width {
		return t.cells[y][x]
	}
	return Cell{Char: ' ', Style: tcell.StyleDefault}
}

// GetCursor returns the current cursor position
// insertChar inserts n characters at the current cursor position
func (t *AnsiTerminal) insertChar(n int) {
	if t.cursorX >= t.width {
		return
	}

	// Shift characters to the right
	copy(t.cells[t.cursorY][t.cursorX+n:], t.cells[t.cursorY][t.cursorX:])

	// Clear the inserted space
	for i := 0; i < n && t.cursorX+i < t.width; i++ {
		t.cells[t.cursorY][t.cursorX+i] = Cell{Char: ' ', Style: t.currentStyle}
	}
}

// deleteChar deletes n characters at the current cursor position
func (t *AnsiTerminal) deleteChar(n int) {
	if t.cursorX >= t.width {
		return
	}

	// Shift characters to the left
	if t.cursorX+n < t.width {
		copy(t.cells[t.cursorY][t.cursorX:], t.cells[t.cursorY][t.cursorX+n:])
	}

	// Clear the space at the end
	startClear := t.width - n
	if startClear < t.cursorX {
		startClear = t.cursorX
	}
	for i := startClear; i < t.width; i++ {
		t.cells[t.cursorY][i] = Cell{Char: ' ', Style: t.currentStyle}
	}
}

// eraseChar erases n characters at the current cursor position
func (t *AnsiTerminal) eraseChar(n int) {
	if t.cursorX >= t.width {
		return
	}

	for i := 0; i < n && t.cursorX+i < t.width; i++ {
		t.cells[t.cursorY][t.cursorX+i] = Cell{Char: ' ', Style: t.currentStyle}
	}
}

func (t *AnsiTerminal) GetCursor() (int, int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.cursorX, t.cursorY
}

// IsCursorVisible returns whether the cursor should be displayed
func (t *AnsiTerminal) IsCursorVisible() bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.cursorVisible
}

func (t *AnsiTerminal) saveCursor() {
	t.savedCursorX = t.cursorX
	t.savedCursorY = t.cursorY
	t.savedStyle = t.currentStyle
}

func (t *AnsiTerminal) restoreCursor() {
	t.cursorX = t.savedCursorX
	t.cursorY = t.savedCursorY
	t.currentStyle = t.savedStyle
	t.latentWrap = false
}

func (t *AnsiTerminal) handleDSR(params []int) {
	if len(params) == 0 {
		return
	}

	switch params[0] {
	case 5: // Status Report - always report OK
		if t.responseCallback != nil {
			t.responseCallback([]byte("\x1b[0n"))
		}
	case 6: // Report Cursor Position (CPR)
		if t.responseCallback != nil {
			// Response: CSI <row>;<col> R (1-based)
			response := fmt.Sprintf("\x1b[%d;%dR", t.cursorY+1, t.cursorX+1)
			t.responseCallback([]byte(response))
		}
	}
}

func (t *AnsiTerminal) handleSetMode(params []int) {
	for _, p := range params {
		switch p {
		case 25: // Show cursor
			t.cursorVisible = true
		case 1049: // Enable alternate screen buffer
			if !t.useAltScreen {
				t.switchToAltScreen()
			}
		}
	}
}

func (t *AnsiTerminal) handleResetMode(params []int) {
	for _, p := range params {
		switch p {
		case 25: // Hide cursor
			t.cursorVisible = false
		case 1049: // Disable alternate screen buffer
			if t.useAltScreen {
				t.switchToMainScreen()
			}
		}
	}
}

func (t *AnsiTerminal) switchToAltScreen() {
	// Save main screen
	t.mainScreen = make([][]Cell, t.height)
	for i := range t.mainScreen {
		t.mainScreen[i] = make([]Cell, t.width)
		copy(t.mainScreen[i], t.cells[i])
	}

	// Create and switch to alt screen
	t.altScreen = make([][]Cell, t.height)
	for i := range t.altScreen {
		t.altScreen[i] = make([]Cell, t.width)
		for j := range t.altScreen[i] {
			t.altScreen[i][j] = Cell{Char: ' ', Style: t.currentStyle}
		}
	}
	t.cells = t.altScreen
	t.useAltScreen = true
	t.cursorX = 0
	t.cursorY = 0
	// Reset scrolling region to full screen
	t.scrollTop = 0
	t.scrollBottom = t.height - 1
}

func (t *AnsiTerminal) switchToMainScreen() {
	// Restore main screen
	if t.mainScreen != nil {
		t.cells = t.mainScreen
		t.mainScreen = nil
	}
	t.altScreen = nil
	t.useAltScreen = false
	// Reset scrolling region to full screen
	t.scrollTop = 0
	t.scrollBottom = t.height - 1
}

func (t *AnsiTerminal) handleDCS(b byte) {
	if b == 0x1b { // ESC
		t.parseState = stateEscape
	}
	// Otherwise ignore content of DCS sequence
}

func (t *AnsiTerminal) handleOSC(b byte) {
	switch b {
	case 0x1b: // ESC
		t.parseState = stateEscape
	case 0x07: // BEL
		t.parseState = stateNormal
	}
	// Otherwise ignore content of OSC sequence
}
