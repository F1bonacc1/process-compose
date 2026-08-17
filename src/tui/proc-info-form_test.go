package tui

import (
	"slices"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rivo/tview"
)

// formLabels lists a form's item labels, which is what the dialog actually
// shows the user.
func formLabels(f *tview.Form) []string {
	labels := make([]string, 0, f.GetFormItemCount())
	for i := 0; i < f.GetFormItemCount(); i++ {
		labels = append(labels, f.GetFormItem(i).GetLabel())
	}
	return labels
}

func Test_addWatchInfo(t *testing.T) {
	triggerTime := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	watch := &types.WatchConfig{
		Paths:    []types.WatchPath{{Path: "./src"}},
		Debounce: "500ms",
		Cascade:  true,
	}

	tests := []struct {
		name    string
		watch   *types.WatchConfig
		state   *types.ProcessState
		want    []string
		notWant []string
	}{
		{
			name:    "no watch configured shows nothing",
			watch:   nil,
			state:   &types.ProcessState{},
			notWant: []string{"Watch Paths:", "Watch Debounce:", "Watch Cascade:", "Watch Armed:"},
		},
		{
			name:    "a watch block with no paths watches nothing",
			watch:   &types.WatchConfig{Debounce: "500ms"},
			state:   &types.ProcessState{},
			notWant: []string{"Watch Paths:", "Watch Debounce:"},
		},
		{
			name:  "configured watch shows its settings",
			watch: watch,
			state: &types.ProcessState{},
			want:  []string{"Watch Paths:", "Watch Debounce:", "Watch Cascade:", "Watch Armed:"},
			// No trigger recorded yet, so there is nothing to report.
			notWant: []string{"Last Watch Trigger:"},
		},
		{
			name:  "a recorded trigger is reported",
			watch: watch,
			state: &types.ProcessState{
				IsWatched:        true,
				WatchTriggerPath: "main.go",
				WatchTriggerTime: &triggerTime,
			},
			want: []string{"Watch Paths:", "Last Watch Trigger:"},
		},
		{
			name:  "without a state the config is still shown",
			watch: watch,
			state: nil,
			want:  []string{"Watch Paths:", "Watch Debounce:", "Watch Cascade:"},
			// Armed is a runtime fact, so it cannot be reported without a state.
			notWant: []string{"Watch Armed:", "Last Watch Trigger:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tview.NewForm()
			addWatchInfo(tt.watch, tt.state, f)
			labels := formLabels(f)
			for _, label := range tt.want {
				if !slices.Contains(labels, label) {
					t.Errorf("form labels = %v, want it to contain %q", labels, label)
				}
			}
			for _, label := range tt.notWant {
				if slices.Contains(labels, label) {
					t.Errorf("form labels = %v, want it NOT to contain %q", labels, label)
				}
			}
		})
	}
}

// Test_addWatchInfo_Debounce pins that the dialog reports the debounce actually
// in effect, not the raw config string - an unset debounce still has a value.
func Test_addWatchInfo_Debounce(t *testing.T) {
	tests := []struct {
		name     string
		debounce string
		want     string
	}{
		{name: "explicit", debounce: "1s", want: "1s"},
		{name: "unset falls back to the default", debounce: "", want: types.DefaultWatchDebounce.String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tview.NewForm()
			addWatchInfo(&types.WatchConfig{
				Paths:    []types.WatchPath{{Path: "./src"}},
				Debounce: tt.debounce,
			}, nil, f)

			item := f.GetFormItemByLabel("Watch Debounce:")
			if item == nil {
				t.Fatal("no debounce field in the form")
			}
			field, ok := item.(*tview.InputField)
			if !ok {
				t.Fatalf("debounce field is %T, want *tview.InputField", item)
			}
			if got := field.GetText(); got != tt.want {
				t.Errorf("debounce = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_watchPathsSummary(t *testing.T) {
	tests := []struct {
		name  string
		watch *types.WatchConfig
		want  []string
	}{
		{
			name:  "bare path",
			watch: &types.WatchConfig{Paths: []types.WatchPath{{Path: "./src"}}},
			want:  []string{"./src"},
		},
		{
			name: "filters are shown, since they explain what will not trigger",
			watch: &types.WatchConfig{Paths: []types.WatchPath{{
				Path:    "./src",
				Include: []string{"**/*.go"},
				Exclude: []string{"**/*_test.go", "bin/**"},
			}}},
			want: []string{"./src  include: **/*.go  exclude: **/*_test.go, bin/**"},
		},
		{
			name: "every root is listed",
			watch: &types.WatchConfig{Paths: []types.WatchPath{
				{Path: "./src"},
				{Path: "./assets", Exclude: []string{"*.map"}},
			}},
			want: []string{"./src", "./assets  exclude: *.map"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := watchPathsSummary(tt.watch); !slices.Equal(got, tt.want) {
				t.Errorf("watchPathsSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}
