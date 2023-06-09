package tui

import (
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
	"strconv"
)

func (pv *pcView) createStatTable() *tview.Table {
	table := tview.NewTable().SetBorders(false).SetSelectable(false, false)

	table.SetCell(0, 0, tview.NewTableCell("Version:").SetSelectable(false).SetTextColor(tcell.ColorYellow))
	table.SetCell(0, 1, tview.NewTableCell(config.Version).SetSelectable(false).SetExpansion(1))

	table.SetCell(1, 0, tview.NewTableCell(pv.getHostNameTitle()).SetSelectable(false).SetTextColor(tcell.ColorYellow))
	hostname := pv.getHostName()
	table.SetCell(1, 1, tview.NewTableCell(hostname).SetSelectable(false).SetExpansion(1))

	table.SetCell(2, 0, tview.NewTableCell("Processes:").SetSelectable(false).SetTextColor(tcell.ColorYellow))
	pv.procCountCell = tview.NewTableCell(strconv.Itoa(len(pv.procNames))).SetSelectable(false).SetExpansion(1)
	table.SetCell(2, 1, pv.procCountCell)
	table.SetCell(0, 2, tview.NewTableCell("").SetSelectable(false).SetExpansion(0))

	table.SetCell(0, 3, tview.NewTableCell(pv.getPcTitle()).
		SetSelectable(false).
		SetAlign(tview.AlignRight).
		SetExpansion(1).
		SetTextColor(tcell.ColorYellow))
	return table
}

func (pv *pcView) getPcTitle() string {
	if pv.project.IsRemote() {
		return "Process Compose ⚡"
	} else {
		return "Process Compose 🔥"
	}
}

func (pv *pcView) getHostName() string {
	name, err := pv.project.GetHostName()
	if err != nil {
		log.Err(err).Msg("Unable to retrieve hostname")
		return "Unknown"
	}
	return name
}

func (pv *pcView) getHostNameTitle() string {
	if pv.project.IsRemote() {
		return "Server Name:"
	} else {
		return "Hostname:"
	}
}
