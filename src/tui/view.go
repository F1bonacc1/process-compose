package tui

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type pcView struct {
	procTable  *tview.Table
	statTable  *tview.Table
	appView    *tview.Application
	logsText   *LogView
	statusText *tview.TextView
	helpText   *tview.TextView
	procNames  []string
	version    string
	logFollow  bool
	loggedProc string
}

func newPcView(version string, logLength int) *pcView {
	pv := &pcView{
		appView:    tview.NewApplication(),
		logsText:   NewLogView(logLength),
		statusText: tview.NewTextView().SetDynamicColors(true),
		procNames:  app.PROJ.GetLexicographicProcessNames(),
		version:    version,
		logFollow:  true,
		helpText:   tview.NewTextView().SetDynamicColors(true),
		loggedProc: "",
	}
	pv.procTable = pv.createProcTable()
	pv.statTable = pv.createStatTable()
	pv.updateHelpTextView()
	pv.appView.SetRoot(pv.createGrid(), true).EnableMouse(true).SetInputCapture(pv.onAppKey)
	if len(pv.procNames) > 0 {
		name := pv.procNames[0]
		pv.logsText.SetTitle(name)
		pv.followLog(name)
	}
	return pv
}

func (pv *pcView) onAppKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF10:
		pv.terminateAppView()
	case tcell.KeyF5:
		pv.logFollow = !pv.logFollow
		name := pv.getSelectedProcName()
		if pv.logFollow {
			pv.followLog(name)
			go pv.updateLogs()
		} else {
			pv.unFollowLog()
		}
		pv.updateHelpTextView()
	case tcell.KeyF6:
		pv.logsText.ToggleWrap()
		pv.updateHelpTextView()
	case tcell.KeyCtrlC:
		pv.terminateAppView()
	default:
		return event
	}
	return nil
}

func (pv *pcView) terminateAppView() {

	m := tview.NewModal().
		SetText("Are you sure you want to quit?\nThis will terminate all the running processes.").
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Quit" {
				go pv.handleShutDown()
			}
			pv.appView.SetRoot(pv.createGrid(), true)

		})
	// Display and focus the dialog
	pv.appView.SetRoot(m, false)
}

func (pv *pcView) handleShutDown() {
	pv.statTable.SetCell(0, 2, tview.NewTableCell("Shutting Down...").
		SetSelectable(false).
		SetAlign(tview.AlignRight).
		SetExpansion(0).
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorRed))
	app.PROJ.ShutDownProject()
	pv.appView.Stop()

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
		pv.procTable.SetCell(r+1, 0, tview.NewTableCell(strconv.Itoa(state.Pid)).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 1, tview.NewTableCell(state.Name).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 2, tview.NewTableCell(state.Status).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 3, tview.NewTableCell(state.SystemTime).SetAlign(tview.AlignLeft).SetExpansion(1).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 4, tview.NewTableCell(strconv.Itoa(state.Restarts)).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
		pv.procTable.SetCell(r+1, 5, tview.NewTableCell(strconv.Itoa(state.ExitCode)).SetAlign(tview.AlignRight).SetExpansion(0).SetTextColor(tcell.ColorLightSkyBlue))
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

func (pv *pcView) onTableSelectionChange(row, column int) {
	name := pv.getSelectedProcName()
	pv.logsText.SetBorder(true).SetTitle(name)
	pv.unFollowLog()
	pv.followLog(name)
	if !pv.logFollow {
		// call follow and unfollow to update the buffer and stop following
		// in case the following is disabled
		pv.unFollowLog()
	}
}

func (pv *pcView) followLog(name string) {
	pv.loggedProc = name
	pv.logsText.Clear()
	app.PROJ.GetLogsAndSubscribe(name, pv.logsText)
}

func (pv *pcView) unFollowLog() {
	if pv.loggedProc != "" {
		app.PROJ.UnSubscribeLogger(pv.loggedProc)
	}
	pv.logsText.Flush()
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
		"PID", "NAME", "STATUS", "AGE", "RESTARTS", "EXIT CODE",
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
	return table
}

func (pv *pcView) createStatTable() *tview.Table {
	table := tview.NewTable().SetBorders(false).SetSelectable(false, false)

	table.SetCell(0, 0, tview.NewTableCell("Version:").SetSelectable(false).SetTextColor(tcell.ColorYellow))
	table.SetCell(0, 1, tview.NewTableCell(pv.version).SetSelectable(false).SetExpansion(1))

	table.SetCell(1, 0, tview.NewTableCell("Hostname:").SetSelectable(false).SetTextColor(tcell.ColorYellow))
	hostname, err := os.Hostname()
	if err != nil {
		hostname = err.Error()
	}
	table.SetCell(1, 1, tview.NewTableCell(hostname).SetSelectable(false).SetExpansion(1))

	table.SetCell(2, 0, tview.NewTableCell("Processes:").SetSelectable(false).SetTextColor(tcell.ColorYellow))
	table.SetCell(2, 1, tview.NewTableCell(strconv.Itoa(len(pv.procNames))).SetSelectable(false).SetExpansion(1))
	table.SetCell(0, 2, tview.NewTableCell("").SetSelectable(false))

	table.SetCell(0, 3, tview.NewTableCell("🔥 Process Compose").
		SetSelectable(false).
		SetAlign(tview.AlignRight).
		SetExpansion(1).
		SetTextColor(tcell.ColorYellow))
	return table
}

func (pv *pcView) updateHelpTextView() {
	wrap := "Wrap On"
	if pv.logsText.IsWrapOn() {
		wrap = "Wrap Off"
	}
	follow := "Follow On"
	if pv.logFollow {
		follow = "Follow Off"
	}
	pv.helpText.Clear()
	fmt.Fprintf(pv.helpText, "%s ", "[lightskyblue:]LOGS:[-:-:-]")
	fmt.Fprintf(pv.helpText, "%s%s%s ", "F5[black:green]", follow, "[-:-:-]")
	fmt.Fprintf(pv.helpText, "%s%s%s ", "F6[black:green]", wrap, "[-:-:-]")
	fmt.Fprintf(pv.helpText, "%s ", "[lightskyblue::b]PROCESS:[-:-:-]")
	fmt.Fprintf(pv.helpText, "%s ", "F7[black:green]Start[-:-:-]")
	fmt.Fprintf(pv.helpText, "%s ", "F9[black:green]Kill[-:-:-]")
	fmt.Fprintf(pv.helpText, "%s ", "F10[black:green]Quit[-:-:-]")
}

func (pv pcView) createGrid() *tview.Grid {
	grid := tview.NewGrid().
		SetRows(3, 0, 0, 1).
		//SetColumns(30, 0, 30).
		SetBorders(true).
		AddItem(pv.statTable, 0, 0, 1, 1, 0, 0, false).
		AddItem(pv.procTable, 1, 0, 1, 1, 0, 0, true).
		AddItem(pv.logsText, 2, 0, 1, 1, 0, 0, false).
		AddItem(pv.helpText, 3, 0, 1, 1, 0, 0, false)

	grid.SetTitle("Process Compose")
	return grid
}

func (pv *pcView) updateTable() {
	for {
		time.Sleep(1000 * time.Millisecond)
		pv.appView.QueueUpdateDraw(func() {
			pv.fillTableData()
		})
	}
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

func SetupTui(version string, logLength int) {
	pv := newPcView(version, logLength)

	go pv.updateTable()
	go pv.updateLogs()

	if err := pv.appView.Run(); err != nil {
		panic(err)
	}
}
