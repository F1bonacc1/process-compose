package tui

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/rivo/tview"
)

var (
	regionPattern = regexp.MustCompile(`\["([a-zA-Z0-9_,;: \-\.]*)"\]`)
	// ANSI escape sequence patterns
	// Clear entire screen sequences
	clearScreenPattern = regexp.MustCompile(`\x1b\[2J|\x1bc|\x1b\[H\x1b\[2J`)
)

type truncator interface {
	truncateLog()
}

type LogView struct {
	tview.TextView
	isWrapOn               bool
	buffer                 *bytes.Buffer
	ansiWriter             io.Writer
	useAnsi                bool
	uniqueId               string
	searchCurrentSelection int
	isSearching            bool
	searchTerm             string
	searchIndex            int
	totalSearchCount       int
	truncator              truncator
}

func NewLogView(maxLines int) *LogView {

	l := &LogView{
		isWrapOn: true,
		TextView: *tview.NewTextView().
			SetDynamicColors(true).
			SetScrollable(true).
			SetRegions(true).
			SetMaxLines(maxLines),
		buffer:                 &bytes.Buffer{},
		useAnsi:                false,
		uniqueId:               pclog.GenerateUniqueID(10),
		searchCurrentSelection: 0,
	}
	l.ansiWriter = tview.ANSIWriter(l)
	l.SetBorder(true)
	return l
}

func (l *LogView) WriteString(line string) (n int, err error) {
	if l.useAnsi {
		// Check for clear screen sequences
		if clearScreenPattern.MatchString(line) {
			l.Clear()
			l.truncator.truncateLog()
			// Remove the clear sequence and process remaining text
			line = clearScreenPattern.ReplaceAllString(line, "")
		}
		return l.buffer.WriteString(tview.Escape(line + "\n"))
	}
	if strings.Contains(strings.ToLower(line), "error") {
		return fmt.Fprintf(l.buffer, "[deeppink]%s[-:-:-]\n", tview.Escape(line))
	} else {
		return fmt.Fprintf(l.buffer, "%s\n", tview.Escape(line))
	}
}

func (l *LogView) AddLines(lines []string) {
	for _, line := range lines {
		_, _ = l.WriteString(line)
	}
}

func (l *LogView) SetLines(lines []string) {
	l.Clear()
	l.AddLines(lines)
}

func (l *LogView) GetUniqueID() string {
	return l.uniqueId
}

func (l *LogView) GetTailLength() int {
	return math.MaxInt
}

func (l *LogView) ToggleWrap() {
	l.isWrapOn = !l.isWrapOn
	l.SetWrap(l.isWrapOn)
}

func (l *LogView) IsWrapOn() bool {
	return l.isWrapOn
}

func (l *LogView) Flush() {
	if l.useAnsi {
		_, _ = l.buffer.WriteTo(l.ansiWriter)
	} else {
		_, _ = l.buffer.WriteTo(l)
	}
}

func (l *LogView) addRegions(regex *regexp.Regexp, text string) string {
	newText := regex.ReplaceAllStringFunc(text, func(match string) string {
		region := fmt.Sprintf(`["%d"]%s[""]`, l.totalSearchCount, match)
		l.totalSearchCount++
		return region
	})

	return newText
}

func (l *LogView) removeRegions() {
	text := regionPattern.ReplaceAllString(l.GetText(false), "")
	l.SetText(text)
}

func (l *LogView) searchString(search string, isRegex, caseSensitive bool) error {
	if search == "" {
		return nil
	}
	l.resetSearch()
	searchRegexString := search
	if !isRegex {
		searchRegexString = regexp.QuoteMeta(searchRegexString)
	}
	if !caseSensitive {
		searchRegexString = "(?i)" + searchRegexString
	}
	searchRegex, err := regexp.Compile(searchRegexString)
	if err != nil {
		return err
	}
	log := l.GetText(false)
	l.SetText(l.addRegions(searchRegex, strings.TrimSpace(log)))
	if l.totalSearchCount > 0 {
		l.Highlight("0").ScrollToHighlight()
	}
	l.isSearching = true
	l.searchTerm = search
	return nil
}

func (l *LogView) SearchNext() {
	if l.totalSearchCount > 0 {
		l.searchIndex = (l.searchIndex + 1) % l.totalSearchCount
		l.Highlight(strconv.Itoa(l.searchIndex)).ScrollToHighlight()
	}
}

func (l *LogView) SearchPrev() {
	if l.totalSearchCount > 0 {
		l.searchIndex = (l.searchIndex - 1 + l.totalSearchCount) % l.totalSearchCount
		l.Highlight(strconv.Itoa(l.searchIndex)).ScrollToHighlight()
	}
}

func (l *LogView) isSearchActive() bool {
	return l.isSearching
}

func (l *LogView) resetSearch() {
	if l.isSearching {
		l.isSearching = false
		l.searchIndex = 0
		l.totalSearchCount = 0
		l.Highlight()
		l.removeRegions()
	}
}

func (l *LogView) getSearchTerm() string {
	return l.searchTerm
}

func (l *LogView) getCurrentSearchIndex() int {
	if l.totalSearchCount == 0 {
		return -1
	}
	return l.searchIndex
}

func (l *LogView) getTotalSearchCount() int {
	return l.totalSearchCount
}

func (l *LogView) AddMark() {
	_, _, w, _ := l.GetInnerRect()
	mark := strings.Repeat("-", w)
	fmt.Fprintf(l.buffer, "%s\n", mark)
}

func (l *LogView) setTruncator(t truncator) {
	l.truncator = t
}
