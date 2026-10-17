package tui

import (
	"testing"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rivo/tview"
)

func TestMemToString(t *testing.T) {
	type args struct {
		mem int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "negative",
			args: args{
				mem: -1,
			},
			want: "unknown",
		},
		{
			name: "zero",
			args: args{
				mem: 0,
			},
			want: "-",
		},
		{
			name: "bytes",
			args: args{
				mem: 345,
			},
			want: "0.0 MiB",
		},
		{
			name: "kilobytes",
			args: args{
				mem: 1024 * 513,
			},
			want: "0.5 MiB",
		},
		{
			name: "megabytes",
			args: args{
				mem: 1024*1024*7 + 1024*513,
			},
			want: "7.5 MiB",
		},
		{
			name: "gigabytes",
			args: args{
				mem: 1024*1024*1024*4 + 1024*1024*740,
			},
			want: "4.7 GiB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStrForMem(tt.args.mem, true); got != tt.want {
				t.Errorf("getStrForMem() = %v, want %v", got, tt.want)
			}
		})
	}
}

// subscribeCountingProject counts log subscriptions, so a test can tell
// whether the selection-changed callback, which re-subscribes, has run.
type subscribeCountingProject struct {
	*logProject
	subscribeCalls int
}

func (p *subscribeCountingProject) GetLogsAndSubscribe(name string, observer pclog.LogObserver) error {
	p.subscribeCalls++
	return p.logProject.GetLogsAndSubscribe(name, observer)
}

func newProcTableTestView(project app.IProject) *pcView {
	pv := &pcView{
		appView:       tview.NewApplication(),
		logsText:      NewLogView(1000),
		helpFooter:    tview.NewFlex(),
		mainGrid:      tview.NewGrid(),
		statTable:     tview.NewTable(),
		shortcuts:     newShortCuts(),
		monitor:       newProcessMonitor(),
		watchNotifier: newWatchNotifier(),
		styles:        config.NewStyles(),
		project:       project,
		selectedNs:    AllNS,
		logFollow:     true,
		stateSorter:   StateSorter{sortByColumn: ProcessStateMem, isAsc: true},
	}
	pv.termView = NewTerminalView(pv.appView)
	pv.logsText.setTruncator(pv)
	pv.setShortCutsActions()
	pv.procTable = pv.createProcTable()
	return pv
}

func memSortedStates(alphaMem, betaMem int64) *types.ProcessesState {
	return &types.ProcessesState{States: []types.ProcessState{
		{Name: "alpha", Status: types.ProcessStateRunning, IsRunning: true, Mem: alphaMem},
		{Name: "beta", Status: types.ProcessStateRunning, IsRunning: true, Mem: betaMem},
		{Name: "tool-a", Status: types.ProcessStateDisabled},
	}}
}

// When a refresh moves the selected process to another row (e.g. sorted by
// MEM and its memory grew), the cursor must move with it, so that actions
// like start/stop target the process the user picked (#547).
func TestFillTableData_SelectionFollowsProcessAcrossReorder(t *testing.T) {
	project := &subscribeCountingProject{logProject: newLogProject(true)}
	pv := newProcTableTestView(project)

	pv.procStates = memSortedStates(100, 200) // tool-a, alpha, beta
	pv.fillTableData()
	pv.selectProcForTest("alpha")
	subscribed := project.subscribeCalls

	pv.procStates = memSortedStates(300, 200) // tool-a, beta, alpha
	pv.fillTableData()

	if got := pv.getSelectedProcName(); got != "alpha" {
		t.Fatalf("selected process after reorder = %q, want %q", got, "alpha")
	}
	if row, _ := pv.procTable.GetSelection(); row != 3 {
		t.Errorf("selected row after reorder = %d, want 3", row)
	}
	if project.subscribeCalls != subscribed {
		t.Errorf("refresh re-ran the selection callback: %d log subscriptions, want %d", project.subscribeCalls, subscribed)
	}
}
