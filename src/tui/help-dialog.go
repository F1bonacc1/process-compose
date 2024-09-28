package tui

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	logs      = "Logs"
	processes = "Processes"
)

type helpDialog struct {
	*tview.Grid
	table       *tview.Table
	closeButton *tview.Button
}

func newHelpDialog(shortcuts *ShortCuts, closeFn func()) *helpDialog {
	dialog := &helpDialog{
		Grid:        tview.NewGrid().SetBorders(true).SetRows(30, 1),
		table:       createHelpTable(shortcuts, closeFn),
		closeButton: tview.NewButton("Close").SetSelectedFunc(closeFn),
	}
	dialog.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			closeFn()
			return nil
		}
		return event
	})
	dialog.AddItem(dialog.table, 0, 0, 1, 1, 0, 0, true).
		AddItem(dialog.closeButton, 1, 0, 1, 1, 0, 0, false)
	return dialog
}

func (hd *helpDialog) StylesChanged(s *config.Styles) {
	hd.SetBackgroundColor(s.BgColor())
	hd.SetBordersColor(s.BorderColor())
	hd.table.SetBackgroundColor(s.BgColor())
	hd.table.SetBorderColor(s.BorderColor())
	hd.table.SetTitleColor(s.Body().SecondaryTextColor.Color())
	hd.closeButton.SetLabelColor(s.Dialog().ButtonFgColor.Color())
	hd.closeButton.SetBackgroundColor(s.Dialog().ButtonBgColor.Color())

	hd.closeButton.SetLabelColorActivated(s.Dialog().ButtonFgColor.Color())
	hd.closeButton.SetBackgroundColorActivated(s.Dialog().ButtonBgColor.Color())
	for r := 0; r < hd.table.GetRowCount(); r++ {
		for c := 0; c < hd.table.GetColumnCount(); c++ {
			if c == 1 {
				hd.table.GetCell(r, c).SetTextColor(s.FgColor())
			} else if c == 0 {
				cell := hd.table.GetCell(r, c)
				if cell.Text == logs || cell.Text == processes {
					cell.SetTextColor(s.Body().TertiaryTextColor.Color())
				} else {
					cell.SetTextColor(s.Dialog().LabelFgColor.Color())
				}
			}
		}
	}
}

func createHelpTable(shortcuts *ShortCuts, fn func()) *tview.Table {
	table := tview.NewTable().SetBorders(false).SetSelectable(true, false)

	row := 0
	//GENERAL
	for _, act := range generalActionsOrder {
		action := shortcuts.ShortCutKeys[act]
		table.SetCell(row, 0, tview.NewTableCell(action.ShortCut).SetSelectable(false))
		if len(action.Description) > 0 {
			table.SetCell(row, 1, tview.NewTableCell(action.Description).SetSelectable(true).SetExpansion(1).SetReference(act))
		} else {
			td := fmt.Sprintf("%s/%s", action.ToggleDescription[true], action.ToggleDescription[false])
			table.SetCell(row, 1, tview.NewTableCell(td).SetSelectable(true).SetExpansion(1).SetReference(act))
		}
		row++
	}

	//LOGS
	row++
	table.SetCell(row, 0, tview.NewTableCell(logs).SetSelectable(false))
	row++
	for _, act := range logActionsOrder {
		if act == ActionLogSelection && !config.IsLogSelectionOn() {
			continue
		}
		action := shortcuts.ShortCutKeys[act]
		table.SetCell(row, 0, tview.NewTableCell(action.ShortCut).SetSelectable(false))
		if len(action.Description) > 0 {
			table.SetCell(row, 1, tview.NewTableCell(action.Description).SetSelectable(true).SetExpansion(1).SetReference(act))
		} else {
			td := fmt.Sprintf("%s/%s", action.ToggleDescription[true], action.ToggleDescription[false])
			table.SetCell(row, 1, tview.NewTableCell(td).SetSelectable(true).SetExpansion(1).SetReference(act))
		}
		row++
	}

	//PROCESSES
	table.SetCell(row, 0, tview.NewTableCell(processes).SetSelectable(false))
	row++
	for _, act := range procActionsOrder {
		action := shortcuts.ShortCutKeys[act]
		table.SetCell(row, 0, tview.NewTableCell(action.ShortCut).SetSelectable(false))
		if len(action.Description) > 0 {
			table.SetCell(row, 1, tview.NewTableCell(action.Description).SetSelectable(true).SetExpansion(1).SetReference(act))
		} else {
			td := fmt.Sprintf("%s/%s", action.ToggleDescription[true], action.ToggleDescription[false])
			table.SetCell(row, 1, tview.NewTableCell(td).SetSelectable(true).SetExpansion(1).SetReference(act))
		}
		row++
	}

	table.SetBorder(true).SetTitle("Shortcuts")
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			act := table.GetCell(table.GetSelection()).GetReference().(ActionName)
			fn()
			shortcuts.ShortCutKeys[act].actionFn()
		}
		return event
	})

	return table
}
