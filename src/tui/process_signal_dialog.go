package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

type processSignalOption struct {
	Signal      int
	Name        string
	Description string
}

func (pv *pcView) showSignalDialog() {
	options := availableSignalOptions()
	if len(options) == 0 {
		pv.showError("Sending signals from the TUI is not supported on this platform")
		return
	}

	name := pv.getSelectedProcName()
	form := tview.NewForm()
	form.SetCancelFunc(func() {
		pv.pages.RemovePage(PageDialog)
	})
	form.SetItemPadding(1)
	form.SetBorder(true)
	form.SetButtonsAlign(tview.AlignCenter)
	form.SetTitle("Send Signal to " + name)

	labels := make([]string, len(options))
	current := 0
	for i, option := range options {
		labels[i] = fmt.Sprintf("%s (%d) - %s", option.Name, option.Signal, option.Description)
		if option.Name == "SIGTERM" {
			current = i
		}
	}

	form.AddDropDown("Signal:", labels, current, nil)
	form.AddButton("Send", func() {
		selected, _ := form.GetFormItem(0).(*tview.DropDown).GetCurrentOption()
		if selected < 0 || selected >= len(options) {
			pv.showError("Please select a signal")
			return
		}
		pv.pages.RemovePage(PageDialog)
		go pv.handleProcessSignaled(name, options[selected])
	})
	form.AddButton("Cancel", func() {
		pv.pages.RemovePage(PageDialog)
	})

	pv.styleForm(form)
	pv.showDialog(form, 76, 10)
}

func (pv *pcView) handleProcessSignaled(name string, option processSignalOption) {
	ctx, cancel := context.WithCancel(context.Background())
	go pv.showAttentionMessage(ctx, fmt.Sprintf("Sending %s to %s", option.Name, name), time.Second, false)
	pv.showAutoProgress(ctx, time.Second)
	err := pv.project.SendSignal(name, option.Signal)
	cancel()
	if err != nil {
		log.Error().Err(err).Msg("Failed to send signal")
		pv.showError(err.Error())
	}
}
