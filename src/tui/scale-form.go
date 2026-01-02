package tui

import (
	"strconv"

	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

func (pv *pcView) showScale() {
	f := tview.NewForm()
	f.SetCancelFunc(func() {
		pv.pages.RemovePage(PageDialog)
	})
	f.SetItemPadding(3)
	f.SetBorder(true)
	name := pv.getSelectedProcName()
	procConfig, err := pv.project.GetProcessInfo(name)
	if err != nil {
		pv.showError(err.Error())
		return
	}
	if procConfig.Schedule != nil {
		pv.showError("Scaling is not supported for scheduled processes")
		return
	}
	f.SetTitle("Scale " + name + " Process")
	f.AddInputField("Replicas:", "1", 0, nil, nil)
	f.AddButton("Scale", func() {
		scale, err := strconv.Atoi(f.GetFormItem(0).(*tview.InputField).GetText())
		if err != nil {
			pv.showError("Invalid Scale: " + err.Error())
			return
		}
		log.Info().Msgf("Scaling %s to %d", name, scale)
		err = pv.project.ScaleProcess(name, scale)
		if err != nil {
			pv.showError("Invalid Scale: " + err.Error())
		}
		pv.pages.RemovePage(PageDialog)
	})
	f.AddButton("Cancel", func() {
		pv.pages.RemovePage(PageDialog)
	})
	f.SetButtonsAlign(tview.AlignCenter)
	pv.styleForm(f)
	pv.showDialog(f, 60, 10)
}
