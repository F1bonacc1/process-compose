package tui

import (
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

const cancelLbl = "Cancel"

func (pv *pcView) showThemeSelector() {
	themeSelector := pv.createThemeSelector()
	_, _, w, h := themeSelector.GetRect()
	pv.pages.AddPage(PageDialog, createDialogPage(themeSelector, w, h), true, true)
	pv.appView.SetFocus(themeSelector)
}

func (pv *pcView) createThemeSelector() tview.Primitive {
	list := tview.NewList()
	themeNames := pv.themes.GetThemeNames()
	maxLblLen := 0
	currentStyle := pv.styles.GetStyleName()
	const scList = "123456789abcdefghijklmnopqrstuvwyzABCDEGHIJKLMNOPQRSTUVWXYZ"
	scIdx := 0
	customFound := false
	for _, theme := range themeNames {
		if theme == config.CustomStyleName {
			customFound = true
			continue
		}
		selectThemeLbl := "Select " + theme
		r := '0'
		if scIdx < len(scList) {
			r = rune(scList[scIdx])
		}
		scIdx++
		list.AddItem(theme, selectThemeLbl, r, func() {
			pv.setTheme(theme)
			pv.saveTuiState()
			pv.pages.RemovePage(PageDialog)
		})
		if theme == currentStyle {
			list.SetCurrentItem(scIdx)
		}
		if len(selectThemeLbl) > maxLblLen {
			maxLblLen = len(selectThemeLbl)
		}
	}
	list.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if mainText == cancelLbl {
			return
		} else {
			pv.setTheme(mainText)
		}
	})
	if customFound {
		list.AddItem(config.CustomStyleName, "Load From File", 'F', func() {
			pv.setTheme(config.CustomStyleName)
			pv.pages.RemovePage(PageDialog)
		})
		if config.CustomStyleName == currentStyle {
			list.SetCurrentItem(-1)
		}
	}
	list.AddItem(cancelLbl, "Select to close", 'x', func() {
		log.Debug().Msgf("reverting to the original theme %s", currentStyle)
		pv.setTheme(currentStyle)
		pv.pages.RemovePage(PageDialog)
	})
	list.SetBorder(true).SetTitle("Themes")

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(list, (len(themeNames)+2)*2, 1, true), maxLblLen+10, 1, true).
		AddItem(nil, 0, 1, false)
	list.SetBackgroundColor(pv.styles.BgColor())
	return flex
}

func (pv *pcView) setTheme(name string) {
	pv.themes.SelectStyles(name)
}
