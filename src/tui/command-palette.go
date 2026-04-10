package tui

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
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
	palettePhaseTextInput
)

type paletteStep struct {
	Label   string
	Footer  string
	Toggles map[string]bool // toggle names with default values, nil if no toggles
}

type paletteCommand struct {
	Name        string
	Description string
	ArgPhase    palettePhase
	Steps       []paletteStep // defines multi-step text input sequences
	Execute     func(cp *commandPalette, arg string)
}

type commandPalette struct {
	*tview.Flex
	view         *pcView
	input        *tview.InputField
	list         *tview.List
	footer       *tview.TextView
	commands     []paletteCommand
	filtered     []int
	phase        palettePhase
	selectedCmd  *paletteCommand
	targetProc   string
	textInputs   map[string]string
	toggleInputs map[string]bool
	stepIndex    int
}

func newCommandPalette(pv *pcView) *commandPalette {
	cp := &commandPalette{
		Flex:         tview.NewFlex().SetDirection(tview.FlexRow),
		view:         pv,
		textInputs:   make(map[string]string),
		toggleInputs: make(map[string]bool),
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

func (cp *commandPalette) close() {
	cp.view.pages.RemovePage(PageDialog)
}

func (cp *commandPalette) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		if cp.phase != palettePhaseCommand {
			if cp.stepIndex > 0 {
				cp.stepIndex--
				cp.goBackToStep()
				return nil
			}
			if cp.phase == palettePhaseNumericInput || cp.phase == palettePhaseSignalSelect {
				cp.transitionToProcessSelect()
				return nil
			}
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
	case tcell.KeyTab:
		if cp.phase == palettePhaseTextInput && len(cp.toggleInputs) > 0 {
			// Toggle all registered toggles (typically just one per step)
			for k, v := range cp.toggleInputs {
				cp.toggleInputs[k] = !v
			}
			cp.updateToggleFooter()
			return nil
		}
		return event
	case tcell.KeyEnter:
		if cp.phase == palettePhaseNumericInput {
			text := cp.input.GetText()
			cp.close()
			cp.selectedCmd.Execute(cp, text)
			return nil
		}
		if cp.phase == palettePhaseTextInput {
			text := cp.input.GetText()
			if text == "" {
				return nil
			}
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
	case palettePhaseNumericInput, palettePhaseTextInput:
		// no filtering needed for direct input
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
		if cmd.ArgPhase == palettePhaseTextInput {
			cp.stepIndex = 0
			cp.goBackToStep()
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

func (cp *commandPalette) goBackToStep() {
	if cp.selectedCmd == nil || cp.stepIndex >= len(cp.selectedCmd.Steps) {
		cp.resetToCommandPhase()
		return
	}
	step := cp.selectedCmd.Steps[cp.stepIndex]
	cp.toggleInputs = make(map[string]bool)
	for k, v := range step.Toggles {
		cp.toggleInputs[k] = v
	}
	cp.transitionToTextInput(step.Label, step.Footer)
	if len(cp.toggleInputs) > 0 {
		cp.updateToggleFooter()
	}
}

func (cp *commandPalette) transitionToTextInput(label, footerText string) {
	cp.phase = palettePhaseTextInput
	cp.input.SetLabel(label)
	cp.input.SetText("")
	cp.input.SetAcceptanceFunc(nil)

	cp.list.Clear()
	cp.filtered = cp.filtered[:0]

	cp.footer.SetText(footerText)
	cp.resizeDialog()
}

func (cp *commandPalette) resetToCommandPhase() {
	cp.phase = palettePhaseCommand
	cp.selectedCmd = nil
	cp.targetProc = ""
	cp.textInputs = make(map[string]string)
	cp.toggleInputs = make(map[string]bool)
	cp.stepIndex = 0
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

func (cp *commandPalette) updateToggleFooter() {
	var parts []string
	for k, v := range cp.toggleInputs {
		hlColor := string(cp.view.styles.Body().SecondaryTextColor)
		state := fmt.Sprintf("[%s]NO[-]", hlColor)
		if v {
			state = fmt.Sprintf("[%s]YES[-]", hlColor)
		}
		parts = append(parts, fmt.Sprintf("Tab: %s: %s", k, state))
	}
	parts = append(parts, "Enter: Confirm  Esc: Back")
	cp.footer.SetText(strings.Join(parts, "  "))
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
		{
			Name:        "Create Process",
			Description: "Create and start an ephemeral process",
			ArgPhase:    palettePhaseTextInput,
			Steps: []paletteStep{
				{Label: "Name: ", Footer: "Enter process name  Esc: Back"},
				{Label: "Command: ", Toggles: map[string]bool{"Interactive": false}},
			},
			Execute: func(cp *commandPalette, arg string) {
				switch cp.stepIndex {
				case 0: // name entered
					names, _ := cp.view.project.GetLexicographicProcessNames()
					if slices.Contains(names, arg) {
						cp.input.SetText("")
						cp.footer.SetText(fmt.Sprintf("[red]Process %q already exists[-]  Esc: Back", arg))
						return
					}
					cp.textInputs["name"] = arg
					cp.stepIndex = 1
					cp.goBackToStep()
				case 1: // command entered
					name := cp.textInputs["name"]
					cmd := arg
					interactive := cp.toggleInputs["Interactive"]
					cwd, _ := os.Getwd()
					cp.close()

					go func() {
						proj, err := cp.view.buildCurrentProject()
						if err != nil {
							cp.view.showError(err.Error())
							return
						}
						shell := command.DefaultShellConfig()
						proc := types.ProcessConfig{
							Name:          name,
							ReplicaName:   name,
							Command:       cmd,
							WorkingDir:    cwd,
							IsInteractive: interactive,
						}
						proc.AssignProcessExecutableAndArgs(shell, shell.ElevatedShellArg)
						proj.Processes[name] = proc
						_, err = cp.view.project.UpdateProject(proj)
						if err != nil {
							cp.view.showError(err.Error())
							return
						}
						cp.view.refreshTableState()
						cp.view.appView.QueueUpdateDraw(func() {
							cp.view.fillTableData()
							cp.view.selectTableProcess(name)
						})
					}()
				}
			},
		},
		{
			Name:        "Delete Process",
			Description: "Stop and remove a process",
			ArgPhase:    palettePhaseProcessSelect,
			Execute: func(cp *commandPalette, arg string) {
				// Check reverse dependencies
				graph, err := cp.view.project.GetDependencyGraph()
				if err != nil {
					cp.view.showError(err.Error())
					return
				}
				var dependents []string
				for nodeName, node := range graph.AllNodes {
					if nodeName == arg {
						continue
					}
					if _, ok := node.DependsOn[arg]; ok {
						dependents = append(dependents, nodeName)
					}
				}
				if len(dependents) > 0 {
					cp.view.showError(fmt.Sprintf("Cannot delete %q: depended on by %s", arg, strings.Join(dependents, ", ")))
					return
				}

				go func() {
					proj, err := cp.view.buildCurrentProject()
					if err != nil {
						cp.view.showError(err.Error())
						return
					}
					delete(proj.Processes, arg)
					_, err = cp.view.project.UpdateProject(proj)
					if err != nil {
						cp.view.showError(err.Error())
						return
					}
					cp.view.refreshTableState()
					cp.view.appView.QueueUpdateDraw(func() {
						cp.view.loggedProc = ""
						cp.view.logsText.Clear()
						cp.view.fillTableData()
						cp.view.selectFirstEnabledProcess()
					})
				}()
			},
		},
	}
}

func (pv *pcView) buildCurrentProject() (*types.Project, error) {
	names, err := pv.project.GetLexicographicProcessNames()
	if err != nil {
		return nil, err
	}
	procs := make(types.Processes, len(names))
	for _, name := range names {
		info, err := pv.project.GetProcessInfo(name)
		if err != nil {
			return nil, err
		}
		procs[name] = *info
	}
	return &types.Project{Processes: procs}, nil
}
