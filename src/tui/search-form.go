package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (pv *pcView) showSearch() {
	const fieldWidth = 50
	f := tview.NewForm()
	f.SetCancelFunc(func() {
		pv.pages.RemovePage(PageDialog)
	})
	f.SetItemPadding(1)
	f.SetBorder(true)
	f.SetButtonsAlign(tview.AlignCenter)
	f.SetTitle("Search Log")
	f.AddInputField("Search For", pv.logsText.getSearchTerm(), fieldWidth, nil, nil)
	f.AddCheckbox("Case Sensitive", false, nil)
	f.AddCheckbox("Regex", false, nil)
	searchFunc := func() {
		searchTerm := f.GetFormItem(0).(*tview.InputField).GetText()
		caseSensitive := f.GetFormItem(1).(*tview.Checkbox).IsChecked()
		isRegex := f.GetFormItem(2).(*tview.Checkbox).IsChecked()
		pv.stopFollowLog()
		if err := pv.logsText.searchString(searchTerm, isRegex, caseSensitive); err != nil {
			f.SetTitle(err.Error())
			return
		}
		pv.pages.RemovePage(PageDialog)
		pv.logsText.SetTitle(pv.getLogTitle(pv.getSelectedProcName()))
		pv.updateHelpTextView()
	}
	f.AddButton("Search", searchFunc)
	f.AddButton("Cancel", func() {
		pv.pages.RemovePage(PageDialog)
	})
	f.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			searchFunc()
		case tcell.KeyEsc:
			pv.pages.RemovePage(PageDialog)
		default:
			return event
		}
		return nil
	})
	f.SetFocus(0)
	pv.styleForm(f)
	// Display and focus the dialog
	pv.pages.AddPage(PageDialog, createDialogPage(f, fieldWidth+20, 11), true, true)
	pv.appView.SetFocus(f)
}
