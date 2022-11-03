package tui

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/rivo/tview"
)

type LogView struct {
	tview.TextView
	isWrapOn   bool
	buffer     *strings.Builder
	ansiWriter io.Writer
	mx         sync.Mutex
	useAnsi    bool
}

func NewLogView(maxLines int) *LogView {

	l := &LogView{
		isWrapOn: true,
		TextView: *tview.NewTextView().SetDynamicColors(true).SetScrollable(true).SetMaxLines(maxLines),
		buffer:   &strings.Builder{},
		useAnsi:  false,
	}
	l.ansiWriter = tview.ANSIWriter(l)
	l.SetBorder(true)
	return l
}

func (l *LogView) AddLine(line string) {
	l.mx.Lock()
	defer l.mx.Unlock()
	if l.useAnsi {
		fmt.Fprintf(l.buffer, "%s\n", line)
		return
	}
	if strings.Contains(strings.ToLower(line), "error") {
		fmt.Fprintf(l.buffer, "[deeppink]%s[-:-:-]\n", tview.Escape(line))
	} else {
		fmt.Fprintf(l.buffer, "%s\n", tview.Escape(line))
	}
}

func (l *LogView) AddLines(lines []string) {
	for _, line := range lines {
		l.AddLine(line)
	}
}

func (l *LogView) SetLines(lines []string) {
	l.Clear()
	l.AddLines(lines)
}

func (l *LogView) ToggleWrap() {
	l.isWrapOn = !l.isWrapOn
	l.SetWrap(l.isWrapOn)
}

func (l *LogView) IsWrapOn() bool {
	return l.isWrapOn
}

func (l *LogView) Flush() {
	l.mx.Lock()
	defer l.mx.Unlock()
	if l.useAnsi {
		l.ansiWriter.Write([]byte(l.buffer.String()))
	} else {
		l.Write([]byte(l.buffer.String()))
	}

	l.buffer.Reset()
}
