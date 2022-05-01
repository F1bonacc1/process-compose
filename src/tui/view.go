package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type pcView struct {
	procTable  *tview.Table
	statTable  *tview.Table
	appView    *tview.Application
	logsText   *tview.TextView
	statusText *tview.TextView
	procNames  []string
}

func newPcView() *pcView {
	pv := &pcView{
		appView:    tview.NewApplication(),
		logsText:   tview.NewTextView().SetDynamicColors(true).SetScrollable(true),
		statusText: tview.NewTextView().SetDynamicColors(true),
		procNames:  app.PROJ.GetLexicographicProcessNames(),
	}
	pv.procTable = pv.createProcTable()
	pv.statTable = pv.createStatTable()
	pv.appView.SetRoot(pv.createGrid(), true).EnableMouse(true).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyF10:
				pv.appView.Stop()
			}
			return event
		})
	if len(pv.procNames) > 0 {
		pv.logsText.SetTitle(pv.procNames[0])
	}
	return pv
}

func (pv *pcView) fillTableData() {
	if app.PROJ == nil {
		return
	}

	for r, name := range pv.procNames {
		state := app.PROJ.GetProcessState(name)
		if state == nil {
			return
		}
		pv.procTable.SetCell(r+1, 0, tview.NewTableCell(strconv.Itoa(state.Pid)).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 1, tview.NewTableCell(state.Name).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 2, tview.NewTableCell(state.Status).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 3, tview.NewTableCell(state.SystemTime).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 4, tview.NewTableCell(strconv.Itoa(state.Restarts)).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 5, tview.NewTableCell(strconv.Itoa(state.ExitCode)).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
	}
}

func (pv pcView) getSelectedProcName() string {
	if pv.procTable == nil {
		return ""
	}
	row, _ := pv.procTable.GetSelection()
	if row > 0 && row <= len(pv.procNames) {
		return pv.procNames[row-1]
	}
	return ""
}

func (pv *pcView) fillLogs() {
	name := pv.getSelectedProcName()
	logs, err := app.PROJ.GetProcessLog(name, 1000, 0)
	if err != nil {
		pv.logsText.SetBorder(true).SetTitle(err.Error())
		pv.logsText.Clear()
	} else {
		pv.logsText.SetText(strings.Join(logs, "\n"))
	}
}

func (pv *pcView) onTableSelectionChange(row, column int) {
	name := pv.getSelectedProcName()
	pv.logsText.SetBorder(true).SetTitle(name)
}

func (pv *pcView) createProcTable() *tview.Table {
	table := tview.NewTable().SetBorders(false).SetSelectable(true, false).SetSelectionChangedFunc(pv.onTableSelectionChange)
	//pv.fillTableData()
	table.Select(1, 1).SetFixed(1, 0).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF9:
			name := pv.getSelectedProcName()
			app.PROJ.StopProcess(name)
		case tcell.KeyF7:
			name := pv.getSelectedProcName()
			app.PROJ.StartProcess(name)
		}
		return event
	})
	columns := []string{
		"PID", "NAME", "STATUS", "TIME", "RESTARTS", "EXIT CODE",
	}
	for i := 0; i < len(columns); i++ {
		table.SetCell(0, i, tview.NewTableCell(columns[i]).
			SetSelectable(false).SetExpansion(1))
	}
	return table
}

func (pv *pcView) createStatTable() *tview.Table {
	table := tview.NewTable().SetBorders(false).SetSelectable(false, false)

	table.SetCell(0, 0, tview.NewTableCell("Version:").SetSelectable(false).SetTextColor(tcell.ColorYellow))
	table.SetCell(0, 1, tview.NewTableCell("v0.7.2").SetSelectable(false))

	table.SetCell(1, 0, tview.NewTableCell("Hostname:").SetSelectable(false).SetTextColor(tcell.ColorYellow))
	table.SetCell(1, 1, tview.NewTableCell("hhhppp").SetSelectable(false))

	table.SetCell(2, 0, tview.NewTableCell("Processes:").SetSelectable(false).SetTextColor(tcell.ColorYellow))
	table.SetCell(2, 1, tview.NewTableCell(strconv.Itoa(len(pv.procNames))).SetSelectable(false))

	table.SetCell(0, 3, tview.NewTableCell("🔥 Process Compose").
		SetSelectable(false).
		SetAlign(tview.AlignRight).
		SetExpansion(1).
		SetTextColor(tcell.ColorYellow))
	return table
}

func getHelpTextView() *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true)
	fmt.Fprintf(textView, "%s ", "F7[black:green]Start[-:-:-]")
	fmt.Fprintf(textView, "%s ", "F9[black:green]Kill[-:-:-]")
	fmt.Fprintf(textView, "%s ", "F10[black:green]Quit[-:-:-]")
	return textView
}

func (pv pcView) createGrid() *tview.Grid {
	grid := tview.NewGrid().
		SetRows(3, 0, 0, 1).
		//SetColumns(30, 0, 30).
		SetBorders(true).
		AddItem(pv.statTable, 0, 0, 1, 1, 0, 0, false).
		AddItem(pv.procTable, 1, 0, 1, 1, 0, 0, true).
		AddItem(pv.logsText, 2, 0, 1, 1, 0, 0, false).
		AddItem(getHelpTextView(), 3, 0, 1, 1, 0, 0, false)

	grid.SetTitle("Process Compose")
	return grid
}

func (pv *pcView) updateTable() {
	for {
		time.Sleep(1000 * time.Millisecond)
		pv.appView.QueueUpdateDraw(func() {
			pv.fillTableData()
			pv.fillLogs()
		})
	}
}

func SetupTui() {
	pv := newPcView()

	go pv.updateTable()

	if err := pv.appView.Run(); err != nil {
		panic(err)
	}
}
