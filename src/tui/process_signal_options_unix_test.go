//go:build !windows

package tui

import "testing"

func TestAvailableSignalOptionsIncludeCommonSignals(t *testing.T) {
	options := availableSignalOptions()
	found := map[string]bool{}
	for _, option := range options {
		found[option.Name] = true
	}

	for _, signalName := range []string{"SIGTERM", "SIGKILL", "SIGALRM"} {
		if !found[signalName] {
			t.Fatalf("expected %s to be available in the signal picker", signalName)
		}
	}
}
