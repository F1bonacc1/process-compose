package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (pv *pcView) createGrid() {
	pv.mainGrid.SetBorders(true)
	row := 0
	if !pv.isFullScreen {
		pv.mainGrid.AddItem(pv.statTable, row, 0, 1, 1, 0, 0, false)
		row++
	}
	if pv.commandMode {
		textInput := pv.getSearchInput()
		pv.mainGrid.AddItem(textInput, row, 0, 1, 1, 0, 0, true)
		pv.appView.SetFocus(textInput)
		row++
	}
	var log tview.Primitive
	if !pv.logSelect {
		log = pv.logsText
	} else {
		log = pv.logsTextArea
	}
	switch pv.scrSplitState {
	case LogFull:
		pv.mainGrid.AddItem(log, row, 0, 1, 1, 0, 0, true)
		row++
	case ProcFull:
		pv.mainGrid.AddItem(pv.procTable, row, 0, 1, 1, 0, 0, !pv.commandMode)
		row++
	case LogProcHalf:
		pv.mainGrid.AddItem(pv.procTable, row, 0, 1, 1, 0, 0, !pv.commandMode)
		row++
		pv.mainGrid.AddItem(log, row, 0, 1, 1, 0, 0, false)
		row++
	}
	if !pv.isFullScreen {
		pv.mainGrid.AddItem(pv.helpText, row, 0, 1, 1, 0, 0, false)
	}
	pv.autoAdjustProcTableHeight()
}

func (pv *pcView) autoAdjustProcTableHeight() {
	maxProcHeight := pv.getMaxProcHeight()
	procTblHeight := pv.procTable.GetRowCount() + 1
	if procTblHeight > maxProcHeight {
		procTblHeight = maxProcHeight
	}
	rows := []int{}
	if !pv.isFullScreen {
		//stat table
		rows = append(rows, pv.statTable.GetRowCount())
	}
	if pv.commandMode {
		//search row
		rows = append(rows, 1)
	}
	if pv.scrSplitState == LogProcHalf {
		rows = append(rows, procTblHeight, 0)
	} else {
		// full proc or full log
		rows = append(rows, 0)
	}

	if !pv.isFullScreen {
		//help row
		rows = append(rows, 1)
	}
	//stat table, (command), processes table, logs, help text
	//0 means to take all the available height
	pv.mainGrid.SetRows(rows...)
}

func (pv *pcView) getSearchInput() tview.Primitive {
	textInput := tview.NewInputField().SetLabel("Search:")
	textInput.SetFieldBackgroundColor(pv.styles.Dialog().FieldBgColor.Color())
	textInput.SetFieldTextColor(pv.styles.Dialog().FieldFgColor.Color())
	textInput.SetLabelColor(pv.styles.Dialog().LabelFgColor.Color())
	textInput.SetLabelStyle(textInput.GetLabelStyle().Background(pv.styles.BgColor()))

	textInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter || key == tcell.KeyEsc {
			pv.commandMode = false
			pv.appView.SetFocus(pv.procTable)
			pv.redrawGrid()
		}
	})
	textInput.SetChangedFunc(func(text string) {
		_ = pv.searchProcess(text, false, false)
	})
	return textInput
}

func (pv *pcView) redrawGrid() {
	go pv.appView.QueueUpdateDraw(func() {
		pv.mainGrid.Clear()
		pv.createGrid()
	})
}
