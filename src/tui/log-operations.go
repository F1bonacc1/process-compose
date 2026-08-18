package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/f1bonacc1/glippy"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

func (pv *pcView) toggleLogSelection() {
	name := pv.getSelectedProcName()
	pv.logSelect = !pv.logSelect
	if pv.logSelect {
		row, col := pv.logsText.GetScrollOffset()
		pv.logsTextArea.SetText(pv.logsText.GetText(true), false).
			SetBorder(true).
			SetTitle(fmt.Sprintf("%s [Select to Copy (or press %s)]", name, pv.logCopyShortcut()))
		pv.logsTextArea.SetOffset(row, col)
	} else {
		pv.logsTextArea.SetText("", false)
	}

	pv.redrawGrid()
}

func (pv *pcView) toggleLogFollow() {
	if pv.logFollow {
		pv.stopFollowLog()
	} else {
		name := pv.getSelectedProcName()
		pv.startFollowLog(name)
	}
}

func (pv *pcView) startFollowLog(name string) {
	pv.exitSearch()
	pv.logFollow = true
	pv.followLog(name)
	var ctx context.Context
	ctx, pv.cancelLogFn = context.WithCancel(context.Background())
	go pv.updateLogs(ctx)
	pv.updateHelpTextView()
}

func (pv *pcView) stopFollowLog() {
	pv.logFollow = false
	if pv.cancelLogFn != nil {
		pv.cancelLogFn()
		pv.cancelLogFn = nil
	}
	pv.unFollowLog()
	pv.updateHelpTextView()
}

func (pv *pcView) followLog(name string) {
	pv.loggedProc = name
	pv.logsText.Clear()
	config, err := pv.project.GetProcessInfo(name)
	if err != nil {
		return
	}
	pv.logsText.useAnsi = !config.DisableAnsiColors
	if err = pv.project.GetLogsAndSubscribe(name, pv.logsText); err != nil {
		pv.attentionMessage(fmt.Sprintf("Couldn't subscribe to the process logs: %s", err.Error()), 5*time.Second, true)
		return
	}
	pv.logsText.ScrollToEnd()
}

func (pv *pcView) unFollowLog() {
	if pv.loggedProc != "" {
		if err := pv.project.UnSubscribeLogger(pv.loggedProc, pv.logsText); err != nil {
			log.Err(err).Msg("failed to unfollow log")
		}
	}
	pv.logsText.Flush()
}

func (pv *pcView) updateLogs(ctx context.Context) {
	pv.appView.QueueUpdateDraw(func() {
		pv.logsText.Flush()
	})
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Logs monitoring canceled")
			return
		case <-time.After(300 * time.Millisecond):
			pv.appView.QueueUpdateDraw(func() {
				pv.logsText.Flush()
			})
		}
	}
}

func (pv *pcView) createLogSelectionTextArea() {
	pv.logsTextArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case pv.isLogCopyEvent(event):
			pv.copyLogSelection(true)
		case event.Key() == tcell.KeyEsc:
			pv.toggleLogSelection()
			pv.updateHelpTextView()
		}
		return nil
	})

	// Copy-on-select: completing a mouse selection (drag release) copies it to
	// the clipboard automatically, no key press required. The capture runs
	// before the text area's own handler, but the selection has already been
	// extended by the preceding mouse-move events, so it is current here.
	pv.logsTextArea.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if pv.logSelect && action == tview.MouseLeftUp {
			pv.copyLogSelection(false)
		}
		return action, event
	})
}

// copyLogSelection copies the current text-area selection to the clipboard.
// When collapse is true the selection is cleared afterwards (used by the
// keyboard path); the mouse path keeps the highlight visible. Empty selections
// are ignored so a stray click or key press never clobbers the clipboard.
func (pv *pcView) copyLogSelection(collapse bool) {
	text, start, _ := pv.logsTextArea.GetSelection()
	if len(text) == 0 {
		return
	}
	method, err := glippy.SetWithMethod(text)
	if err != nil {
		log.Err(err).Msg("failed to set clipboard")
		pv.attentionMessage(fmt.Sprintf("Failed to copy to clipboard: %s", err.Error()), 5*time.Second, true)
	} else if method == glippy.MethodOSC52 {
		pv.attentionMessage("Copied via OSC 52 (terminal must allow OSC 52)", 3*time.Second, false)
	} else {
		pv.attentionMessage("Copied to clipboard", 2*time.Second, false)
	}
	if collapse {
		pv.logsTextArea.Select(start, start)
	}
}

// logCopyShortcut returns the human-readable label of the configured copy
// shortcut (e.g. "Enter", "Ctrl-Y", "y"), falling back to "Enter".
func (pv *pcView) logCopyShortcut() string {
	if action := pv.shortcuts.ShortCutKeys[ActionLogCopy]; action != nil && action.ShortCut != "" {
		return action.ShortCut
	}
	return tcell.KeyNames[tcell.KeyCR]
}

// isLogCopyEvent reports whether the given key event matches the configured
// copy-selection shortcut. The shortcut may be bound to either a special key
// (e.g. Enter, Ctrl-Y) or a single rune (e.g. y).
func (pv *pcView) isLogCopyEvent(event *tcell.EventKey) bool {
	action := pv.shortcuts.ShortCutKeys[ActionLogCopy]
	if action == nil {
		return event.Key() == tcell.KeyCR
	}
	if action.rune != 0 {
		return event.Key() == tcell.KeyRune && event.Rune() == action.rune
	}
	return event.Key() == action.key
}

func (pv *pcView) getLogTitle(name string) string {
	if pv.logsText.isSearchActive() {
		return fmt.Sprintf("Find: %s [%d of %d] - %s", pv.logsText.getSearchTerm(), pv.logsText.getCurrentSearchIndex()+1, pv.logsText.getTotalSearchCount(), name)
	} else {
		return name
	}
}

func (pv *pcView) truncateLog() {
	name := pv.getSelectedProcName()
	err := pv.project.TruncateProcessLogs(name)
	if err != nil {
		log.Err(err).Msgf("failed to truncate process %s logs", name)
	}
}
