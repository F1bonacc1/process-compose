package tui

import (
	"context"
	"time"

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
	if !pv.isCommandModeDisabled() {
		textInput := pv.getCommandInput()
		pv.mainGrid.AddItem(textInput, row, 0, 1, 1, 0, 0, true)
		pv.appView.SetFocus(textInput)
		row++
	}
	var logPrimitive tview.Primitive
	if !pv.logSelect {
		logPrimitive = pv.logsText
	} else {
		logPrimitive = pv.logsTextArea
	}

	name := pv.getSelectedProcName()
	pv.currentProcInteractive = false
	if name != "" && pv.isInteractive(name) {
		pv.currentProcInteractive = true
		logPrimitive = pv.termView
		// Ensure PTY is set
		pty := pv.project.GetProcessPty(name)
		pv.termView.SetPty(pty)
	} else {
		pv.termView.Stop()
	}

	switch pv.scrSplitState {
	case LogFull:
		pv.mainGrid.AddItem(logPrimitive, row, 0, 1, 1, 0, 0, true)
		row++
	case ProcFull:
		pv.mainGrid.AddItem(pv.procTable, row, 0, 1, 1, 0, 0, pv.isCommandModeDisabled())
		row++
	case LogProcHalf:
		pv.mainGrid.AddItem(pv.procTable, row, 0, 1, 1, 0, 0, pv.isCommandModeDisabled())
		row++
		pv.mainGrid.AddItem(logPrimitive, row, 0, 1, 1, 0, 0, false)
		row++
	}
	if !pv.isFullScreen {
		pv.mainGrid.AddItem(pv.helpFooter, row, 0, 1, 1, 0, 0, false)
	}
	pv.appView.SetFocus(pv.mainGrid)
	pv.autoAdjustProcTableHeight()
}

func (pv *pcView) isCommandModeDisabled() bool {
	return pv.commandModeType == commandModeOff || pv.commandModeType == commandModeDisabled
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
	if !pv.isCommandModeDisabled() {
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

func (pv *pcView) getCommandInput() tview.Primitive {
	switch pv.commandModeType {
	case commandModeSearch:
		return pv.getSearchInput()
	case commandModePassword:
		return pv.getPassInput()
	default:
		return nil
	}
}

func (pv *pcView) getSearchInput() tview.Primitive {
	textInput := tview.NewInputField().SetLabel("Search:")
	textInput.SetFieldBackgroundColor(pv.styles.Dialog().FieldBgColor.Color())
	textInput.SetFieldTextColor(pv.styles.Dialog().FieldFgColor.Color())
	textInput.SetLabelColor(pv.styles.Dialog().LabelFgColor.Color())
	textInput.SetLabelStyle(textInput.GetLabelStyle().Background(pv.styles.BgColor()))

	textInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter || key == tcell.KeyEsc {
			pv.commandModeType = commandModeOff
			pv.appView.SetFocus(pv.procTable)
			pv.redrawGrid()
		}
	})
	textInput.SetChangedFunc(func(text string) {
		_ = pv.searchProcess(text)
	})
	return textInput
}

func (pv *pcView) getPassInput() tview.Primitive {
	textInput := tview.NewInputField().SetLabel("Enter Password:")
	textInput.SetFieldBackgroundColor(pv.styles.Dialog().FieldBgColor.Color())
	textInput.SetFieldTextColor(pv.styles.Dialog().FieldFgColor.Color())
	textInput.SetLabelColor(pv.styles.Dialog().LabelFgColor.Color())
	textInput.SetLabelStyle(textInput.GetLabelStyle().Background(pv.styles.BgColor()))
	textInput.SetMaskCharacter('●')
	textInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {

			go pv.handlePassEntered(textInput)
		}
		if key == tcell.KeyEsc {
			pv.commandModeType = commandModeDisabled
			pv.appView.SetFocus(pv.procTable)
			pv.redrawGrid()
		}
	})

	return textInput
}

func (pv *pcView) handlePassEntered(textInput *tview.InputField) {
	pass := textInput.GetText()
	name := pv.getSelectedProcName()
	ctx, cancel := context.WithCancel(context.Background())
	pv.showAutoProgress(ctx, time.Second*1)
	err := pv.project.SetProcessPassword(name, pass)
	cancel()
	if err != nil {
		pv.attentionMessage(err.Error(), 3*time.Second)
		return
	}
	pv.commandModeType = commandModeOff
	pv.appView.SetFocus(pv.procTable)
	pv.redrawGrid()
}

func (pv *pcView) redrawGrid() {
	go pv.appView.QueueUpdateDraw(func() {
		pv.mainGrid.Clear()
		pv.createGrid()
	})
}
