package config

import "testing"

func TestDisableUpdateCheckDefault(t *testing.T) {
	if *NewFlags().DisableUpdateCheck {
		t.Fatalf("update check disabled without %s set", EnvVarDisableUpdateCheck)
	}
	t.Setenv(EnvVarDisableUpdateCheck, "1")
	if !*NewFlags().DisableUpdateCheck {
		t.Fatalf("%s=1 did not disable the update check", EnvVarDisableUpdateCheck)
	}
}
