package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rivo/tview"
)

// logProject implements the subset of app.IProject the log pane uses; any
// other method call panics via the embedded nil interface.
//
// With syncBacklog set, GetLogsAndSubscribe hands the backlog to the observer
// before returning, like the local ProjectRunner. Without it, nothing is
// delivered synchronously, like the remote client, whose backlog arrives later
// over a websocket.
type logProject struct {
	app.IProject
	syncBacklog bool
	logs        map[string][]string
	subscribed  map[string]bool
}

func newLogProject(syncBacklog bool) *logProject {
	return &logProject{
		syncBacklog: syncBacklog,
		logs: map[string][]string{
			"alpha": {"alpha 0", "alpha 1"},
			"beta":  {"beta 0", "beta 1", "beta 2"},
		},
		subscribed: map[string]bool{},
	}
}

func (p *logProject) IsRemote() bool {
	return !p.syncBacklog
}

func (p *logProject) GetProcessInfo(name string) (*types.ProcessConfig, error) {
	return &types.ProcessConfig{Name: name}, nil
}

func (p *logProject) GetProcessState(name string) (*types.ProcessState, error) {
	return &types.ProcessState{Name: name}, nil
}

func (p *logProject) GetLogLength() int {
	return 1000
}

func (p *logProject) GetProcessLog(name string, _, _ int) ([]string, error) {
	lines, ok := p.logs[name]
	if !ok {
		return nil, fmt.Errorf("process %s doesn't exist", name)
	}
	return lines, nil
}

func (p *logProject) GetLogsAndSubscribe(name string, observer pclog.LogObserver) error {
	p.subscribed[name] = true
	if p.syncBacklog {
		observer.SetLines(p.logs[name])
	}
	return nil
}

func (p *logProject) UnSubscribeLogger(name string, _ pclog.LogObserver) error {
	delete(p.subscribed, name)
	return nil
}

func newLogTestView(project *logProject, names ...string) *pcView {
	pv := &pcView{
		appView:    tview.NewApplication(),
		logsText:   NewLogView(1000),
		helpFooter: tview.NewFlex(),
		shortcuts:  newShortCuts(),
		monitor:    newProcessMonitor(),
		project:    project,
		procTable:  tview.NewTable().SetSelectable(true, false),
	}
	pv.termView = NewTerminalView(pv.appView)
	pv.logsText.setTruncator(pv)
	pv.setShortCutsActions()
	for i, name := range names {
		pv.procTable.SetCell(i+1, int(ProcessStateName), tview.NewTableCell(name))
	}
	return pv
}

func (pv *pcView) selectProcForTest(name string) {
	pv.selectTableProcess(name)
	pv.onTableSelectionChange(0, 0)
}

func logPaneLines(pv *pcView) string {
	return strings.TrimSpace(pv.logsText.GetText(true))
}

var logProjectModes = []struct {
	name        string
	syncBacklog bool
}{
	{name: "local", syncBacklog: true},
	{name: "remote", syncBacklog: false},
}

func TestSelectionChange_NotFollowing_LoadsBacklogWithoutSubscribing(t *testing.T) {
	for _, mode := range logProjectModes {
		t.Run(mode.name, func(t *testing.T) {
			project := newLogProject(mode.syncBacklog)
			pv := newLogTestView(project, "alpha", "beta")
			pv.selectProcForTest("alpha")

			pv.logFollow = false
			pv.selectProcForTest("beta")

			if got, want := logPaneLines(pv), "beta 0\nbeta 1\nbeta 2"; got != want {
				t.Errorf("log pane = %q, want %q", got, want)
			}
			if len(project.subscribed) != 0 {
				t.Errorf("expected no log subscriptions while not following, got %v", project.subscribed)
			}
		})
	}
}

func TestSelectionChange_Following_Subscribes(t *testing.T) {
	for _, mode := range logProjectModes {
		t.Run(mode.name, func(t *testing.T) {
			project := newLogProject(mode.syncBacklog)
			pv := newLogTestView(project, "alpha", "beta")
			pv.logFollow = true
			pv.selectProcForTest("alpha")
			pv.selectProcForTest("beta")

			if len(project.subscribed) != 1 || !project.subscribed["beta"] {
				t.Errorf("expected a single subscription to beta, got %v", project.subscribed)
			}
		})
	}
}

func TestPrettyPrintToggle_NotFollowing_ReloadsBacklogWithoutSubscribing(t *testing.T) {
	for _, mode := range logProjectModes {
		t.Run(mode.name, func(t *testing.T) {
			project := newLogProject(mode.syncBacklog)
			pv := newLogTestView(project, "alpha", "beta")
			pv.logFollow = false
			pv.selectProcForTest("beta")

			pv.shortcuts.ShortCutKeys[ActionLogPrettyPrint].actionFn()

			if got, want := logPaneLines(pv), "beta 0\nbeta 1\nbeta 2"; got != want {
				t.Errorf("log pane = %q, want %q", got, want)
			}
			if len(project.subscribed) != 0 {
				t.Errorf("expected no log subscriptions while not following, got %v", project.subscribed)
			}
		})
	}
}
