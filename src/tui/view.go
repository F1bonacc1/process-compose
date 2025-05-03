package tui

import (
	"context"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/updater"
	"github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/rivo/tview"
)

type scrSplitState int
type commandType int

const (
	LogFull     scrSplitState = 0
	ProcFull    scrSplitState = 1
	LogProcHalf scrSplitState = 2
)

const (
	commandModeOff commandType = iota
	commandModeDisabled
	commandModeSearch
	commandModePassword
)

const (
	PageMain   = "main"
	PageDialog = "dialog"
	AllNS      = "PC_ALL_NS_FILTER"
)

const shutDownAfterSec = 10

var pcv *pcView

type pcView struct {
	procTable             *tview.Table
	statTable             *tview.Table
	appView               *tview.Application
	logsText              *LogView
	statusText            *tview.TextView
	helpFooter            *tview.Flex
	pages                 *tview.Pages
	procNames             []string
	logFollow             bool
	logSelect             bool
	scrSplitState         scrSplitState
	loggedProc            string
	shortcuts             *ShortCuts
	extraShortCutsPaths   []string
	procCountCell         *tview.TableCell
	procMemCpuCell        *tview.TableCell
	mainGrid              *tview.Grid
	logsTextArea          *tview.TextArea
	project               app.IProject
	sortMtx               sync.Mutex
	stateSorter           StateSorter
	procRegex             *regexp.Regexp
	procRegexMtx          sync.Mutex
	procColumns           map[ColumnID]string
	refreshRate           time.Duration
	cancelFn              context.CancelFunc
	cancelLogFn           context.CancelFunc
	cancelSigFn           context.CancelFunc
	ctxApp                context.Context
	cancelAppFn           context.CancelFunc
	selectedNsMtx         sync.Mutex
	selectedNs            string
	selectedNsChanged     atomic.Bool
	hideDisabled          atomic.Bool
	commandModeType       commandType
	styles                *config.Styles
	themes                *config.Themes
	helpDialog            *helpDialog
	settings              *config.Settings
	isFullScreen          bool
	isReadOnlyMode        bool
	isExitConfirmDisabled bool
	detachOnSuccess       bool
}

func newPcView(project app.IProject) *pcView {

	pv := &pcView{
		appView:        tview.NewApplication(),
		logsText:       NewLogView(project.GetLogLength()),
		statusText:     tview.NewTextView().SetDynamicColors(true),
		logFollow:      true,
		scrSplitState:  LogProcHalf,
		helpFooter:     tview.NewFlex(),
		loggedProc:     "",
		procCountCell:  tview.NewTableCell(""),
		procMemCpuCell: tview.NewTableCell(""),
		mainGrid:       tview.NewGrid(),
		logsTextArea:   tview.NewTextArea(),
		logSelect:      false,
		project:        project,
		refreshRate:    time.Second,
		stateSorter: StateSorter{
			sortByColumn: ProcessStateName,
			isAsc:        true,
		},
		procColumns: map[ColumnID]string{},
		selectedNs:  AllNS,

		// configuration
		shortcuts: newShortCuts(),
		themes:    config.NewThemes(),
		settings:  config.NewSettings(),
	}
	pv.ctxApp, pv.cancelAppFn = context.WithCancel(context.Background())
	pv.statTable = pv.createStatTable()
	go pv.loadProcNames()
	pv.startMonitoring()
	pv.loadShortcuts()
	pv.setShortCutsActions()
	pv.procTable = pv.createProcTable()
	pv.updateHelpTextView()
	pv.createGrid()
	pv.createLogSelectionTextArea()
	pv.pages = tview.NewPages().
		AddPage(PageMain, pv.mainGrid, true, true)

	pv.mainGrid.SetInputCapture(pv.onMainGridKey)
	pv.appView.SetRoot(pv.pages, true).EnableMouse(true).SetInputCapture(pv.onAppKey)
	pv.recreateHelpDialog()
	pv.loadThemes()

	if len(pv.procNames) > 0 {
		name := pv.procNames[0]
		pv.logsText.SetTitle(name)
		pv.followLog(name)
	}
	//pv.dumpStyles()
	return pv
}

func (pv *pcView) loadProcNames() {
	for {
		var err error
		pv.procNames, err = pv.project.GetLexicographicProcessNames()
		if err != nil {
			continue
		}
		break
	}
}

func (pv *pcView) loadShortcuts() {
	paths := config.GetShortCutsPaths(pv.extraShortCutsPaths)
	for _, path := range paths {
		_ = pv.shortcuts.loadFromFile(path)
	}
}

func (pv *pcView) setShortCutsActions() {
	pv.shortcuts.setAction(ActionQuit, pv.terminateAppView)
	pv.shortcuts.setAction(ActionLogScreen, func() {
		if pv.scrSplitState == LogFull {
			pv.scrSplitState = LogProcHalf
		} else {
			pv.scrSplitState = LogFull
		}
		pv.redrawGrid()
		pv.updateHelpTextView()
	})
	pv.shortcuts.setAction(ActionFollowLog, pv.toggleLogFollow)
	pv.shortcuts.setAction(ActionWrapLog, func() {
		pv.logsText.ToggleWrap()
		pv.updateHelpTextView()
	})
	pv.shortcuts.setAction(ActionLogSelection, func() {
		pv.stopFollowLog()
		pv.toggleLogSelection()
		pv.appView.SetFocus(pv.logsTextArea)
		pv.updateHelpTextView()
	})
	pv.shortcuts.setAction(ActionProcessScreen, func() {
		if pv.scrSplitState == ProcFull {
			pv.scrSplitState = LogProcHalf
		} else {
			pv.scrSplitState = ProcFull
		}
		pv.redrawGrid()
		pv.onProcRowSpanChange()
		pv.updateHelpTextView()
	})
	pv.shortcuts.setAction(ActionProcessScale, pv.showScale)
	pv.shortcuts.setAction(ActionProcessInfo, pv.showInfo)
	pv.shortcuts.setAction(ActionLogFind, pv.showSearch)
	pv.shortcuts.setAction(ActionLogFindNext, func() {
		pv.logsText.SearchNext()
		pv.logsText.SetTitle(pv.getLogTitle(pv.getSelectedProcName()))
	})
	pv.shortcuts.setAction(ActionLogFindPrev, func() {
		pv.logsText.SearchPrev()
		pv.logsText.SetTitle(pv.getLogTitle(pv.getSelectedProcName()))
	})
	pv.shortcuts.setAction(ActionLogFindExit, func() {
		if pv.logsText.isSearchActive() {
			pv.exitSearch()
		} else if pv.procRegex != nil {
			pv.resetProcessSearch()
			pv.commandModeType = commandModeOff
			pv.redrawGrid()
		}
	})
	pv.shortcuts.setAction(ActionNsFilter, pv.showNsFilter)
	pv.shortcuts.setAction(ActionHideDisabled, func() {
		pv.hideDisabled.Store(!pv.hideDisabled.Load())
		pv.updateHelpTextView()
	})
	pv.shortcuts.setAction(ActionHelp, func() {
		pv.showDialog(pv.helpDialog, 50, 33)
	})
	pv.shortcuts.setAction(ActionThemeSelector, pv.showThemeSelector)
	pv.shortcuts.setAction(ActionSendToBackground, pv.runShellProcess)
	pv.shortcuts.setAction(ActionFullScreen, func() {
		pv.setFullScreen(!pv.isFullScreen)
	})
	pv.shortcuts.setAction(ActionFocusChange, pv.changeFocus)
	pv.shortcuts.setAction(ActionProcFilter, func() {
		pv.commandModeType = commandModeSearch
		pv.redrawGrid()
	})
	pv.shortcuts.setAction(ActionClearLog, func() {
		pv.logsText.Clear()
		pv.truncateLog()
	})
	pv.shortcuts.setAction(ActionMarkLog, func() {
		pv.logsText.AddMark()
	})
	pv.shortcuts.setAction(ActionEditProcess, func() {
		pv.editSelectedProcess()
	})
	pv.shortcuts.setAction(ActionReloadConfig, func() {
		_, err := pv.project.ReloadProject()
		if err != nil {
			pv.showError(err.Error())
		}
	})
	pv.shortcuts.setAction(ActionProcessStop, func() {
		name := pv.getSelectedProcName()
		go pv.handleProcessStopped(name)
	})
	pv.shortcuts.setAction(ActionProcessStart, func() {
		pv.startProcess()
		pv.showPassIfNeeded()
	})
	pv.shortcuts.setAction(ActionProcessRestart, func() {
		name := pv.getSelectedProcName()
		err := pv.project.RestartProcess(name)
		if err != nil {
			pv.showError(err.Error())
		}
		pv.showPassIfNeeded()
	})
}

func (pv *pcView) setFullScreen(isFullScreen bool) {
	pv.isFullScreen = isFullScreen
	pv.logsText.SetBorder(!pv.isFullScreen)
	pv.redrawGrid()
}

func (pv *pcView) loadThemes() {
	pv.themes.AddListener(pv)
}

func (pv *pcView) onAppKey(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyCtrlC {
		pv.terminateAppView()
		return nil
	} else {
		return event
	}
}

func (pv *pcView) onMainGridKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case pv.shortcuts.ShortCutKeys[ActionLogSelection].key:
		if !config.IsLogSelectionOn() {
			return event
		}
		pv.shortcuts.ShortCutKeys[ActionLogSelection].actionFn()
	case pv.shortcuts.ShortCutKeys[ActionLogFindExit].key:
		if !(pv.logsText.isSearchActive() || pv.procRegex != nil) {
			return event
		}
		pv.shortcuts.ShortCutKeys[ActionLogFindExit].actionFn()

	case tcell.KeyRune:
		switch event.Rune() {
		case pv.shortcuts.ShortCutKeys[ActionLogSelection].rune:
			if !config.IsLogSelectionOn() {
				return event
			}
			pv.shortcuts.ShortCutKeys[ActionLogSelection].actionFn()
		case pv.shortcuts.ShortCutKeys[ActionLogFindExit].rune:
			if !(pv.logsText.isSearchActive() || pv.procRegex != nil) {
				return event
			}
			pv.shortcuts.ShortCutKeys[ActionLogFindExit].actionFn()
		default:
			if !pv.isCommandModeDisabled() {
				return event
			}
			return pv.shortcuts.runRuneAction(event.Rune(), event) //event
		}
	default:
		return pv.shortcuts.runKeyAction(event.Key(), event) //event
	}
	return nil
}

func (pv *pcView) exitSearch() {
	pv.logsText.resetSearch()
	pv.logsText.SetTitle(pv.getLogTitle(pv.getSelectedProcName()))
	pv.updateHelpTextView()
}

func (pv *pcView) terminateAppView() {

	result := "Are you sure you want to quit?\nThis will terminate all the running processes."
	if pv.project.IsRemote() {
		result = "Detach from the remote project?"
	}
	if pv.isExitConfirmDisabled {
		go pv.handleShutDown()
		return
	}
	m := tview.NewModal().
		SetText(result).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Yes" {
				go pv.handleShutDown()
			}
			pv.pages.SwitchToPage(PageMain)
			pv.pages.RemovePage(PageDialog)
		})
	m.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Rune() {
		case 'h':
			return tcell.NewEventKey(tcell.KeyLeft, e.Rune(), e.Modifiers())
		case 'l':
			return tcell.NewEventKey(tcell.KeyRight, e.Rune(), e.Modifiers())
		case 'j':
			return tcell.NewEventKey(tcell.KeyDown, e.Rune(), e.Modifiers())
		case 'k':
			return tcell.NewEventKey(tcell.KeyUp, e.Rune(), e.Modifiers())
		}

		return e
	})
	// Display and focus the dialog
	pv.pages.AddPage(PageDialog, createDialogPage(m, 50, 50), true, true)
}

func (pv *pcView) showError(errMessage string) {
	m := tview.NewModal().
		SetText(fmt.Sprintf("Error: [white:red]%s[-:-:-]", errMessage)).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pv.pages.SwitchToPage(PageMain)
			pv.pages.RemovePage(PageDialog)
		})

	pv.pages.AddPage(PageDialog, createDialogPage(m, 50, 50), true, true)
}

func (pv *pcView) showInfo() {
	name := pv.getSelectedProcName()
	info, err := pv.project.GetProcessInfo(name)
	if err != nil {
		pv.showError(err.Error())
		return
	}
	ports, _ := pv.project.GetProcessPorts(name)
	form := pv.createProcInfoForm(info, ports)
	pv.showDialog(form, 0, 5+form.GetFormItemCount()*2)
}

func (pv *pcView) handleShutDown() {
	if pv.project.IsRemote() {
		pv.attentionMessage("Detaching...", 0)
	} else {
		pv.attentionMessage("Shutting Down...", 0)
		_ = pv.project.ShutDownProject()
	}
	pv.stopFollowLog()
	pv.appView.Stop()
	pv.cancelAppFn()
}

func (pv *pcView) handleConnectivityError() {
	if pv.project.IsRemote() {
		errSecs := pv.project.ErrorForSecs()
		if errSecs > 0 {
			pv.attentionMessage(fmt.Sprintf("Reconnecting... Terminating in %d sec", shutDownAfterSec-errSecs), 0)
		}
		if errSecs >= shutDownAfterSec {
			pv.handleShutDown()
		}
	}
}

func (pv *pcView) onProcRowSpanChange() {
	if pv.scrSplitState == ProcFull && pv.logFollow {
		pv.stopFollowLog()
	}
}

func (pv *pcView) recreateHelpDialog() {
	pv.themes.RemoveListener(pv.helpDialog)
	pv.helpDialog = newHelpDialog(pv.shortcuts, func() {
		pv.pages.RemovePage(PageDialog)
	})
	pv.themes.AddListener(pv.helpDialog)
}

func (pv *pcView) updateHelpTextView() {
	logScrBool := pv.scrSplitState != LogFull
	procScrBool := pv.scrSplitState != ProcFull
	pv.helpFooter.Clear()
	defer pv.helpFooter.AddItem(tview.NewBox(), 0, 1, false)
	if pv.logsText.isSearchActive() {
		pv.shortcuts.addButton(ActionLogFind, pv.helpFooter)
		pv.shortcuts.addButton(ActionLogFindNext, pv.helpFooter)
		pv.shortcuts.addButton(ActionLogFindPrev, pv.helpFooter)
		if config.IsLogSelectionOn() {
			pv.shortcuts.addToggleButton(ActionLogSelection, pv.helpFooter, !pv.logSelect)
		}
		pv.shortcuts.addButton(ActionLogFindExit, pv.helpFooter)
		return
	}
	pv.shortcuts.addButton(ActionHelp, pv.helpFooter)
	pv.shortcuts.addCategory("LOGS:", pv.helpFooter)
	pv.shortcuts.addToggleButton(ActionLogScreen, pv.helpFooter, logScrBool)
	pv.shortcuts.addToggleButton(ActionFollowLog, pv.helpFooter, !pv.logFollow)
	pv.shortcuts.addToggleButton(ActionWrapLog, pv.helpFooter, !pv.logsText.IsWrapOn())
	if config.IsLogSelectionOn() {
		pv.shortcuts.addToggleButton(ActionLogSelection, pv.helpFooter, !pv.logSelect)
	}
	pv.shortcuts.addButton(ActionLogFind, pv.helpFooter)
	pv.shortcuts.addCategory("PROCESS:", pv.helpFooter)
	pv.shortcuts.addButton(ActionProcessScale, pv.helpFooter)
	pv.shortcuts.addButton(ActionProcessInfo, pv.helpFooter)
	pv.shortcuts.addButton(ActionProcessStart, pv.helpFooter)
	pv.shortcuts.addToggleButton(ActionProcessScreen, pv.helpFooter, procScrBool)
	pv.shortcuts.addButton(ActionProcessStop, pv.helpFooter)
	pv.shortcuts.addButton(ActionProcessRestart, pv.helpFooter)
	pv.shortcuts.addButton(ActionQuit, pv.helpFooter)
}

func (pv *pcView) saveTuiState() {
	if pv.isReadOnlyMode {
		log.Debug().Msg("Not saving TUI state in read-only mode")
		return
	}
	pv.settings.Sort.By = columnNames[pv.stateSorter.sortByColumn]
	pv.settings.Sort.IsReversed = !pv.stateSorter.isAsc
	pv.settings.Theme = pv.styles.GetStyleName()
	err := pv.settings.Save()
	if err != nil {
		log.Error().Err(err).Msg("Failed to save settings")
	} else {
		log.Debug().Msg("Saved TUI state")
	}
}

func (pv *pcView) runOnce() {
	version, err := updater.GetLatestReleaseName()
	if err != nil {
		return
	}
	if config.Version != version {
		pv.showUpdateAvailable(version)
	}
}

func (pv *pcView) showUpdateAvailable(version string) {
	pv.appView.QueueUpdateDraw(func() {
		pv.statTable.GetCell(0, 1).SetText(fmt.Sprintf("%s  [green:]%s[-:-:-]", config.Version, version))
	})
}

func (pv *pcView) startMonitoring() {
	if !pv.project.IsRemote() {
		return
	}
	pcClient, ok := pv.project.(*client.PcClient)
	if !ok {
		return
	}
	go func(pcClient *client.PcClient) {
		isErrorDetected := false
		for {
			if err := pcClient.IsAlive(); err != nil {
				pv.handleConnectivityError()
				isErrorDetected = true
			} else {
				if isErrorDetected {
					isErrorDetected = false
					pv.hideAttentionMessage()
				}
			}
			time.Sleep(time.Second)
		}
	}(pcClient)
}

// Halt stop the application event loop.
func (pv *pcView) halt() {
	if pv.cancelFn != nil {
		pv.cancelFn()
		pv.cancelFn = nil
	}
	if pv.cancelLogFn != nil {
		pv.cancelLogFn()
		pv.cancelLogFn = nil
	}
	if pv.cancelSigFn != nil {
		pv.cancelSigFn()
		pv.cancelSigFn = nil
	}
}

// Resume restarts the app event loop.
func (pv *pcView) resume() {
	var ctxTbl context.Context
	var ctxLog context.Context
	var ctxSig context.Context
	ctxTbl, pv.cancelFn = context.WithCancel(context.Background())
	ctxLog, pv.cancelLogFn = context.WithCancel(context.Background())
	ctxSig, pv.cancelSigFn = context.WithCancel(context.Background())

	go pv.updateTable(ctxTbl)
	go pv.updateLogs(ctxLog)
	go setSignal(ctxSig)
}

func (pv *pcView) changeFocus() {
	if pv.procTable.HasFocus() {
		pv.appView.SetFocus(pv.logsText)
	} else if pv.logsText.HasFocus() {
		pv.appView.SetFocus(pv.procTable)
	}
}

func setupTui(project app.IProject, options ...Option) {

	pv := newPcView(project)
	for _, option := range options {
		if err := option(pv); err != nil {
			log.Error().Err(err).Msg("Failed to apply option")
		}
	}

	pv.resume()
	if config.CheckForUpdates == "true" {
		go pv.runOnce()
	}

	pcv = pv

}

func RunTUIAsync(project app.IProject, options ...Option) {
	setupTui(project, options...)
	go func() {
		if err := pcv.appView.Run(); err != nil {
			log.Error().Err(err).Msgf("TUI stopped")
			pcv.handleShutDown()
		}
	}()
}

func RunTUI(project app.IProject, options ...Option) {
	setupTui(project, options...)
	if err := pcv.appView.Run(); err != nil {
		log.Error().Err(err).Msgf("TUI stopped")
		pcv.handleShutDown()
	}
}

func setSignal(ctx context.Context) {
	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, os.Interrupt, syscall.SIGHUP)
	select {
	case sig := <-cancelChan:
		log.Info().Msgf("Caught %v - Shutting down the running processes...", sig)
		Stop()
		os.Exit(1)
	case <-ctx.Done():
		log.Debug().Msg("TUI Signal handler stopped")
	}
}

func Stop() {
	if pcv != nil {
		pcv.handleShutDown()
	}
}

func Wait() {
	if pcv != nil {
		<-pcv.ctxApp.Done()
		log.Debug().Msg("TUI Wait stopped")
	}
}
