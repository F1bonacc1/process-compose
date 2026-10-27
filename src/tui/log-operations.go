package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/f1bonacc1/glippy"
	"github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog/log"
)

func (pv *pcView) toggleLogSelection() {
	name := pv.getSelectedProcName()
	pv.logSelect = !pv.logSelect
	if pv.logSelect {
		row, col := pv.logsText.GetScrollOffset()
		pv.logsTextArea.SetText(pv.logsText.GetText(true), false).
			SetBorder(true).
			SetTitle(name + " [Select & Press Enter to Copy]")
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

// showLog switches the log pane to the named process. It follows the new
// process's log when log following is on, and only loads its backlog when off.
func (pv *pcView) showLog(name string) {
	pv.unFollowLog()
	if pv.logFollow {
		pv.followLog(name)
	} else {
		pv.loadLog(name)
	}
}

func (pv *pcView) followLog(name string) {
	if !pv.resetLogView(name) {
		return
	}
	if err := pv.project.GetLogsAndSubscribe(name, pv.logsText); err != nil {
		pv.attentionMessage(fmt.Sprintf("Couldn't subscribe to the process logs: %s", err.Error()), 5*time.Second, true)
		return
	}
	pv.logsText.ScrollToEnd()
}

// loadLog shows the current backlog of the named process without subscribing
// to new lines. The backlog is fetched synchronously: a remote project delivers
// the backlog of a subscription asynchronously, so subscribing and immediately
// unsubscribing would leave the pane empty.
func (pv *pcView) loadLog(name string) {
	if !pv.resetLogView(name) {
		return
	}
	lines, err := pv.project.GetProcessLog(name, 0, pv.project.GetLogLength())
	if err != nil {
		pv.attentionMessage(fmt.Sprintf("Couldn't load the process logs: %s", err.Error()), 5*time.Second, true)
		return
	}
	pv.logsText.SetLines(lines)
	pv.logsText.Flush()
	pv.logsText.ScrollToEnd()
}

func (pv *pcView) resetLogView(name string) bool {
	pv.loggedProc = name
	pv.logsText.Clear()
	config, err := pv.project.GetProcessInfo(name)
	if err != nil {
		return false
	}
	pv.logsText.useAnsi = !config.DisableAnsiColors
	return true
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
		switch event.Key() {
		case tcell.KeyCR:
			text, start, _ := pv.logsTextArea.GetSelection()
			method, err := glippy.SetWithMethod(text)
			if err != nil {
				log.Err(err).Msg("failed to set clipboard")
				pv.attentionMessage(fmt.Sprintf("Failed to copy to clipboard: %s", err.Error()), 5*time.Second, true)
			} else if method == glippy.MethodOSC52 {
				pv.attentionMessage("Copied via OSC 52 (terminal must allow OSC 52)", 3*time.Second, false)
			} else {
				pv.attentionMessage("Copied to clipboard", 2*time.Second, false)
			}
			pv.logsTextArea.Select(start, start)
		case tcell.KeyEsc:
			pv.toggleLogSelection()
			pv.updateHelpTextView()
		}
		return nil
	})
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
