package tui

import (
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
)

func (pv *pcView) createProcInfoForm(info *types.ProcessConfig) *tview.Form {
	f := tview.NewForm()
	f.SetCancelFunc(func() {
		pv.pages.RemovePage(PageDialog)
	})
	f.SetItemPadding(1)
	f.SetBorder(true)
	f.SetFieldBackgroundColor(tcell.ColorLightSkyBlue)
	f.SetFieldTextColor(tcell.ColorBlack)
	f.SetButtonsAlign(tview.AlignCenter)
	f.SetTitle("Process " + info.Name + " Info")
	addStringIfNotEmpty("Command:", info.Command, f)
	addStringIfNotEmpty("Working Directory:", info.WorkingDir, f)
	addStringIfNotEmpty("Log Location:", info.LogLocation, f)
	addSliceIfNotEmpty("Environment:", info.Environment, f)
	addSliceIfNotEmpty("Depends On:", mapKeysToSlice(info.DependsOn), f)
	f.AddCheckbox("Is Disabled:", info.Disabled, nil)
	f.AddCheckbox("Is Daemon:", info.IsDaemon, nil)
	f.AddButton("Close", func() {
		pv.pages.RemovePage(PageDialog)
	})
	f.SetFocus(f.GetFormItemCount())
	return f
}

func addStringIfNotEmpty(label, value string, f *tview.Form) {
	if len(strings.TrimSpace(value)) > 0 {
		f.AddInputField(label, value, 0, nil, nil)
	}
}

func addSliceIfNotEmpty(label string, value []string, f *tview.Form) {
	if len(value) > 0 {
		f.AddDropDown(label, value, 0, nil)
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
