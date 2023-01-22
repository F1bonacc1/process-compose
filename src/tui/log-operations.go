package tui

import (
	"github.com/f1bonacc1/glippy"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/gdamore/tcell/v2"
	"time"
)

func (pv *pcView) toggleLogSelection() {
	name := pv.getSelectedProcName()
	pv.logSelect = !pv.logSelect
	if pv.logSelect {
		pv.logsTextArea.SetText(pv.logsText.GetText(true), true).
			SetBorder(true).
			SetTitle(name + " [Select & Press Enter to Copy]")
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
	pv.logFollow = true
	pv.followLog(name)
	go pv.updateLogs()
	pv.updateHelpTextView()
}

func (pv *pcView) stopFollowLog() {
	pv.logFollow = false
	pv.unFollowLog()
	pv.updateHelpTextView()
}

func (pv *pcView) followLog(name string) {
	pv.loggedProc = name
	pv.logsText.Clear()
	_ = app.PROJ.GetProject().WithProcesses([]string{name}, func(process types.ProcessConfig) error {
		pv.logsText.useAnsi = !process.DisableAnsiColors
		return nil
	})
	app.PROJ.GetLogsAndSubscribe(name, pv.logsText)
}

func (pv *pcView) unFollowLog() {
	if pv.loggedProc != "" {
		app.PROJ.UnSubscribeLogger(pv.loggedProc)
	}
	pv.logsText.Flush()
}

func (pv *pcView) updateLogs() {
	for {
		pv.appView.QueueUpdateDraw(func() {
			pv.logsText.Flush()
		})
		if !pv.logFollow {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func (pv *pcView) createLogSelectionTextArea() {
	pv.logsTextArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCR:
			text, start, _ := pv.logsTextArea.GetSelection()
			glippy.Set(text)
			pv.logsTextArea.Select(start, start)
		case tcell.KeyEsc:
			pv.toggleLogSelection()
			pv.updateHelpTextView()
		}
		return nil
	})
}
