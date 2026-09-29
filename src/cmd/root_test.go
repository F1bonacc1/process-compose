package cmd

import (
	"path/filepath"
	"testing"

	"github.com/f1bonacc1/process-compose/src/config"
)

func TestDisableUpdateCheckFlag(t *testing.T) {
	origCheck, origFlag := config.CheckForUpdates, *pcFlags.DisableUpdateCheck
	t.Cleanup(func() {
		config.CheckForUpdates = origCheck
		*pcFlags.DisableUpdateCheck = origFlag
		updateMsg = nil
		rootCmd.SetArgs(nil)
	})

	// Simulate a release build, which checks for updates by default.
	config.CheckForUpdates = "true"
	updateMsg = nil
	rootCmd.SetArgs([]string{"version", "--short", "--disable-update-check",
		"--log-file", filepath.Join(t.TempDir(), "pc.log")})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if updateMsg != nil {
		t.Error("update check started despite --disable-update-check")
	}
	if config.CheckForUpdates != "false" {
		t.Errorf("CheckForUpdates = %q, want %q so the TUI skips its check too", config.CheckForUpdates, "false")
	}
}
