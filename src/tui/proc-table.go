package tui

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strconv"
	"time"
)

func (pv *pcView) fillTableData() {
	if app.PROJ == nil {
		return
	}
	runningProcCount := 0
	for r, name := range pv.procNames {
		state := app.PROJ.GetProcessState(name)
		if state == nil {
			return
		}
		pv.procTable.SetCell(r+1, 0, tview.NewTableCell(strconv.Itoa(state.Pid)).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 1, tview.NewTableCell(state.Name).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 2, tview.NewTableCell(state.Status).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 3, tview.NewTableCell(state.SystemTime).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 4, tview.NewTableCell(state.Health).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 5, tview.NewTableCell(strconv.Itoa(state.Restarts)).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 6, tview.NewTableCell(strconv.Itoa(state.ExitCode)).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
		if state.IsRunning {
			runningProcCount += 1
		}
	}
	if pv.procCountCell != nil {
		pv.procCountCell.SetText(fmt.Sprintf("%d/%d", runningProcCount, len(pv.procNames)))
	}
}

func (pv *pcView) onTableSelectionChange(row, column int) {
	name := pv.getSelectedProcName()
	if len(name) == 0 {
		return
	}
	pv.logsText.SetBorder(true).SetTitle(name)
	pv.unFollowLog()
	pv.followLog(name)
	if !pv.logFollow {
		// call follow and unfollow to update the buffer and stop following
		// in case the following is disabled
		pv.unFollowLog()
	}
}

func (pv *pcView) createProcTable() *tview.Table {
	table := tview.NewTable().SetBorders(false).SetSelectable(true, false)
	//pv.fillTableData()
	table.Select(1, 1).SetFixed(1, 0).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case pv.shortcuts.ShortCutKeys[ActionProcessStop].key:
			name := pv.getSelectedProcName()
			app.PROJ.StopProcess(name)
		case pv.shortcuts.ShortCutKeys[ActionProcessStart].key:
			name := pv.getSelectedProcName()
			app.PROJ.StartProcess(name)
		case pv.shortcuts.ShortCutKeys[ActionProcessRestart].key:
			name := pv.getSelectedProcName()
			app.PROJ.RestartProcess(name)
		}
		return event
	})
	columns := []string{
		"PID", "NAME", "STATUS", "AGE", "READINESS", "RESTARTS", "EXIT CODE",
	}
	for i := 0; i < len(columns); i++ {
		expan := 1
		align := tview.AlignLeft
		switch columns[i] {
		case
			"PID":
			expan = 0
		case
			"RESTARTS",
			"EXIT CODE":
			align = tview.AlignRight
		}

		table.SetCell(0, i, tview.NewTableCell(columns[i]).
			SetSelectable(false).SetExpansion(expan).SetAlign(align))
	}
	table.SetSelectionChangedFunc(pv.onTableSelectionChange)
	return table
}

func (pv *pcView) updateTable() {
	for {
		time.Sleep(1000 * time.Millisecond)
		pv.appView.QueueUpdateDraw(func() {
			pv.fillTableData()
		})
	}
}
