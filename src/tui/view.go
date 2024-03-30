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

type ScrSplitState int

const (
	LogFull     ScrSplitState = 0
	ProcFull                  = 1
	LogProcHalf               = 2
)

const (
	PageMain   = "main"
	PageDialog = "dialog"
	AllNS      = "PC_ALL_NS_FILTER"
)

const shutDownAfterSec = 10

var pcv *pcView

type pcView struct {
	procTable         *tview.Table
	statTable         *tview.Table
	appView           *tview.Application
	logsText          *LogView
	statusText        *tview.TextView
	helpText          *tview.TextView
	pages             *tview.Pages
	procNames         []string
	logFollow         bool
	logSelect         bool
	scrSplitState     ScrSplitState
	loggedProc        string
	shortcuts         *ShortCuts
	procCountCell     *tview.TableCell
	mainGrid          *tview.Grid
	logsTextArea      *tview.TextArea
	project           app.IProject
	sortMtx           sync.Mutex
	stateSorter       StateSorter
	procRegex         *regexp.Regexp
	procRegexMtx      sync.Mutex
	procColumns       map[ColumnID]string
	refreshRate       time.Duration
	cancelFn          context.CancelFunc
	cancelLogFn       context.CancelFunc
	cancelSigFn       context.CancelFunc
	ctxApp            context.Context
	cancelAppFn       context.CancelFunc
	selectedNsMtx     sync.Mutex
	selectedNs        string
	selectedNsChanged atomic.Bool
	hideDisabled      atomic.Bool
	commandMode       bool
	styles            *config.Styles
	themes            *config.Themes
	helpDialog        *helpDialog
	settings          *config.Settings
	isFullScreen      bool
}

func newPcView(project app.IProject) *pcView {

	pv := &pcView{
		appView:       tview.NewApplication(),
		logsText:      NewLogView(project.GetLogLength()),
		statusText:    tview.NewTextView().SetDynamicColors(true),
		logFollow:     true,
		scrSplitState: LogProcHalf,
		helpText:      tview.NewTextView().SetDynamicColors(true),
		loggedProc:    "",
		shortcuts:     getDefaultActions(),
		procCountCell: tview.NewTableCell(""),
		mainGrid:      tview.NewGrid(),
		logsTextArea:  tview.NewTextArea(),
		logSelect:     false,
		project:       project,
		refreshRate:   time.Second,
		stateSorter: StateSorter{
			sortByColumn: ProcessStateName,
			isAsc:        true,
		},
		procColumns: map[ColumnID]string{},
		selectedNs:  AllNS,
		themes:      config.NewThemes(),
		settings:    config.NewSettings(),
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
	pv.helpDialog = newHelpDialog(pv.shortcuts, func() {
		pv.pages.RemovePage(PageDialog)
	})
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
	path := config.GetShortCutsPath()
	if len(path) > 0 {
		pv.shortcuts.loadFromFile(path)
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
		}
	})
	pv.shortcuts.setAction(ActionNsFilter, pv.showNsFilter)
	pv.shortcuts.setAction(ActionHideDisabled, func() {
		pv.hideDisabled.Store(!pv.hideDisabled.Load())
		pv.updateHelpTextView()
	})
	pv.shortcuts.setAction(ActionHelp, func() {
		pv.showDialog(pv.helpDialog, 50, 30)
	})
	pv.shortcuts.setAction(ActionThemeSelector, pv.showThemeSelector)
	pv.shortcuts.setAction(ActionSendToBackground, pv.runShellProcess)
	pv.shortcuts.setAction(ActionFullScreen, func() {
		pv.isFullScreen = !pv.isFullScreen
		pv.logsText.SetBorder(!pv.isFullScreen)
		pv.redrawGrid()
	})
	pv.shortcuts.setAction(ActionFocusChange, pv.changeFocus)
	pv.shortcuts.setAction(ActionProcFilter, func() {
		pv.commandMode = true
		pv.redrawGrid()
	})
}

func (pv *pcView) loadThemes() {
	pv.themes.AddListener(pv)
	pv.themes.AddListener(pv.helpDialog)
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

	result := "This will terminate all the running processes."
	if pv.project.IsRemote() {
		result = ""
	}
	m := tview.NewModal().
		SetText("Are you sure you want to quit?\n" + result).
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Quit" {
				go pv.handleShutDown()
			}
			pv.pages.SwitchToPage(PageMain)
			pv.pages.RemovePage(PageDialog)
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
	pv.showDialog(form, 0, 0)
}

func (pv *pcView) handleShutDown() {
	pv.attentionMessage("Shutting Down...")
	if !pv.project.IsRemote() {
		_ = pv.project.ShutDownProject()
	}
	time.Sleep(time.Second)
	pv.stopFollowLog()
	pv.appView.Stop()
	pv.cancelAppFn()
}
func (pv *pcView) attentionMessage(message string) {
	pv.statTable.SetCell(0, 2, tview.NewTableCell(message).
		SetSelectable(false).
		SetAlign(tview.AlignCenter).
		SetExpansion(0).
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorRed))
}

func (pv *pcView) hideAttentionMessage() {
	pv.statTable.SetCell(0, 2, tview.NewTableCell(""))
}

func (pv *pcView) handleConnectivityError() {
	if pv.project.IsRemote() {
		errSecs := pv.project.ErrorForSecs()
		if errSecs > 0 {
			pv.attentionMessage(fmt.Sprintf("Reconnecting... Terminating in %d sec", shutDownAfterSec-errSecs))
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

func (pv *pcView) updateHelpTextView() {
	logScrBool := pv.scrSplitState != LogFull
	procScrBool := pv.scrSplitState != ProcFull
	pv.helpText.Clear()
	if pv.logsText.isSearchActive() {
		pv.shortcuts.writeButton(ActionLogFind, pv.helpText)
		pv.shortcuts.writeButton(ActionLogFindNext, pv.helpText)
		pv.shortcuts.writeButton(ActionLogFindPrev, pv.helpText)
		if config.IsLogSelectionOn() {
			pv.shortcuts.writeToggleButton(ActionLogSelection, pv.helpText, !pv.logSelect)
		}
		pv.shortcuts.writeButton(ActionLogFindExit, pv.helpText)
		return
	}
	pv.shortcuts.writeButton(ActionHelp, pv.helpText)
	pv.shortcuts.writeCategory("LOGS:", pv.helpText)
	pv.shortcuts.writeToggleButton(ActionLogScreen, pv.helpText, logScrBool)
	pv.shortcuts.writeToggleButton(ActionFollowLog, pv.helpText, !pv.logFollow)
	pv.shortcuts.writeToggleButton(ActionWrapLog, pv.helpText, !pv.logsText.IsWrapOn())
	if config.IsLogSelectionOn() {
		pv.shortcuts.writeToggleButton(ActionLogSelection, pv.helpText, !pv.logSelect)
	}
	pv.shortcuts.writeButton(ActionLogFind, pv.helpText)
	//fmt.Fprintf(pv.helpText, "%s ", "[lightskyblue::b]PROCESS:[-:-:-]")
	pv.shortcuts.writeCategory("PROCESS:", pv.helpText)
	pv.shortcuts.writeButton(ActionProcessScale, pv.helpText)
	pv.shortcuts.writeButton(ActionProcessInfo, pv.helpText)
	pv.shortcuts.writeButton(ActionProcessStart, pv.helpText)
	pv.shortcuts.writeToggleButton(ActionProcessScreen, pv.helpText, procScrBool)
	pv.shortcuts.writeButton(ActionProcessStop, pv.helpText)
	pv.shortcuts.writeButton(ActionProcessRestart, pv.helpText)
	pv.shortcuts.writeButton(ActionQuit, pv.helpText)
}

func (pv *pcView) saveTuiState() {
	pv.settings.Sort.By = columnNames[pv.stateSorter.sortByColumn]
	pv.settings.Sort.IsReversed = !pv.stateSorter.isAsc
	pv.settings.Theme = pv.styles.GetStyleName()
	err := pv.settings.Save()
	if err != nil {
		log.Error().Err(err).Msg("Failed to save settings")
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

func SetupTui(project app.IProject, options ...Option) {

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
	if err := pv.appView.Run(); err != nil {
		panic(err)
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
	}
}
