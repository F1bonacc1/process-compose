package tui

import "github.com/rivo/tview"

func (pv *pcView) createGrid() {
	pv.mainGrid.Clear().
		SetRows(3, 0, 0, 1).
		//SetColumns(30, 0, 30).
		SetBorders(true).
		AddItem(pv.statTable, 0, 0, 1, 1, 0, 0, false).
		AddItem(pv.helpText, 3, 0, 1, 1, 0, 0, false)

	var log tview.Primitive
	if !pv.logSelect {
		log = pv.logsText
	} else {
		log = pv.logsTextArea
	}
	switch pv.fullScrState {
	case LogFull:
		pv.mainGrid.AddItem(log, 1, 0, 2, 1, 0, 0, true)
	case ProcFull:
		pv.mainGrid.AddItem(pv.procTable, 1, 0, 2, 1, 0, 0, true)
	case LogProcHalf:
		pv.mainGrid.AddItem(pv.procTable, 1, 0, 1, 1, 0, 0, true)
		pv.mainGrid.AddItem(log, 2, 0, 1, 1, 0, 0, false)
	}

	pv.mainGrid.SetTitle("Process Compose")
	//pv.mainGrid = grid
}

func (pv *pcView) redrawGrid() {
	go pv.appView.QueueUpdateDraw(func() {
		pv.createGrid()
	})
}
