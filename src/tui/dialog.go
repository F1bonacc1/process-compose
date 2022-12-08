package tui

import (
	"github.com/rivo/tview"
)

func (pv *pcView) showDialog(primitive tview.Primitive) {
	pv.pages.AddPage(PageDialog, createDialogPage(primitive, 0, 0), true, true)
	pv.appView.SetFocus(primitive)
}

func createDialogPage(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}
