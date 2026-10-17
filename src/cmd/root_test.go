package cmd

import (
	"testing"

	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/spf13/cobra"
)

func Test_shouldCheckForUpdates(t *testing.T) {
	saved := config.CheckForUpdates
	config.CheckForUpdates = "true"
	t.Cleanup(func() { config.CheckForUpdates = saved })

	tests := []struct {
		name string
		cmd  *cobra.Command
		want bool
	}{
		{"root", rootCmd, true},
		{"up", upCmd, true},
		{"version", versionCmd, true},
		{"version update", versionUpdateCmd, false},
		{"process logs", logsCmd, false},
		{"process logs truncate", truncateCmd, false},
		{"process list", listCmd, false},
		{"project is-ready", isReadyCmd, false},
		{"namespace list", namespaceListCmd, false},
		{"attach", attachCmd, false},
		{"down", downCmd, false},
		{"graph", graphCmd, false},
		{"analyze critical-chain", analyzeCriticalChainCmd, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldCheckForUpdates(tt.cmd); got != tt.want {
				t.Errorf("shouldCheckForUpdates(%s) = %v, want %v", tt.cmd.CommandPath(), got, tt.want)
			}
		})
	}

	t.Run("disabled at build time", func(t *testing.T) {
		config.CheckForUpdates = "false"
		defer func() { config.CheckForUpdates = "true" }()
		if shouldCheckForUpdates(upCmd) {
			t.Error("shouldCheckForUpdates(up) = true with CheckForUpdates=false")
		}
	})
}
