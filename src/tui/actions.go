package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type ActionName string

const (
	ActionHelp             = ActionName("help")
	ActionLogScreen        = ActionName("log_screen")
	ActionFollowLog        = ActionName("log_follow")
	ActionWrapLog          = ActionName("log_wrap")
	ActionLogSelection     = ActionName("log_select")
	ActionProcessStart     = ActionName("process_start")
	ActionProcessScale     = ActionName("process_scale")
	ActionProcessInfo      = ActionName("process_info")
	ActionProcessStop      = ActionName("process_stop")
	ActionProcessRestart   = ActionName("process_restart")
	ActionProcessScreen    = ActionName("process_screen")
	ActionQuit             = ActionName("quit")
	ActionLogFind          = ActionName("find")
	ActionLogFindNext      = ActionName("find_next")
	ActionLogFindPrev      = ActionName("find_prev")
	ActionLogFindExit      = ActionName("find_exit")
	ActionNsFilter         = ActionName("ns_filter")
	ActionHideDisabled     = ActionName("hide_disabled")
	ActionProcFilter       = ActionName("proc_filter")
	ActionThemeSelector    = ActionName("theme_selector")
	ActionSendToBackground = ActionName("send_to_background")
	ActionFullScreen       = ActionName("full_screen")
	ActionFocusChange      = ActionName("focus_change")
	ActionClearLog         = ActionName("clear_log")
	ActionMarkLog          = ActionName("mark_log")
	ActionEditProcess      = ActionName("edit_process")
	ActionTermExit         = ActionName("term_exit")
	ActionReloadConfig     = ActionName("reload_config")
	ActionDependencyGraph  = ActionName("dependency_graph")
)

var defaultShortcuts = map[ActionName]tcell.Key{
	ActionHelp:             tcell.KeyF1,
	ActionLogScreen:        tcell.KeyF4,
	ActionFollowLog:        tcell.KeyF5,
	ActionWrapLog:          tcell.KeyF6,
	ActionLogSelection:     tcell.KeyCtrlS,
	ActionProcessScale:     tcell.KeyF2,
	ActionProcessInfo:      tcell.KeyF3,
	ActionProcessStart:     tcell.KeyF7,
	ActionProcessStop:      tcell.KeyF9,
	ActionProcessRestart:   tcell.KeyCtrlR,
	ActionProcessScreen:    tcell.KeyF8,
	ActionQuit:             tcell.KeyF10,
	ActionLogFind:          tcell.KeyCtrlF,
	ActionLogFindNext:      tcell.KeyCtrlN,
	ActionLogFindPrev:      tcell.KeyCtrlP,
	ActionLogFindExit:      tcell.KeyEsc,
	ActionNsFilter:         tcell.KeyCtrlG,
	ActionHideDisabled:     tcell.KeyCtrlD,
	ActionProcFilter:       tcell.KeyRune,
	ActionThemeSelector:    tcell.KeyCtrlT,
	ActionSendToBackground: tcell.KeyCtrlB,
	ActionFullScreen:       tcell.KeyCtrlRightSq,
	ActionFocusChange:      tcell.KeyTab,
	ActionClearLog:         tcell.KeyCtrlK,
	ActionMarkLog:          tcell.KeyRune,
	ActionEditProcess:      tcell.KeyCtrlE,
	ActionReloadConfig:     tcell.KeyCtrlL,
	ActionTermExit:         tcell.KeyCtrlA,
	ActionDependencyGraph:  tcell.KeyCtrlQ,
}

var defaultShortcutsRunes = map[ActionName]rune{
	ActionProcFilter: '/',
	ActionMarkLog:    'm',
}

var generalActionsOrder = []ActionName{
	ActionHelp,
	ActionThemeSelector,
	ActionSendToBackground,
	ActionFullScreen,
	ActionTermExit,
}

var logActionsOrder = []ActionName{
	ActionLogScreen,
	ActionFollowLog,
	ActionWrapLog,
	ActionLogSelection,
	ActionLogFind,
	ActionClearLog,
	ActionMarkLog,
}

var procActionsOrder = []ActionName{
	ActionProcFilter,
	ActionProcessScale,
	ActionProcessInfo,
	ActionProcessStart,
	ActionProcessScreen,
	ActionProcessStop,
	ActionProcessRestart,
	ActionEditProcess,
	ActionReloadConfig,
	ActionNsFilter,
	ActionHideDisabled,
	ActionDependencyGraph,
	ActionQuit,
}

type ShortCuts struct {
	ShortCutKeys  map[ActionName]*Action `yaml:"shortcuts"`
	keyActionMap  map[tcell.Key]*Action
	runeActionMap map[rune]*Action
	style         config.Help
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
		log.Err(err).Msgf("Failed to unmarshal file %s", filePath)
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
			// Deep copy `action` into `oldAction`.
			oldAction.ShortCut = action.ShortCut
			oldAction.key = action.key
			oldAction.rune = action.rune
			if len(strings.TrimSpace(action.Description)) > 0 {
				oldAction.Description = action.Description
			}
			if len(action.ToggleDescription) == 2 {
				oldAction.ToggleDescription = action.ToggleDescription
			}
			// Note: this leaves the entries for the old key/rune in the map.
			if oldAction.rune != 0 {
				s.runeActionMap[oldAction.rune] = oldAction
			} else {
				s.keyActionMap[oldAction.key] = oldAction
			}
		} else {
			log.Error().Msgf("Invalid action '%s' shortcut", actionName)
		}

	}
}

func (s *ShortCuts) addButton(action ActionName, flex *tview.Flex) {
	s.ShortCutKeys[action].addButton(flex, s.style)
}

func (s *ShortCuts) addToggleButton(action ActionName, flex *tview.Flex, state bool) {
	s.ShortCutKeys[action].addToggleButton(flex, state, s.style)
}

func (s *ShortCuts) addCategory(category string, flex *tview.Flex) {
	textView := tview.NewTextView().SetDynamicColors(true)
	_, _ = fmt.Fprintf(textView, "[%s::b]%s[-:-:-] ",
		string(s.style.FgCategoryColor),
		category)

	flex.AddItem(textView, len(category)+1, 1, false)
}

func (s *ShortCuts) StylesChanged(style *config.Styles) {
	s.style = style.Help()
}

func (s *ShortCuts) setAction(actName ActionName, actionFn func()) {
	if act, found := s.ShortCutKeys[actName]; found {
		act.actionFn = actionFn
		if act.rune != 0 {
			s.runeActionMap[act.rune] = act
		} else {
			s.keyActionMap[act.key] = act
		}
	} else {
		log.Error().Msgf("Invalid action '%s' shortcut", actName)
	}
}

func (s *ShortCuts) runRuneAction(r rune, event *tcell.EventKey) *tcell.EventKey {
	if act, found := s.runeActionMap[r]; found {
		act.actionFn()
		return nil
	}
	return event
}

func (s *ShortCuts) runKeyAction(key tcell.Key, event *tcell.EventKey) *tcell.EventKey {
	if act, found := s.keyActionMap[key]; found {
		act.actionFn()
		return nil
	}
	return event
}

func parseShortCuts(sc *ShortCuts) {
	for actionName, action := range sc.ShortCutKeys {
		if len(action.ShortCut) == 1 {
			action.rune = []rune(action.ShortCut)[0]
			continue
		}
		key, err := keyName2Key(action.ShortCut)
		if err != nil {
			log.Err(err).Msgf("Failed in parsing '%s' shortcut. Using default: %s", actionName, action.ShortCut)
			assignDefaultKeys(actionName, action)
			continue
		}
		action.key = key
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
	ShortCut          string          `yaml:"shortcut"`
	rune              rune            `yaml:"char,omitempty"`
	Description       string          //`yaml:"description,omitempty"`
	ToggleDescription map[bool]string //`yaml:"toggle_description,omitempty"`
	key               tcell.Key
	actionFn          func()
}

func (a Action) addButton(flex *tview.Flex, style config.Help) {
	btnText := fmt.Sprintf("%s[%s:%s:]%s[-:-:-] ",
		a.ShortCut,
		string(style.FgColor),
		string(style.HlColor),
		a.Description)
	button := tview.NewButton(btnText).SetSelectedFunc(a.actionFn)
	button.SetStyle(tcell.StyleDefault.Background(style.ButtonBgColor.Color()).Foreground(style.KeyColor.Color()))
	button.SetActivatedStyle(tcell.StyleDefault.Background(style.ButtonBgColor.Color()).Foreground(style.KeyColor.Color()))
	flex.AddItem(button, len(a.ShortCut+a.Description)+1, 1, false)
}

func (a Action) addToggleButton(flex *tview.Flex, state bool, style config.Help) {
	btnText := fmt.Sprintf("%s[%s:%s:]%s[-:-:-] ",
		a.ShortCut,
		string(style.FgColor),
		string(style.HlColor),
		a.ToggleDescription[state])
	button := tview.NewButton(btnText).SetSelectedFunc(a.actionFn)
	button.SetStyle(tcell.StyleDefault.Background(style.ButtonBgColor.Color()).Foreground(style.KeyColor.Color()))
	button.SetActivatedStyle(tcell.StyleDefault.Background(style.ButtonBgColor.Color()).Foreground(style.KeyColor.Color()))
	flex.AddItem(button, len(a.ShortCut+a.ToggleDescription[state])+1, 1, false)
}

func newShortCuts() *ShortCuts {
	sc := &ShortCuts{
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
			ActionThemeSelector: {
				Description: "Select Theme",
			},
			ActionSendToBackground: {
				Description: "Send Process Compose to Background",
			},
			ActionFullScreen: {
				Description: "Toggle Full Screen",
			},
			ActionFocusChange: {
				Description: "Toggle Log/Process Focus",
			},
			ActionClearLog: {
				Description: "Clear Process Log",
			},
			ActionMarkLog: {
				Description: "Add Mark to Log",
			},
			ActionEditProcess: {
				Description: "Edit Process",
			},
			ActionReloadConfig: {
				Description: "Reload Project Configuration",
			},
			ActionTermExit: {
				Description: "Switch to Escape Mode",
			},
			ActionDependencyGraph: {
				Description: "Dependency Graph",
			},
		},
	}
	for k, v := range sc.ShortCutKeys {
		assignDefaultKeys(k, v)
	}
	sc.keyActionMap = make(map[tcell.Key]*Action)
	sc.runeActionMap = make(map[rune]*Action)
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
