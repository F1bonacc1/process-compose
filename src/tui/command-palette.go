package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

type palettePhase int

const (
	palettePhaseCommand palettePhase = iota
	palettePhaseProcessSelect
	palettePhaseSignalSelect
	palettePhaseNumericInput
)

type paletteCommand struct {
	Name        string
	Description string
	ArgPhase    palettePhase
	Execute     func(cp *commandPalette, arg string)
}

type commandPalette struct {
	*tview.Flex
	view        *pcView
	input       *tview.InputField
	list        *tview.List
	footer      *tview.TextView
	commands    []paletteCommand
	filtered    []int
	phase       palettePhase
	selectedCmd *paletteCommand
	targetProc  string
}

func newCommandPalette(pv *pcView) *commandPalette {
	cp := &commandPalette{
		Flex: tview.NewFlex().SetDirection(tview.FlexRow),
		view: pv,
	}

	cp.commands = cp.buildCommandRegistry()
	cp.filtered = make([]int, len(cp.commands))
	for i := range cp.commands {
		cp.filtered[i] = i
	}

	cp.input = tview.NewInputField()
	cp.input.SetLabel("Command: ")
	cp.input.SetFieldWidth(0)
	cp.input.SetChangedFunc(cp.onInputChanged)

	cp.list = tview.NewList()
	cp.list.ShowSecondaryText(true)
	cp.list.SetHighlightFullLine(true)
	cp.list.SetWrapAround(false)

	cp.footer = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("↑/↓: Navigate  Enter: Select  Esc: Cancel")

	cp.applyStyles()
	cp.rebuildList()

	cp.AddItem(cp.input, 1, 0, true)
	cp.AddItem(cp.list, 0, 1, false)
	cp.AddItem(cp.footer, 1, 0, false)
	cp.SetBorder(true)
	cp.SetTitle(" Command Palette ")

	cp.SetInputCapture(cp.handleInput)

	return cp
}

func (cp *commandPalette) applyStyles() {
	styles := cp.view.styles
	bg := styles.BgColor()
	dlg := styles.Dialog()

	cp.SetBackgroundColor(bg)
	cp.SetBorderColor(styles.BorderColor())
	cp.SetTitleColor(styles.Body().SecondaryTextColor.Color())

	cp.input.SetBackgroundColor(bg)
	cp.input.SetFieldBackgroundColor(dlg.FieldBgColor.Color())
	cp.input.SetFieldTextColor(dlg.FieldFgColor.Color())
	cp.input.SetLabelColor(dlg.LabelFgColor.Color())

	cp.list.SetBackgroundColor(bg)
	cp.list.SetMainTextColor(dlg.FgColor.Color())
	cp.list.SetSecondaryTextColor(dlg.LabelFgColor.Color())
	cp.list.SetSelectedBackgroundColor(dlg.ButtonFocusBgColor.Color())
	cp.list.SetSelectedTextColor(dlg.ButtonFocusFgColor.Color())

	cp.footer.SetBackgroundColor(bg)
	cp.footer.SetTextColor(dlg.LabelFgColor.Color())
}

func (cp *commandPalette) height() int {
	itemCount := len(cp.filtered)
	rowsPerItem := 2
	if cp.phase == palettePhaseProcessSelect {
		rowsPerItem = 1
	}
	// item rows + input(1) + footer(1) + borders(2)
	h := itemCount*rowsPerItem + 4
	if h > 18 {
		h = 18
	}
	if h < 8 {
		h = 8
	}
	return h
}

func (cp *commandPalette) show() {
	cp.view.pages.AddPage(PageDialog, createDialogPage(cp, 55, cp.height()), true, true)
	cp.view.appView.SetFocus(cp.input)
}

func (cp *commandPalette) close() {
	cp.view.pages.RemovePage(PageDialog)
}

func (cp *commandPalette) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		if cp.phase != palettePhaseCommand {
			cp.resetToCommandPhase()
			return nil
		}
		cp.close()
		return nil
	case tcell.KeyUp:
		cur := cp.list.GetCurrentItem()
		if cur > 0 {
			cp.list.SetCurrentItem(cur - 1)
		}
		return nil
	case tcell.KeyDown:
		cur := cp.list.GetCurrentItem()
		if cur < cp.list.GetItemCount()-1 {
			cp.list.SetCurrentItem(cur + 1)
		}
		return nil
	case tcell.KeyEnter:
		if cp.phase == palettePhaseNumericInput {
			text := cp.input.GetText()
			cp.close()
			cp.selectedCmd.Execute(cp, text)
			return nil
		}
		if cp.list.GetItemCount() == 0 {
			return nil
		}
		cp.onSelect()
		return nil
	}
	return event
}

func (cp *commandPalette) onInputChanged(text string) {
	switch cp.phase {
	case palettePhaseCommand:
		cp.filterByText(text, cp.commandItems())
	case palettePhaseProcessSelect:
		cp.filterByText(text, cp.processItems())
	case palettePhaseSignalSelect:
		cp.filterByText(text, cp.signalItems())
	case palettePhaseNumericInput:
		// no filtering needed for numeric input
	}
}

type listItem struct {
	main      string
	secondary string
}

func (cp *commandPalette) commandItems() []listItem {
	items := make([]listItem, len(cp.commands))
	for i, cmd := range cp.commands {
		items[i] = listItem{main: cmd.Name, secondary: cmd.Description}
	}
	return items
}

func (cp *commandPalette) processItems() []listItem {
	names, err := cp.view.project.GetLexicographicProcessNames()
	if err != nil {
		return nil
	}
	items := make([]listItem, len(names))
	for i, name := range names {
		items[i] = listItem{main: name}
	}
	return items
}

func (cp *commandPalette) signalItems() []listItem {
	options := availableSignalOptions()
	items := make([]listItem, len(options))
	for i, opt := range options {
		items[i] = listItem{
			main:      fmt.Sprintf("%s (%d)", opt.Name, opt.Signal),
			secondary: opt.Description,
		}
	}
	return items
}

func (cp *commandPalette) filterByText(text string, allItems []listItem) {
	cp.list.Clear()
	lower := strings.ToLower(text)
	cp.filtered = cp.filtered[:0]
	for i, item := range allItems {
		if text == "" || strings.Contains(strings.ToLower(item.main), lower) {
			cp.filtered = append(cp.filtered, i)
			cp.list.AddItem(item.main, item.secondary, 0, nil)
		}
	}
}

func (cp *commandPalette) rebuildList() {
	cp.list.Clear()
	for _, idx := range cp.filtered {
		cmd := cp.commands[idx]
		cp.list.AddItem(cmd.Name, cmd.Description, 0, nil)
	}
}

func (cp *commandPalette) onSelect() {
	cur := cp.list.GetCurrentItem()
	if cur < 0 || cur >= len(cp.filtered) {
		return
	}

	switch cp.phase {
	case palettePhaseCommand:
		cmd := &cp.commands[cp.filtered[cur]]
		cp.selectedCmd = cmd
		if cmd.ArgPhase == palettePhaseCommand {
			// No argument needed - execute directly
			cp.close()
			cmd.Execute(cp, "")
			return
		}
		// Commands that need process selection first
		cp.transitionToProcessSelect()

	case palettePhaseProcessSelect:
		names, err := cp.view.project.GetLexicographicProcessNames()
		if err != nil || cur >= len(cp.filtered) {
			return
		}
		cp.targetProc = names[cp.filtered[cur]]

		// Check if we need a further argument phase
		switch cp.selectedCmd.ArgPhase {
		case palettePhaseProcessSelect:
			// Process was the final arg
			cp.close()
			cp.selectedCmd.Execute(cp, cp.targetProc)
		case palettePhaseSignalSelect:
			cp.transitionToSignalSelect()
		case palettePhaseNumericInput:
			cp.transitionToNumericInput()
		}

	case palettePhaseSignalSelect:
		options := availableSignalOptions()
		if cur >= len(cp.filtered) {
			return
		}
		opt := options[cp.filtered[cur]]
		cp.close()
		cp.selectedCmd.Execute(cp, strconv.Itoa(opt.Signal))

	case palettePhaseNumericInput:
		// Enter on numeric input uses the text field value directly
		text := cp.input.GetText()
		cp.close()
		cp.selectedCmd.Execute(cp, text)
	}
}

func (cp *commandPalette) transitionToProcessSelect() {
	cp.phase = palettePhaseProcessSelect
	cp.input.SetLabel("Process: ")
	cp.input.SetText("")
	cp.list.ShowSecondaryText(false)

	items := cp.processItems()
	cp.list.Clear()
	cp.filtered = make([]int, len(items))
	for i, item := range items {
		cp.filtered[i] = i
		cp.list.AddItem(item.main, item.secondary, 0, nil)
	}

	// Pre-select current process
	currentProc := cp.view.getSelectedProcName()
	for i, item := range items {
		if item.main == currentProc {
			cp.list.SetCurrentItem(i)
			break
		}
	}

	cp.footer.SetText("↑/↓: Navigate  Enter: Select  Esc: Back")
	cp.resizeDialog()
}

func (cp *commandPalette) transitionToSignalSelect() {
	cp.phase = palettePhaseSignalSelect
	cp.input.SetLabel("Signal: ")
	cp.input.SetText("")
	cp.list.ShowSecondaryText(true)

	items := cp.signalItems()
	cp.list.Clear()
	cp.filtered = make([]int, len(items))
	for i, item := range items {
		cp.filtered[i] = i
		cp.list.AddItem(item.main, item.secondary, 0, nil)
	}

	// Pre-select SIGTERM
	for i, item := range items {
		if strings.HasPrefix(item.main, "SIGTERM") {
			cp.list.SetCurrentItem(i)
			break
		}
	}

	cp.footer.SetText("↑/↓: Navigate  Enter: Send  Esc: Back")
	cp.resizeDialog()
}

func (cp *commandPalette) transitionToNumericInput() {
	cp.phase = palettePhaseNumericInput
	cp.input.SetLabel(fmt.Sprintf("Scale %s to replicas: ", cp.targetProc))
	cp.input.SetText("")
	cp.input.SetAcceptanceFunc(tview.InputFieldInteger)

	cp.list.Clear()
	cp.filtered = cp.filtered[:0]

	cp.footer.SetText("Enter: Apply  Esc: Back")
	cp.resizeDialog()
}

func (cp *commandPalette) resetToCommandPhase() {
	cp.phase = palettePhaseCommand
	cp.selectedCmd = nil
	cp.targetProc = ""
	cp.input.SetLabel("Command: ")
	cp.input.SetText("")
	cp.input.SetAcceptanceFunc(nil)
	cp.list.ShowSecondaryText(true)

	cp.filtered = make([]int, len(cp.commands))
	for i := range cp.commands {
		cp.filtered[i] = i
	}
	cp.rebuildList()

	cp.footer.SetText("↑/↓: Navigate  Enter: Select  Esc: Cancel")
	cp.resizeDialog()
}

func (cp *commandPalette) resizeDialog() {
	cp.view.pages.RemovePage(PageDialog)
	cp.view.pages.AddPage(PageDialog, createDialogPage(cp, 55, cp.height()), true, true)
	cp.view.appView.SetFocus(cp.input)
}

func (cp *commandPalette) buildCommandRegistry() []paletteCommand {
	return []paletteCommand{
		{
			Name:        "Start Process",
			Description: "Start a stopped process",
			ArgPhase:    palettePhaseProcessSelect,
			Execute: func(cp *commandPalette, arg string) {
				info, err := cp.view.project.GetProcessInfo(arg)
				if err != nil {
					cp.view.showError(err.Error())
					return
				}
				if info.IsForeground {
					cp.view.runForeground(info)
					return
				}
				go func() {
					err := cp.view.project.StartProcess(arg)
					if err != nil {
						cp.view.showError(err.Error())
					}
				}()
			},
		},
		{
			Name:        "Stop Process",
			Description: "Stop a running process",
			ArgPhase:    palettePhaseProcessSelect,
			Execute: func(cp *commandPalette, arg string) {
				go cp.view.handleProcessStopped(arg)
			},
		},
		{
			Name:        "Restart Process",
			Description: "Restart a process",
			ArgPhase:    palettePhaseProcessSelect,
			Execute: func(cp *commandPalette, arg string) {
				go func() {
					err := cp.view.project.RestartProcess(arg)
					if err != nil {
						cp.view.showError(err.Error())
					}
				}()
			},
		},
		{
			Name:        "Scale Process",
			Description: "Set the number of replicas for a process",
			ArgPhase:    palettePhaseNumericInput,
			Execute: func(cp *commandPalette, arg string) {
				scale, err := strconv.Atoi(arg)
				if err != nil {
					cp.view.showError("Invalid replica count: " + err.Error())
					return
				}
				proc := cp.targetProc
				go func() {
					log.Info().Msgf("Scaling %s to %d", proc, scale)
					err := cp.view.project.ScaleProcess(proc, scale)
					if err != nil {
						cp.view.showError(err.Error())
					}
				}()
			},
		},
		{
			Name:        "Send Signal",
			Description: "Send an OS signal to a process",
			ArgPhase:    palettePhaseSignalSelect,
			Execute: func(cp *commandPalette, arg string) {
				sig, err := strconv.Atoi(arg)
				if err != nil {
					return
				}
				go cp.view.handleProcessSignaled(cp.targetProc, processSignalOption{Signal: sig})
			},
		},
	}
}
