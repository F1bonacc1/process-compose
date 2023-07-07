package tui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
	"strconv"
	"time"
)

func (pv *pcView) fillTableData() {
	if pv.project == nil {
		return
	}
	runningProcCount := 0
	states, err := pv.project.GetProcessesState()
	if err != nil {
		log.Err(err).Msg("failed to get processes state")
		return
	}
	sorter := pv.getTableSorter()
	err = sortProcessesState(sorter.sortByColumn, sorter.isAsc, states)
	if err != nil {
		log.Err(err).Msg("failed to sort states")
		return
	}
	for r, state := range states.States {
		pv.procTable.SetCell(r+1, 0, tview.NewTableCell(strconv.Itoa(state.Pid)).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 1, tview.NewTableCell(state.Name).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 2, tview.NewTableCell(state.Namespace).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 3, tview.NewTableCell(state.Status).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 4, tview.NewTableCell(state.SystemTime).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 5, tview.NewTableCell(state.Health).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 6, tview.NewTableCell(strconv.Itoa(state.Restarts)).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 7, tview.NewTableCell(strconv.Itoa(state.ExitCode)).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
		if state.IsRunning {
			runningProcCount += 1
		}
	}
	// remove unnecessary rows, don't forget the title row (-1)
	if pv.procTable.GetRowCount()-1 > len(states.States) {
		for i := len(states.States); i < pv.procTable.GetRowCount()-1; i++ {
			pv.procTable.RemoveRow(i)
		}
	}
	if pv.procCountCell != nil {
		pv.procCountCell.SetText(fmt.Sprintf("%d/%d", runningProcCount, len(pv.procNames)))
	}
}

func (pv *pcView) onTableSelectionChange(_, _ int) {
	name := pv.getSelectedProcName()
	if len(name) == 0 {
		return
	}
	pv.logsText.resetSearch()
	pv.updateHelpTextView()
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
	pv.procColumns = map[ColumnID]string{
		ProcessStatePid:       "PID(P)",
		ProcessStateName:      "NAME(N)",
		ProcessStateNamespace: "NAMESPACE(C)",
		ProcessStateStatus:    "STATUS(S)",
		ProcessStateAge:       "AGE(A)",
		ProcessStateHealth:    "HEALTH(H)",
		ProcessStateRestarts:  "RESTARTS(R)",
		ProcessStateExit:      "EXIT CODE(E)",
	}

	table.Select(1, 1).SetFixed(1, 0).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case pv.shortcuts.ShortCutKeys[ActionProcessStop].key:
			name := pv.getSelectedProcName()
			pv.project.StopProcess(name)
		case pv.shortcuts.ShortCutKeys[ActionProcessStart].key:
			name := pv.getSelectedProcName()
			pv.project.StartProcess(name)
		case pv.shortcuts.ShortCutKeys[ActionProcessRestart].key:
			name := pv.getSelectedProcName()
			pv.project.RestartProcess(name)
		case tcell.KeyRune:
			if event.Rune() == 'S' {
				pv.setTableSorter(ProcessStateStatus)
			} else if event.Rune() == 'N' {
				pv.setTableSorter(ProcessStateName)
			} else if event.Rune() == 'C' {
				pv.setTableSorter(ProcessStateNamespace)
			} else if event.Rune() == 'A' {
				pv.setTableSorter(ProcessStateAge)
			} else if event.Rune() == 'H' {
				pv.setTableSorter(ProcessStateHealth)
			} else if event.Rune() == 'R' {
				pv.setTableSorter(ProcessStateRestarts)
			} else if event.Rune() == 'E' {
				pv.setTableSorter(ProcessStateExit)
			} else if event.Rune() == 'P' {
				pv.setTableSorter(ProcessStatePid)
			}
		}
		return event
	})
	for i := 0; i < len(pv.procColumns); i++ {
		expansion := 10
		align := tview.AlignLeft
		switch ColumnID(i) {
		case
			ProcessStatePid:
			expansion = 1
		case
			ProcessStateRestarts,
			ProcessStateExit:
			align = tview.AlignRight
		}

		table.SetCell(0, i, tview.NewTableCell(pv.procColumns[ColumnID(i)]).
			SetSelectable(false).SetExpansion(expansion).SetAlign(align))
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

func (pv *pcView) setTableSorter(sortBy ColumnID) {
	pv.sortMtx.Lock()
	defer pv.sortMtx.Unlock()
	prevSortColumn := ProcessStateUndefined
	if pv.stateSorter.sortByColumn == sortBy {
		pv.stateSorter.isAsc = !pv.stateSorter.isAsc
	} else {
		prevSortColumn = pv.stateSorter.sortByColumn
		pv.stateSorter.sortByColumn = sortBy
		pv.stateSorter.isAsc = true
	}
	order := "[pink]↓[-:-:-]"
	if !pv.stateSorter.isAsc {
		order = "[pink]↑[-:-:-]"
	}
	pv.procTable.GetCell(0, int(sortBy)).SetText(pv.procColumns[sortBy] + order)
	if prevSortColumn != ProcessStateUndefined {
		pv.procTable.GetCell(0, int(prevSortColumn)).SetText(pv.procColumns[prevSortColumn])
	}
}

func (pv *pcView) getTableSorter() StateSorter {
	pv.sortMtx.Lock()
	defer pv.sortMtx.Unlock()
	return pv.stateSorter
}
