package config

import (
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
)

type (
	Sort struct {
		By         string `yaml:"by"`
		IsReversed bool   `yaml:"isReversed"`
	}
	Settings struct {
		Theme string `yaml:"theme"`
		Sort  Sort   `yaml:"sort"`
	}
)

func NewSettings() *Settings {
	return &Settings{
		Theme: "Default",
		Sort: Sort{
			By:         DefaultSortColumn,
			IsReversed: false,
		},
	}
}

func (s *Settings) Load() *Settings {
	filePath := GetSettingsPath()
	b, err := os.ReadFile(filePath)
	if err != nil {
		log.Warn().Err(err).Msgf("Error reading settings file %s", filePath)
	}
	err = yaml.Unmarshal(b, s)
	if err != nil {
		log.Warn().Err(err).Msgf("Error parsing settings file %s", filePath)
	}
	return s
}

func (s *Settings) Save() error {
	filePath := GetSettingsPath()
	b, err := yaml.Marshal(s)
	if err != nil {
		log.Warn().Err(err).Msgf("Error marshalling settings file %s", filePath)
		return err
	}
	err = os.WriteFile(filePath, b, 0644)
	return err
}
