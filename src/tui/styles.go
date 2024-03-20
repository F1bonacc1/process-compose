package tui

import (
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (pv *pcView) setStyles(s *config.Styles) {
	pv.styles = s
}

func (pv *pcView) StylesChanged(s *config.Styles) {
	pv.setStyles(s)

	pv.setStatTableStyles(s)
	pv.mainGrid.SetBackgroundColor(s.BgColor())
	pv.mainGrid.SetBordersColor(s.BorderColor())
	pv.setProcTableStyles(s)
	pv.setLogViewStyle(s)
	pv.setHelpTextStyles(s)
}

func (pv *pcView) setStatTableStyles(s *config.Styles) {
	pv.statTable.SetBackgroundColor(s.BgColor())
	for r := range pv.statTable.GetRowCount() {
		pv.statTable.GetCell(r, 0).SetTextColor(s.Style.StatTable.KeyFgColor.Color())
		pv.statTable.GetCell(r, 1).SetTextColor(s.Style.StatTable.ValueFgColor.Color())
	}
	pv.statTable.GetCell(0, 3).SetTextColor(s.StatTable().LogoColor.Color())
}

func (pv *pcView) setProcTableStyles(s *config.Styles) {
	pv.procTable.SetBackgroundColor(s.BgColor())
	for r := range pv.procTable.GetRowCount() {
		for c := range pv.procTable.GetColumnCount() {
			if r == 0 {
				pv.procTable.GetCell(r, c).SetTextColor(s.Style.ProcTable.HeaderFgColor.Color())
				continue
			}
			if c == 0 {
				continue
			}
			pv.procTable.GetCell(r, c).SetTextColor(s.Style.ProcTable.FgColor.Color())
		}
	}
}

func (pv *pcView) setLogViewStyle(s *config.Styles) {
	pv.logsText.SetBorderColor(s.BorderColor())
	pv.logsText.SetTitleColor(s.Body().SecondaryTextColor.Color())
	pv.logsText.SetBackgroundColor(s.BgColor())
}

func (pv *pcView) setHelpTextStyles(s *config.Styles) {
	pv.helpText.SetBackgroundColor(s.BgColor())
	pv.helpText.SetBorderColor(s.BorderColor())
	pv.helpText.SetTextColor(s.Help().KeyColor.Color())
	pv.shortcuts.StylesChanged(s)
	pv.updateHelpTextView()
}

func (pv *pcView) styleForm(f *tview.Form) {
	f.SetBackgroundColor(pv.styles.BgColor())
	f.SetBorderColor(pv.styles.BorderColor())
	f.SetTitleColor(pv.styles.Body().SecondaryTextColor.Color())
	f.SetFieldBackgroundColor(pv.styles.Dialog().FieldBgColor.Color())
	f.SetFieldTextColor(pv.styles.Dialog().FieldFgColor.Color())
	f.SetButtonBackgroundColor(pv.styles.Dialog().ButtonBgColor.Color())
	f.SetButtonTextColor(pv.styles.Dialog().ButtonFgColor.Color())
	f.SetButtonActivatedStyle(tcell.StyleDefault.Background(pv.styles.Dialog().ButtonFocusBgColor.Color()).Foreground(pv.styles.Dialog().ButtonFocusFgColor.Color()))
	f.SetLabelColor(pv.styles.Dialog().LabelFgColor.Color())
}
