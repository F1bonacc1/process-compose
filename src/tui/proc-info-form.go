package tui

import (
	"fmt"
	"time"

	"strings"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rivo/tview"
)

func (pv *pcView) createProcInfoForm(info *types.ProcessConfig, state *types.ProcessState, ports *types.ProcessPorts) *tview.Form {
	f := tview.NewForm()
	f.SetCancelFunc(func() {
		pv.pages.RemovePage(PageDialog)
	})
	f.SetItemPadding(1)
	f.SetBorder(true)
	f.SetButtonsAlign(tview.AlignCenter)
	f.SetTitle("Process " + info.Name + " Info")
	addStringIfNotEmpty("Description:", info.Description, f)
	addStringIfNotEmpty("Entrypoint:", strings.Join(info.Entrypoint, " "), f)
	addStringIfNotEmpty("Command:", info.Command, f)
	addStringIfNotEmpty("Working Directory:", info.WorkingDir, f)
	addStringIfNotEmpty("Log Location:", info.LogLocation, f)
	f.AddInputField("Replica:", fmt.Sprintf("%d/%d", info.ReplicaNum+1, info.Replicas), 0, nil, nil)
	// Display next run time for scheduled processes
	if state != nil && state.NextRunTime != nil {
		f.AddInputField("Next Run:", state.NextRunTime.Format(time.RFC1123), 0, nil, nil)
	}
	addDropDownIfNotEmpty("Environment:", info.Environment, f)
	addCSVIfNotEmpty("Depends On:", mapKeysToSlice(info.DependsOn), f)
	if ports != nil {
		addCSVIfNotEmpty("TCP Ports:", ports.TcpPorts, f)
		addCSVIfNotEmpty("UDP Ports:", ports.UdpPorts, f)
	}
	addWatchInfo(info.Watch, state, f)
	f.AddCheckbox("Is Disabled:", info.Disabled, nil)
	f.AddCheckbox("Is Daemon:", info.IsDaemon, nil)
	f.AddCheckbox("Is TTY:", info.IsTty, nil)
	f.AddCheckbox("Is Elevated:", info.IsElevated, nil)
	f.AddButton("Close", func() {
		pv.pages.RemovePage(PageDialog)
	})
	f.SetFocus(f.GetFormItemCount())
	pv.styleForm(f)
	return f
}

// addWatchInfo describes a process's file watch. The fields are kept together
// rather than folded into the boolean group below, because "cascade" and
// "armed" only mean anything next to the paths they apply to.
//
// The armed flag and the last trigger come from the state rather than the
// config, so they are reported in an attached session too.
func addWatchInfo(watch *types.WatchConfig, state *types.ProcessState, f *tview.Form) {
	if !watch.IsEnabled() {
		return
	}
	addDropDownIfNotEmpty("Watch Paths:", watchPathsSummary(watch), f)
	f.AddInputField("Watch Debounce:", watch.GetDebounce().String(), 0, nil, nil)
	f.AddCheckbox("Watch Cascade:", watch.Cascade, nil)
	if state == nil {
		return
	}
	// Armed is not the same as configured: a stopped process keeps its watch
	// config but cannot be restarted by a file change, and a watch suspended as
	// a feedback loop reads false here too.
	f.AddCheckbox("Watch Armed:", state.IsWatched, nil)
	if state.WatchTriggerPath != "" {
		trigger := state.WatchTriggerPath
		if state.WatchTriggerTime != nil {
			trigger = fmt.Sprintf("%s (%s)", trigger, state.WatchTriggerTime.Format(time.RFC1123))
		}
		f.AddInputField("Last Watch Trigger:", trigger, 0, nil, nil)
	}
}

// watchPathsSummary renders each watched root together with its filters, so the
// dialog can answer "why did this path not trigger" and not merely "what is
// watched".
func watchPathsSummary(watch *types.WatchConfig) []string {
	summary := make([]string, 0, len(watch.Paths))
	for _, watchPath := range watch.Paths {
		entry := watchPath.Path
		if len(watchPath.Include) > 0 {
			entry += "  include: " + strings.Join(watchPath.Include, ", ")
		}
		if len(watchPath.Exclude) > 0 {
			entry += "  exclude: " + strings.Join(watchPath.Exclude, ", ")
		}
		summary = append(summary, entry)
	}
	return summary
}

func addStringIfNotEmpty(label, value string, f *tview.Form) {
	if len(strings.TrimSpace(value)) > 0 {
		f.AddInputField(label, value, 0, nil, nil)
	}
}

func addDropDownIfNotEmpty(label string, value []string, f *tview.Form) {
	if len(value) > 0 {
		f.AddDropDown(label, value, 0, nil)
	}
}

func addCSVIfNotEmpty[K comparable](label string, value []K, f *tview.Form) {
	if len(value) > 0 {
		csvPorts := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(value)), ":"), "[]")
		f.AddInputField(label, csvPorts, 0, nil, nil)
	}
}

// mapKeysToSlice extract keys of map as slice,
func mapKeysToSlice[K comparable, V any](m map[K]V) []K {
	keys := make([]K, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}
