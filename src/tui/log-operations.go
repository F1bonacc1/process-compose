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
		// In all-logs mode, name will be empty but startFollowLog handles it
		pv.startFollowLog(name)
	}
}

func (pv *pcView) startFollowLog(name string) {
	pv.exitSearch()
	pv.logFollow = true
	if pv.allLogsMode {
		pv.followAllLogs()
	} else {
		pv.followLog(name)
	}
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
	if pv.allLogsMode {
		pv.unFollowAllLogs()
	} else {
		pv.unFollowLog()
	}
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
		pv.attentionMessage(fmt.Sprintf("Couldn't subscribe to the process logs: %s", err.Error()), 5*time.Second)
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
		switch event.Key() {
		case tcell.KeyCR:
			text, start, _ := pv.logsTextArea.GetSelection()
			err := glippy.Set(text)
			if err != nil {
				log.Err(err).Msg("failed to set clipboard")
				pv.attentionMessage(fmt.Sprintf("Failed to copy to clipboard: %s", err.Error()), 5*time.Second)
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
	displayName := name
	if pv.allLogsMode {
		displayName = "All Processes"
	}
	if pv.logsText.isSearchActive() {
		return fmt.Sprintf("Find: %s [%d of %d] - %s", pv.logsText.getSearchTerm(), pv.logsText.getCurrentSearchIndex()+1, pv.logsText.getTotalSearchCount(), displayName)
	}
	return displayName
}

func (pv *pcView) truncateLog() {
	name := pv.getSelectedProcName()
	err := pv.project.TruncateProcessLogs(name)
	if err != nil {
		log.Err(err).Msgf("failed to truncate process %s logs", name)
	}
}

// followAllLogs subscribes to logs from all processes and displays them with process name prefixes.
func (pv *pcView) followAllLogs() {
	pv.logsText.Clear()
	pv.logsText.useAnsi = false // Process name prefixes use tview color tags

	for _, name := range pv.procNames {
		observer := NewAllLogsObserver(name, pv.logsText)
		pv.allLogsObservers[name] = observer
		if err := pv.project.GetLogsAndSubscribe(name, observer); err != nil {
			log.Err(err).Msgf("failed to subscribe to logs for process %s", name)
		}
	}
	pv.logsText.ScrollToEnd()
}

// unFollowAllLogs unsubscribes from all process logs.
func (pv *pcView) unFollowAllLogs() {
	for name, observer := range pv.allLogsObservers {
		if err := pv.project.UnSubscribeLogger(name, observer); err != nil {
			log.Err(err).Msgf("failed to unsubscribe from logs for process %s", name)
		}
	}
	pv.allLogsObservers = make(map[string]*AllLogsObserver)
	pv.logsText.Flush()
}

// truncateAllLogs truncates logs for all processes.
func (pv *pcView) truncateAllLogs() {
	for _, name := range pv.procNames {
		if err := pv.project.TruncateProcessLogs(name); err != nil {
			log.Err(err).Msgf("failed to truncate process %s logs", name)
		}
	}
}
