package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestDefaultLogCopyShortcut(t *testing.T) {
	sc := newShortCuts()
	action, found := sc.ShortCutKeys[ActionLogCopy]
	if !found {
		t.Fatalf("expected default shortcuts to contain %q", ActionLogCopy)
	}
	if action.key != tcell.KeyCR {
		t.Errorf("expected default log_copy key to be Enter (KeyCR), got %v", action.key)
	}
	if action.rune != 0 {
		t.Errorf("expected default log_copy rune to be unset, got %q", action.rune)
	}
	if action.ShortCut != tcell.KeyNames[tcell.KeyCR] {
		t.Errorf("expected default log_copy label %q, got %q", tcell.KeyNames[tcell.KeyCR], action.ShortCut)
	}
}

func TestLogCopyShortcutOverride(t *testing.T) {
	tests := []struct {
		name     string
		shortcut string
		wantKey  tcell.Key
		wantRune rune
	}{
		{name: "rebind to a single rune (vim-style yank)", shortcut: "y", wantKey: 0, wantRune: 'y'},
		{name: "rebind to a key combination", shortcut: "Ctrl-Y", wantKey: tcell.KeyCtrlY, wantRune: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "shortcuts.yaml")
			yaml := "shortcuts:\n  log_copy:\n    shortcut: \"" + tt.shortcut + "\"\n"
			if err := os.WriteFile(path, []byte(yaml), 0600); err != nil {
				t.Fatalf("failed to write shortcuts file: %v", err)
			}

			sc := newShortCuts()
			if err := sc.loadFromFile(path); err != nil {
				t.Fatalf("loadFromFile failed: %v", err)
			}

			action := sc.ShortCutKeys[ActionLogCopy]
			if action == nil {
				t.Fatalf("log_copy action missing after override")
			}
			if action.key != tt.wantKey {
				t.Errorf("expected key %v, got %v", tt.wantKey, action.key)
			}
			if action.rune != tt.wantRune {
				t.Errorf("expected rune %q, got %q", tt.wantRune, action.rune)
			}
		})
	}
}
