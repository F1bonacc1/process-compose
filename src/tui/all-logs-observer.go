package tui

import (
	"fmt"
	"hash/fnv"
	"math"

	"github.com/f1bonacc1/process-compose/src/pclog"
)

// tviewColors is a list of tview color names for process name prefixes.
var tviewColors = []string{
	"red",
	"green",
	"yellow",
	"blue",
	"magenta",
	"cyan",
	"orange",
	"pink",
	"lime",
	"aqua",
	"violet",
	"gold",
}

// getProcessColor returns a tview color name for the given process name.
func getProcessColor(name string) string {
	hash := fnv.New32a()
	hash.Write([]byte(name))
	return tviewColors[int(hash.Sum32())%len(tviewColors)]
}

// AllLogsObserver wraps a LogView and prefixes log lines with a colored process name.
// It implements pclog.LogObserver interface.
type AllLogsObserver struct {
	processName string
	logView     *LogView
	color       string
	uniqueID    string
}

// NewAllLogsObserver creates a new AllLogsObserver for the given process.
func NewAllLogsObserver(processName string, logView *LogView) *AllLogsObserver {
	return &AllLogsObserver{
		processName: processName,
		logView:     logView,
		color:       getProcessColor(processName),
		uniqueID:    pclog.GenerateUniqueID(10),
	}
}

// WriteString writes a log line prefixed with the colored process name.
func (o *AllLogsObserver) WriteString(line string) (n int, err error) {
	return o.logView.WriteStringWithProcess(line, o.processName, o.color)
}

// SetLines sets multiple log lines, each prefixed with the colored process name.
func (o *AllLogsObserver) SetLines(lines []string) {
	for _, line := range lines {
		_, _ = o.WriteString(line)
	}
}

// GetTailLength returns the tail length for log subscription.
func (o *AllLogsObserver) GetTailLength() int {
	return math.MaxInt
}

// GetUniqueID returns the unique identifier for this observer.
func (o *AllLogsObserver) GetUniqueID() string {
	return fmt.Sprintf("all-logs-%s-%s", o.processName, o.uniqueID)
}
