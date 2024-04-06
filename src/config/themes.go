package config

import (
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"slices"
)

const (
	CustomStyleName = "Custom Style"
)

// Themes represents a list of styles.
type Themes struct {
	styles       []*Styles
	activeStyles *Styles
	listeners    []StyleListener
}

// ---------- THEMES ----------

func NewThemes() *Themes {
	t := &Themes{
		styles:       make([]*Styles, 0),
		activeStyles: NewStyles(),
	}
	files, err := themesFolder.ReadDir("themes")
	if err == nil {
		for _, file := range files {
			fileName := path.Join("themes", file.Name())
			b, err := themesFolder.ReadFile(fileName)
			if err != nil {
				log.Err(err).Msgf("Error reading files %s", fileName)
				continue
			}
			s := &Styles{}
			err = yaml.Unmarshal(b, s)
			if err != nil {
				log.Err(err).Msgf("Error parsing theme %s", fileName)
				continue
			}
			t.styles = append(t.styles, s)
		}
	} else {
		log.Err(err).Msg("Error reading themes folder")
	}
	custom, err := t.loadFromFile()
	if err == nil {
		t.styles = append(t.styles, custom)
	}

	return t
}

func (t *Themes) loadFromFile() (*Styles, error) {
	filePath := GetThemesPath()
	b, err := os.ReadFile(filePath)
	if err != nil {
		log.Warn().Err(err).Msgf("Error reading themes file %s", filePath)
		return nil, err
	}
	s := NewStyles()
	err = yaml.Unmarshal(b, s)
	if err != nil {
		log.Err(err).Msgf("Error parsing theme %s", filePath)
		return nil, err
	}
	s.Style.Name = CustomStyleName
	return s, nil
}

func (t *Themes) SelectStyles(name string) {
	if name == CustomStyleName {
		t.SelectStylesFromFile()
		return
	}
	changed := false
	for _, styles := range t.styles {
		if styles.Style.Name == name {
			t.activeStyles = styles
			changed = true
			break
		}
	}
	if changed {
		t.activeStyles.Update()
		t.fireStylesChanged()
	} else {
		log.Error().Msgf("Theme %s not found", name)
	}
}

func (t *Themes) SelectStylesFromFile() {
	custom, err := t.loadFromFile()
	if err == nil {
		t.activeStyles = custom
		t.activeStyles.Update()
		t.fireStylesChanged()
	} else {
		log.Err(err).Msgf("Failed to load custom theme from %s", GetThemesPath())
	}
}

func (t *Themes) GetThemeNames() []string {
	names := make([]string, 0)
	for _, styles := range t.styles {
		names = append(names, styles.Style.Name)
	}
	slices.Sort(names)
	return names
}

// AddListener registers a new listener.
func (t *Themes) AddListener(l StyleListener) {
	t.listeners = append(t.listeners, l)
}

// RemoveListener removes a listener.
func (t *Themes) RemoveListener(l StyleListener) {
	victim := -1
	for i, lis := range t.listeners {
		if lis == l {
			victim = i
			break
		}
	}
	if victim == -1 {
		return
	}
	t.listeners = append(t.listeners[:victim], t.listeners[victim+1:]...)
}

// GetActiveStyles returns the active styles.
func (t *Themes) GetActiveStyles() *Styles {
	return t.activeStyles
}

func (t *Themes) fireStylesChanged() {
	for _, list := range t.listeners {
		list.StylesChanged(t.activeStyles)
	}
}
