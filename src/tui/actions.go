package tui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"strings"
)

type ActionName string

const (
	ActionHelp           = ActionName("help")
	ActionLogScreen      = ActionName("log_screen")
	ActionFollowLog      = ActionName("log_follow")
	ActionWrapLog        = ActionName("log_wrap")
	ActionLogSelection   = ActionName("log_select")
	ActionProcessStart   = ActionName("process_start")
	ActionProcessScale   = ActionName("process_scale")
	ActionProcessInfo    = ActionName("process_info")
	ActionProcessStop    = ActionName("process_stop")
	ActionProcessRestart = ActionName("process_restart")
	ActionProcessScreen  = ActionName("process_screen")
	ActionQuit           = ActionName("quit")
	ActionLogFind        = ActionName("find")
	ActionLogFindNext    = ActionName("find_next")
	ActionLogFindPrev    = ActionName("find_prev")
	ActionLogFindExit    = ActionName("find_exit")
	ActionNsFilter       = ActionName("ns_filter")
	ActionHideDisabled   = ActionName("hide_disabled")
	ActionProcFilter     = ActionName("proc_filter")
)

var defaultShortcuts = map[ActionName]tcell.Key{
	ActionHelp:           tcell.KeyF1,
	ActionLogScreen:      tcell.KeyF4,
	ActionFollowLog:      tcell.KeyF5,
	ActionWrapLog:        tcell.KeyF6,
	ActionLogSelection:   tcell.KeyCtrlS,
	ActionProcessScale:   tcell.KeyF2,
	ActionProcessInfo:    tcell.KeyF3,
	ActionProcessStart:   tcell.KeyF7,
	ActionProcessStop:    tcell.KeyF9,
	ActionProcessRestart: tcell.KeyCtrlR,
	ActionProcessScreen:  tcell.KeyF8,
	ActionQuit:           tcell.KeyF10,
	ActionLogFind:        tcell.KeyCtrlF,
	ActionLogFindNext:    tcell.KeyCtrlN,
	ActionLogFindPrev:    tcell.KeyCtrlP,
	ActionLogFindExit:    tcell.KeyEsc,
	ActionNsFilter:       tcell.KeyCtrlG,
	ActionHideDisabled:   tcell.KeyCtrlD,
	ActionProcFilter:     tcell.KeyRune,
}

var defaultShortcutsRunes = map[ActionName]rune{
	ActionProcFilter: '/',
}

var logActionsOrder = []ActionName{
	ActionLogScreen,
	ActionFollowLog,
	ActionWrapLog,
	ActionLogSelection,
	ActionLogFind,
}

var procActionsOrder = []ActionName{
	ActionProcFilter,
	ActionProcessScale,
	ActionProcessInfo,
	ActionProcessStart,
	ActionProcessScreen,
	ActionProcessStop,
	ActionProcessRestart,
	ActionNsFilter,
	ActionHideDisabled,
	ActionQuit,
}

type ShortCuts struct {
	ShortCutKeys map[ActionName]*Action `yaml:"shortcuts"`
}

func (s *ShortCuts) saveToFile(filePath string) error {
	data, err := yaml.Marshal(&s)
	if err != nil {
		log.Error().Msgf("Failed to marshal actions - %s", err.Error())
		return err
	}
	err = os.WriteFile(filePath, data, 0600)

	if err != nil {
		log.Error().Msgf("Failed to save file %s - %s", filePath, err.Error())
		return err
	}
	return nil
}

func (s *ShortCuts) loadFromFile(filePath string) error {
	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Error().Msgf("Failed to load shortcuts - %s", err.Error())
		return err
	}

	var sc ShortCuts
	err = yaml.Unmarshal(file, &sc)
	if err != nil {
		log.Error().Msgf("Failed to unmarshal file %s - %s", filePath, err.Error())
		return err
	}

	parseShortCuts(&sc)
	s.applyValid(&sc)
	log.Debug().Msgf("Shortcuts loaded from %s", filePath)
	return nil
}

func (s *ShortCuts) applyValid(newSc *ShortCuts) {
	for actionName, action := range newSc.ShortCutKeys {
		if oldAction, found := s.ShortCutKeys[actionName]; found {
			oldAction.ShortCut = action.ShortCut
			oldAction.key = action.key
			if len(strings.TrimSpace(action.Description)) > 0 {
				oldAction.Description = action.Description
			}
			if len(action.ToggleDescription) == 2 {
				oldAction.ToggleDescription = action.ToggleDescription
			}
		} else {
			log.Error().Msgf("Invalid action '%s' shortcut", actionName)
		}

	}
}

func parseShortCuts(sc *ShortCuts) {
	for k, v := range sc.ShortCutKeys {
		key, err := keyName2Key(v.ShortCut)
		if err != nil {
			assignDefaultKeys(k, v)
			log.Error().Msgf("Failed in parsing '%s' shortcut - %s. Using default: %s", k, err.Error(), v.ShortCut)
			continue
		}
		v.key = key
	}
}

func keyName2Key(key string) (tcell.Key, error) {
	for k, v := range tcell.KeyNames {
		if key == v {
			return k, nil
		}
	}

	return 0, fmt.Errorf("no matching key found %s", key)
}

type Action struct {
	Description       string          //`yaml:"description,omitempty"`
	ToggleDescription map[bool]string //`yaml:"toggle_description,omitempty"`
	ShortCut          string          `yaml:"shortcut"`
	key               tcell.Key
	rune              rune
}

func (a Action) getButton() string {
	return fmt.Sprintf("%s[black:green]%s[-:-:-]", a.ShortCut, a.Description)
}

func (a Action) getToggleButton(state bool) string {
	if len(a.ToggleDescription) != 2 {
		return a.getButton()
	}
	return fmt.Sprintf("%s[black:green]%s[-:-:-]", a.ShortCut, a.ToggleDescription[state])
}

func (a Action) writeButton(w io.Writer) {
	_, _ = fmt.Fprintf(w, "%s ", a.getButton())
}

func (a Action) writeToggleButton(w io.Writer, state bool) {
	_, _ = fmt.Fprintf(w, "%s ", a.getToggleButton(state))
}

func getDefaultActions() ShortCuts {
	sc := ShortCuts{
		ShortCutKeys: map[ActionName]*Action{
			ActionLogScreen: {
				ToggleDescription: map[bool]string{
					true:  "Maximize",
					false: "Minimize",
				},
			},
			ActionFollowLog: {
				ToggleDescription: map[bool]string{
					true:  "Follow",
					false: "Unfollow",
				},
			},
			ActionWrapLog: {
				ToggleDescription: map[bool]string{
					true:  "Wrap",
					false: "Unwrap",
				},
			},
			ActionLogSelection: {
				ToggleDescription: map[bool]string{
					true:  "Select On",
					false: "Select Off",
				},
			},
			ActionProcessScale: {
				Description: "Scale",
			},
			ActionProcessInfo: {
				Description: "Info",
			},
			ActionProcessStart: {
				Description: "Start",
			},
			ActionProcessScreen: {
				ToggleDescription: map[bool]string{
					true:  "Maximize",
					false: "Minimize",
				},
			},
			ActionProcessStop: {
				Description: "Stop",
			},
			ActionProcessRestart: {
				Description: "Restart",
			},
			ActionQuit: {
				Description: "Quit",
			},
			ActionLogFind: {
				Description: "Find",
			},
			ActionLogFindNext: {
				Description: "Next",
			},
			ActionLogFindPrev: {
				Description: "Previous",
			},
			ActionLogFindExit: {
				Description: "Exit Search",
			},
			ActionNsFilter: {
				Description: "Select Namespace",
			},
			ActionHelp: {
				Description: "Shortcuts",
			},
			ActionHideDisabled: {
				ToggleDescription: map[bool]string{
					true:  "Show Disabled",
					false: "Hide Disabled",
				},
			},
			ActionProcFilter: {
				Description: "Search Process",
			},
		},
	}
	for k, v := range sc.ShortCutKeys {
		assignDefaultKeys(k, v)
	}
	return sc
}

func assignDefaultKeys(name ActionName, action *Action) {
	key := defaultShortcuts[name]
	if key == tcell.KeyRune {
		action.ShortCut = string(defaultShortcutsRunes[name])
		action.rune = defaultShortcutsRunes[name]
	} else {
		action.ShortCut = tcell.KeyNames[key]
		action.key = key
	}
}
