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
