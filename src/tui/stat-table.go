package tui

import (
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
	"strconv"
)

func (pv *pcView) createStatTable() *tview.Table {
	table := tview.NewTable().SetBorders(false).SetSelectable(false, false)

	table.SetCell(0, 0, tview.NewTableCell("Version:").SetSelectable(false).SetTextColor(tcell.ColorYellow))
	table.SetCell(0, 1, tview.NewTableCell(config.Version).SetSelectable(false).SetExpansion(1))

	table.SetCell(1, 0, tview.NewTableCell("Hostname:").SetSelectable(false).SetTextColor(tcell.ColorYellow))
	hostname, err := os.Hostname()
	if err != nil {
		hostname = err.Error()
	}
	table.SetCell(1, 1, tview.NewTableCell(hostname).SetSelectable(false).SetExpansion(1))

	table.SetCell(2, 0, tview.NewTableCell("Processes:").SetSelectable(false).SetTextColor(tcell.ColorYellow))
	pv.procCountCell = tview.NewTableCell(strconv.Itoa(len(pv.procNames))).SetSelectable(false).SetExpansion(1)
	table.SetCell(2, 1, pv.procCountCell)
	table.SetCell(0, 2, tview.NewTableCell("").SetSelectable(false).SetExpansion(0))

	table.SetCell(0, 3, tview.NewTableCell("Process Compose 🔥").
		SetSelectable(false).
		SetAlign(tview.AlignRight).
		SetExpansion(1).
		SetTextColor(tcell.ColorYellow))
	return table
}
