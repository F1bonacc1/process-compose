package tui

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (pv *pcView) showHelpDialog() {
	helper := pv.createHelpPrimitive()
	pv.showDialog(helper, 50, 30)
}

func (pv *pcView) createHelpPrimitive() tview.Primitive {
	table := tview.NewTable().SetBorders(false).SetSelectable(false, false)

	row := 0
	actionHelp := pv.shortcuts.ShortCutKeys[ActionHelp]
	table.SetCell(row, 0, tview.NewTableCell(actionHelp.ShortCut).SetSelectable(false).SetTextColor(tcell.ColorYellow))
	table.SetCell(row, 1, tview.NewTableCell(actionHelp.Description).SetSelectable(false).SetExpansion(1))

	//LOGS
	row++
	table.SetCell(row, 0, tview.NewTableCell("Logs").SetSelectable(false).SetTextColor(tcell.ColorGreen))
	row++
	for _, act := range logActionsOrder {
		if act == ActionLogSelection && !config.IsLogSelectionOn() {
			continue
		}
		action := pv.shortcuts.ShortCutKeys[act]
		table.SetCell(row, 0, tview.NewTableCell(action.ShortCut).SetSelectable(false).SetTextColor(tcell.ColorYellow))
		if len(action.Description) > 0 {
			table.SetCell(row, 1, tview.NewTableCell(action.Description).SetSelectable(false).SetExpansion(1))
		} else {
			td := fmt.Sprintf("%s/%s", action.ToggleDescription[true], action.ToggleDescription[false])
			table.SetCell(row, 1, tview.NewTableCell(td).SetSelectable(false).SetExpansion(1))
		}
		row++
	}

	//PROCESSES
	table.SetCell(row, 0, tview.NewTableCell("Processes").SetSelectable(false).SetTextColor(tcell.ColorGreen))
	row++
	for _, act := range procActionsOrder {
		action := pv.shortcuts.ShortCutKeys[act]
		table.SetCell(row, 0, tview.NewTableCell(action.ShortCut).SetSelectable(false).SetTextColor(tcell.ColorYellow))
		if len(action.Description) > 0 {
			table.SetCell(row, 1, tview.NewTableCell(action.Description).SetSelectable(false).SetExpansion(1))
		} else {
			td := fmt.Sprintf("%s/%s", action.ToggleDescription[true], action.ToggleDescription[false])
			table.SetCell(row, 1, tview.NewTableCell(td).SetSelectable(false).SetExpansion(1))
		}
		row++
	}

	table.SetBorder(true).SetTitle("Shortcuts")
	closeBtn := tview.NewButton("Close").SetSelectedFunc(func() {
		pv.pages.RemovePage(PageDialog)
	})

	grid := tview.NewGrid().
		SetBorders(true).
		SetRows(30, 1).
		AddItem(table, 0, 0, 1, 1, 0, 0, false).
		AddItem(closeBtn, 1, 0, 1, 1, 0, 0, true)
	return grid
}
