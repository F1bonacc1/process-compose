package tui

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

type namespaceModal struct {
	*tview.Grid
	view           *pcView
	nsList         *tview.List
	opList         *tview.List
	rawNamespaces  []string
	displayOptions []string
	footer         *tview.TextView
	focusable      []tview.Primitive
	focusIndex     int
}

type namespaceOperation string

const (
	namespaceOperationStart   namespaceOperation = "Start"
	namespaceOperationStop    namespaceOperation = "Stop"
	namespaceOperationRestart namespaceOperation = "Restart"
)

func newNamespaceModal(view *pcView) *namespaceModal {
	modal := &namespaceModal{
		Grid: tview.NewGrid().SetBorders(true).SetRows(0, 1),
		view: view,
	}
	modal.SetTitle("Namespace Operations")
	modal.rawNamespaces, modal.displayOptions = modal.getNamespacesAndOptions()

	// Initialize components
	modal.createComponents()
	layout := modal.createLayout()

	modal.footer = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("↑/↓: Navigate  Tab: Switch fields  Enter: Execute  Esc: Cancel")

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			view.pages.RemovePage(PageDialog)
			return nil
		}
		if event.Key() == tcell.KeyTab {
			if event.Modifiers()&tcell.ModShift != 0 {
				modal.focusPrevious()
			} else {
				modal.focusNext()
			}
			return nil
		}
		// Enter handling
		if event.Key() == tcell.KeyEnter {
			if modal.view.appView.GetFocus() == modal.nsList {
				modal.focusNext()
				return nil
			}
			if modal.view.appView.GetFocus() == modal.opList {
				modal.onExecute()
				return nil
			}
		}
		return event
	})

	modal.AddItem(layout, 0, 0, 1, 1, 0, 0, true).
		AddItem(modal.footer, 1, 0, 1, 1, 0, 0, false)

	// Apply styles
	modal.StylesChanged(view.styles)

	return modal
}

func (nm *namespaceModal) createComponents() {
	// Namespace List
	nm.nsList = tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true)
	nm.nsList.SetBorder(true).SetTitle("Namespace")

	for _, opt := range nm.displayOptions {
		nm.nsList.AddItem(opt, "", 0, nil)
	}

	// Operation List
	nm.opList = tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true)
	nm.opList.SetBorder(true).SetTitle("Operation")

	ops := []string{
		"Stop    - Stop all running processes",
		"Start   - Start all stopped processes",
		"Restart - Restart all processes",
	}
	for _, op := range ops {
		nm.opList.AddItem(op, "", 0, nil)
	}

	// Focusable elements in order
	nm.focusable = []tview.Primitive{
		nm.nsList,
		nm.opList,
	}
	nm.focusIndex = 0
}

func (nm *namespaceModal) createLayout() *tview.Flex {
	// Main Layout
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nm.nsList, 0, 1, true).
		AddItem(tview.NewBox(), 1, 0, false). // Spacer
		AddItem(nm.opList, 5, 0, false)

	return layout
}

func (nm *namespaceModal) focusNext() {
	nm.focusIndex = (nm.focusIndex + 1) % len(nm.focusable)
	nm.view.appView.SetFocus(nm.focusable[nm.focusIndex])
}

func (nm *namespaceModal) focusPrevious() {
	nm.focusIndex = (nm.focusIndex - 1 + len(nm.focusable)) % len(nm.focusable)
	nm.view.appView.SetFocus(nm.focusable[nm.focusIndex])
}

func (nm *namespaceModal) onExecute() {
	// Namespace
	nsIdx := nm.nsList.GetCurrentItem()
	var ns string
	if nsIdx >= 0 && nsIdx < len(nm.rawNamespaces) {
		ns = nm.rawNamespaces[nsIdx]
	} else {
		ns = types.DefaultNamespace
	}

	// Operation
	opIdx := nm.opList.GetCurrentItem()

	nm.view.pages.RemovePage(PageDialog)

	switch opIdx {
	case 0: // Stop
		go nm.executeOperation(ns, namespaceOperationStop)
	case 1: // Start
		go nm.executeOperation(ns, namespaceOperationStart)
	case 2: // Restart
		go nm.executeOperation(ns, namespaceOperationRestart)
	}
}

func (nm *namespaceModal) getNamespacesAndOptions() ([]string, []string) {
	states, err := nm.view.project.GetProcessesState()
	if err != nil {
		log.Err(err).Msg("Failed to get processes state for namespace modal")
		def := types.DefaultNamespace
		return []string{def}, []string{fmt.Sprintf("%s (unknown)", def)}
	}

	type nsStat struct {
		total   int
		running int
	}
	stats := make(map[string]*nsStat)

	for _, state := range states.States {
		for _, ns := range state.Namespace.OrDefault() {
			if _, ok := stats[ns]; !ok {
				stats[ns] = &nsStat{}
			}
			stats[ns].total++
			if state.IsRunning {
				stats[ns].running++
			}
		}
	}

	namespaces := make([]string, 0, len(stats))
	for ns := range stats {
		namespaces = append(namespaces, ns)
	}
	slices.Sort(namespaces)

	options := make([]string, len(namespaces))
	for i, ns := range namespaces {
		s := stats[ns]
		options[i] = fmt.Sprintf("%-15s (%d processes, %d running)", ns, s.total, s.running)
	}

	return namespaces, options
}

func (nm *namespaceModal) executeOperation(namespace string, operation namespaceOperation) {
	ctx, cancel := context.WithCancel(context.Background())
	go nm.view.showAttentionMessage(ctx, fmt.Sprintf("Executing '%s' on namespace '%s'", operation, namespace), time.Second*1, false)
	nm.view.showAutoProgress(ctx, time.Second*1)
	defer cancel()
	var err error
	switch operation {
	case namespaceOperationStart:
		err = nm.view.project.StartNamespace(namespace)
	case namespaceOperationStop:
		err = nm.view.project.StopNamespace(namespace)
	case namespaceOperationRestart:
		err = nm.view.project.RestartNamespace(namespace)
	default:
		log.Error().Msgf("Unknown operation: %s", operation)
		return
	}
	// Stop the progress animation before reporting the result
	cancel()

	if err != nil {
		log.Err(err).Msgf("Failed to %s namespace %s", operation, namespace)
		nm.view.showError(fmt.Sprintf("Failed to %s namespace '%s': %s", operation, namespace, err.Error()))
		return
	}
	log.Info().Msgf("Namespace %s %sed", namespace, operation)
}

func (nm *namespaceModal) StylesChanged(s *config.Styles) {
	nm.SetBackgroundColor(s.BgColor())
	nm.SetBordersColor(s.BorderColor())

	// Lists - match ThemeSelector style (transparent/bg color)
	nm.nsList.SetBackgroundColor(s.BgColor())
	nm.nsList.SetMainTextColor(s.FgColor())
	nm.nsList.SetSelectedBackgroundColor(s.Dialog().ButtonFocusBgColor.Color())
	nm.nsList.SetSelectedTextColor(s.Dialog().ButtonFocusFgColor.Color())
	nm.nsList.SetBorderColor(s.BorderColor())
	nm.nsList.SetTitleColor(s.Body().SecondaryTextColor.Color())

	nm.opList.SetBackgroundColor(s.BgColor())
	nm.opList.SetMainTextColor(s.FgColor())
	nm.opList.SetSelectedBackgroundColor(s.Dialog().ButtonFocusBgColor.Color())
	nm.opList.SetSelectedTextColor(s.Dialog().ButtonFocusFgColor.Color())
	nm.opList.SetBorderColor(s.BorderColor())
	nm.opList.SetTitleColor(s.Body().SecondaryTextColor.Color())

	nm.footer.SetTextColor(s.Dialog().LabelFgColor.Color())
	nm.footer.SetBackgroundColor(s.BgColor())
}

func (nm *namespaceModal) Height() int {
	// Calculate height based on content
	// Namespace list: 2 borders + len(namespaces)
	// Operation list: 2 borders + 3 items
	// Padding/Titles: 2 lines for field titles ("Namespace:", "Operation:")
	// Layout spacers: 1 lines (between lists)
	// Footer: 1 line

	nsHeight := len(nm.rawNamespaces) + 2
	opHeight := 3 + 2

	// Total elements in FlexRow:
	// nsList (nsHeight) + Spacer(1) + opList(opHeight)
	// + Footer (1) inside Grid but outside Flex layout?
	// The Grid has 2 rows: Main layout and Footer.

	contentHeight := nsHeight + 1 + opHeight
	footerHeight := 1

	// Grid borders: 2
	// Footer border: 1
	totalHeight := contentHeight + footerHeight + 2 + 1

	return totalHeight
}
