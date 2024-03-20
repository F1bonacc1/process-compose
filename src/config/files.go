package config

import (
	"embed"
)

var (

	////go:embed templates/hotkeys.yaml
	// hotkeysTpl tracks hotkeys default config template
	//hotkeysTpl []byte

	//go:embed themes/*-theme.yaml
	themesFolder embed.FS
)
