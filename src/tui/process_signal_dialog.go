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
	list := tview.NewList()

	const scList = "123456789abcdefghijklmnopqrstuvwyzABCDEGHIJKLMNOPQRSTUVWXYZ"
	maxLblLen := 0
	current := 0
	for i, option := range options {
		opt := option
		label := fmt.Sprintf("%s (%d)", opt.Name, opt.Signal)
		secondaryText := opt.Description
		r := '0'
		if i < len(scList) {
			r = rune(scList[i])
		}
		if opt.Name == "SIGTERM" {
			current = i
		}
		list.AddItem(label, secondaryText, r, func() {
			pv.pages.RemovePage(PageDialog)
			go pv.handleProcessSignaled(name, opt)
		})
		if len(secondaryText) > maxLblLen {
			maxLblLen = len(secondaryText)
		}
	}
	list.AddItem(cancelLbl, "Select to close", 'x', func() {
		pv.pages.RemovePage(PageDialog)
	})
	list.SetCurrentItem(current)
	list.SetDoneFunc(func() {
		pv.pages.RemovePage(PageDialog)
	})
	list.SetBorder(true).SetTitle("Send Signal to " + name)

	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("↑/↓: Navigate  Enter: Send  Esc: Cancel")
	footer.SetTextColor(pv.styles.Dialog().LabelFgColor.Color())
	footer.SetBackgroundColor(pv.styles.BgColor())

	listHeight := (len(options) + 2) * 2
	footerHeight := 1
	totalHeight := listHeight + footerHeight
	listWidth := maxLblLen + 10

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(list, listHeight, 0, true).
			AddItem(footer, footerHeight, 0, false), listWidth, 1, true).
		AddItem(nil, 0, 1, false)
	list.SetBackgroundColor(pv.styles.BgColor())

	pv.pages.AddPage(PageDialog, createDialogPage(flex, listWidth, totalHeight), true, true)
	pv.appView.SetFocus(flex)
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
