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
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/rivo/tview"
)

type FullScrState int

const (
	LogFull     FullScrState = 0
	ProcFull                 = 1
	LogProcHalf              = 2
)

const (
	PageMain   = "main"
	PageDialog = "dialog"
	AllNS      = "PC_ALL_NS_FILTER"
)

const shutDownAfterSec = 10

var pcv *pcView = nil

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
	fullScrState      FullScrState
	loggedProc        string
	shortcuts         ShortCuts
	procCountCell     *tview.TableCell
	mainGrid          *tview.Grid
	logsTextArea      *tview.TextArea
	project           app.IProject
	sortMtx           sync.Mutex
	stateSorter       StateSorter
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
}

func newPcView(project app.IProject) *pcView {
	//_ = pv.shortcuts.loadFromFile("short-cuts-new.yaml")
	pv := &pcView{
		appView:       tview.NewApplication(),
		logsText:      NewLogView(project.GetLogLength()),
		statusText:    tview.NewTextView().SetDynamicColors(true),
		logFollow:     true,
		fullScrState:  LogProcHalf,
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
	}
	pv.ctxApp, pv.cancelAppFn = context.WithCancel(context.Background())
	pv.statTable = pv.createStatTable()
	go pv.loadProcNames()
	pv.startMonitoring()
	pv.loadShortcuts()
	pv.procTable = pv.createProcTable()
	pv.updateHelpTextView()
	pv.createGrid()
	pv.createLogSelectionTextArea()
	pv.pages = tview.NewPages().
		AddPage(PageMain, pv.mainGrid, true, true)

	pv.mainGrid.SetInputCapture(pv.onMainGridKey)
	pv.appView.SetRoot(pv.pages, true).EnableMouse(true).SetInputCapture(pv.onAppKey)

	if len(pv.procNames) > 0 {
		name := pv.procNames[0]
		pv.logsText.SetTitle(name)
		pv.followLog(name)
	}
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
	case pv.shortcuts.ShortCutKeys[ActionQuit].key:
		pv.terminateAppView()
	case pv.shortcuts.ShortCutKeys[ActionLogScreen].key:
		if pv.fullScrState == LogFull {
			pv.fullScrState = LogProcHalf
		} else {
			pv.fullScrState = LogFull
		}
		pv.redrawGrid()
		pv.updateHelpTextView()
	case pv.shortcuts.ShortCutKeys[ActionFollowLog].key:
		pv.toggleLogFollow()
	case pv.shortcuts.ShortCutKeys[ActionWrapLog].key:
		pv.logsText.ToggleWrap()
		pv.updateHelpTextView()
	case pv.shortcuts.ShortCutKeys[ActionLogSelection].key:
		if !config.IsLogSelectionOn() {
			return event
		}
		pv.stopFollowLog()
		pv.toggleLogSelection()
		pv.appView.SetFocus(pv.logsTextArea)
		pv.updateHelpTextView()
	case pv.shortcuts.ShortCutKeys[ActionProcessScreen].key:
		if pv.fullScrState == ProcFull {
			pv.fullScrState = LogProcHalf
		} else {
			pv.fullScrState = ProcFull
		}
		pv.redrawGrid()
		pv.onProcRowSpanChange()
		pv.updateHelpTextView()
	case pv.shortcuts.ShortCutKeys[ActionProcessScale].key:
		pv.showScale()
	case pv.shortcuts.ShortCutKeys[ActionProcessInfo].key:
		pv.showInfo()
	case pv.shortcuts.ShortCutKeys[ActionLogFind].key:
		pv.showSearch()
	case pv.shortcuts.ShortCutKeys[ActionLogFindNext].key:
		pv.logsText.SearchNext()
		pv.logsText.SetTitle(pv.getLogTitle(pv.getSelectedProcName()))
	case pv.shortcuts.ShortCutKeys[ActionLogFindPrev].key:
		pv.logsText.SearchPrev()
		pv.logsText.SetTitle(pv.getLogTitle(pv.getSelectedProcName()))
	case pv.shortcuts.ShortCutKeys[ActionLogFindExit].key:
		pv.exitSearch()
	case pv.shortcuts.ShortCutKeys[ActionNsFilter].key:
		pv.showNsFilter()
	default:
		return event
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

func (pv *pcView) showScale() {
	f := tview.NewForm()
	f.SetCancelFunc(func() {
		pv.pages.RemovePage(PageDialog)
	})
	f.SetItemPadding(3)
	f.SetBorder(true)
	f.SetFieldBackgroundColor(tcell.ColorBlack)
	f.SetFieldTextColor(tcell.ColorLightSkyBlue)
	name := pv.getSelectedProcName()
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
	pv.showDialog(f, 60, 10)
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

func (pv *pcView) getSelectedProcName() string {
	if pv.procTable == nil {
		return ""
	}
	row, _ := pv.procTable.GetSelection()
	if row > 0 {
		return pv.procTable.GetCell(row, int(ProcessStateName)).Text
	}
	return ""
}

func (pv *pcView) onProcRowSpanChange() {
	if pv.fullScrState == ProcFull && pv.logFollow {
		pv.stopFollowLog()
	}
}

func (pv *pcView) updateHelpTextView() {
	logScrBool := pv.fullScrState != LogFull
	procScrBool := pv.fullScrState != ProcFull
	pv.helpText.Clear()
	if pv.logsText.isSearchActive() {
		pv.shortcuts.ShortCutKeys[ActionLogFind].writeButton(pv.helpText)
		pv.shortcuts.ShortCutKeys[ActionLogFindNext].writeButton(pv.helpText)
		pv.shortcuts.ShortCutKeys[ActionLogFindPrev].writeButton(pv.helpText)
		if config.IsLogSelectionOn() {
			pv.shortcuts.ShortCutKeys[ActionLogSelection].writeToggleButton(pv.helpText, !pv.logSelect)
		}
		pv.shortcuts.ShortCutKeys[ActionLogFindExit].writeButton(pv.helpText)
		return
	}
	fmt.Fprintf(pv.helpText, "%s ", "[lightskyblue:]LOGS:[-:-:-]")
	pv.shortcuts.ShortCutKeys[ActionLogScreen].writeToggleButton(pv.helpText, logScrBool)
	pv.shortcuts.ShortCutKeys[ActionFollowLog].writeToggleButton(pv.helpText, !pv.logFollow)
	pv.shortcuts.ShortCutKeys[ActionWrapLog].writeToggleButton(pv.helpText, !pv.logsText.IsWrapOn())
	if config.IsLogSelectionOn() {
		pv.shortcuts.ShortCutKeys[ActionLogSelection].writeToggleButton(pv.helpText, !pv.logSelect)
	}
	pv.shortcuts.ShortCutKeys[ActionLogFind].writeButton(pv.helpText)
	fmt.Fprintf(pv.helpText, "%s ", "[lightskyblue::b]PROCESS:[-:-:-]")
	pv.shortcuts.ShortCutKeys[ActionProcessScale].writeButton(pv.helpText)
	pv.shortcuts.ShortCutKeys[ActionProcessInfo].writeButton(pv.helpText)
	pv.shortcuts.ShortCutKeys[ActionProcessStart].writeButton(pv.helpText)
	pv.shortcuts.ShortCutKeys[ActionProcessScreen].writeToggleButton(pv.helpText, procScrBool)
	pv.shortcuts.ShortCutKeys[ActionProcessStop].writeButton(pv.helpText)
	pv.shortcuts.ShortCutKeys[ActionProcessRestart].writeButton(pv.helpText)
	pv.shortcuts.ShortCutKeys[ActionNsFilter].writeButton(pv.helpText)
	pv.shortcuts.ShortCutKeys[ActionQuit].writeButton(pv.helpText)
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
