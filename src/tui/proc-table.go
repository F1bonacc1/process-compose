package tui

import (
	"context"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
	"strconv"
	"time"
)

type tableRowValues struct {
	icon      string
	iconColor tcell.Color
	pid       string
	name      string
	ns        string
	status    string
	age       string
	health    string
	restarts  string
	exitCode  string
}

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
	row := 1
	for _, state := range states.States {
		if !pv.isNsSelected(state.Namespace) {
			pv.procTable.RemoveRow(row)
			continue
		}
		rowVals := getTableRowValues(state)
		setRowValues(pv.procTable, row, rowVals)
		if state.IsRunning {
			runningProcCount += 1
		}
		row += 1
	}

	// remove unnecessary rows, don't forget the title row (-1)
	if pv.procTable.GetRowCount()-1 > row {
		for i := len(states.States); i < pv.procTable.GetRowCount()-1; i++ {
			pv.procTable.RemoveRow(i)
		}
	}
	if pv.selectedNsChanged.Swap(false) {
		pv.procTable.Select(1, 1)
	}

	if pv.procCountCell != nil {
		nsLbl := ""
		if !pv.isNsSelected(AllNS) {
			nsLbl = " (" + pv.getSelectedNs() + ")"
		}
		pv.procCountCell.SetText(fmt.Sprintf("%d/%d%s", runningProcCount, len(pv.procNames), nsLbl))
	}
}

func setRowValues(procTable *tview.Table, row int, rowVals tableRowValues) {
	procTable.SetCell(row, int(ProcessStateIcon), tview.NewTableCell(rowVals.icon).SetAlign(tview.AlignCenter).SetExpansion(0).SetTextColor(rowVals.iconColor))
	procTable.SetCell(row, int(ProcessStatePid), tview.NewTableCell(rowVals.pid).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
	procTable.SetCell(row, int(ProcessStateName), tview.NewTableCell(rowVals.name).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
	procTable.SetCell(row, int(ProcessStateNamespace), tview.NewTableCell(rowVals.ns).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
	procTable.SetCell(row, int(ProcessStateStatus), tview.NewTableCell(rowVals.status).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
	procTable.SetCell(row, int(ProcessStateAge), tview.NewTableCell(rowVals.age).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
	procTable.SetCell(row, int(ProcessStateHealth), tview.NewTableCell(rowVals.health).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
	procTable.SetCell(row, int(ProcessStateRestarts), tview.NewTableCell(rowVals.restarts).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
	procTable.SetCell(row, int(ProcessStateExit), tview.NewTableCell(rowVals.exitCode).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
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
		ProcessStateIcon:      "●",
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
			pv.startProcess()
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
			ProcessStateIcon:
			expansion = 1
			align = tview.AlignCenter
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

func (pv *pcView) updateTable(ctx context.Context) {
	pv.appView.QueueUpdateDraw(func() {
		pv.fillTableData()
	})
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("Table monitoring canceled")
			return
		case <-time.After(pv.refreshRate):
			pv.appView.QueueUpdateDraw(func() {
				pv.fillTableData()
			})
		}
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

func getIconForState(state types.ProcessState) (string, tcell.Color) {
	switch state.Status {
	case types.ProcessStateRunning,
		types.ProcessStateLaunching,
		types.ProcessStateLaunched:
		if state.Health == types.ProcessHealthNotReady {
			return "●", tcell.ColorYellow
		}
		return "●", tcell.ColorLightSkyBlue
	case types.ProcessStatePending,
		types.ProcessStateRestarting:
		return "●", tcell.ColorGrey
	case types.ProcessStateCompleted:
		if state.ExitCode == 0 {
			return "●", tcell.ColorLightGreen
		}
		return "●", tcell.ColorRed
	case types.ProcessStateError:
		return "●", tcell.ColorRed
	case types.ProcessStateDisabled,
		types.ProcessStateForeground:
		return "◯", tcell.ColorGrey
	case types.ProcessStateSkipped:
		return "●", tcell.ColorYellow
	default:
		return "●", tcell.ColorLightSkyBlue
	}
}

func getStrForRestarts(restarts int) string {
	if restarts == 0 {
		return types.PlaceHolderValue
	}
	return strconv.Itoa(restarts)
}

func getStrForExitCode(state types.ProcessState) string {
	// running no exit info yet
	if state.IsRunning && state.ExitCode == 0 {
		return types.PlaceHolderValue
	}
	// disabled foreground or pending state
	if state.Status == types.ProcessStateDisabled ||
		state.Status == types.ProcessStatePending ||
		state.Status == types.ProcessStateForeground {
		return types.PlaceHolderValue
	}
	return strconv.Itoa(state.ExitCode)
}

func getTableRowValues(state types.ProcessState) tableRowValues {
	icon, color := getIconForState(state)
	return tableRowValues{
		icon:      icon,
		iconColor: color,
		pid:       strconv.Itoa(state.Pid),
		name:      state.Name,
		ns:        state.Namespace,
		status:    state.Status,
		age:       state.SystemTime,
		health:    state.Health,
		restarts:  getStrForRestarts(state.Restarts),
		exitCode:  getStrForExitCode(state),
	}
}
