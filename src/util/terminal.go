package util

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

type projectNamer interface{ GetProjectName() (string, error) }

func SetProjectNameAsTerminalTitle(n projectNamer) {
	name, err := n.GetProjectName()
	if err != nil {
		log.Err(err).Msgf("Failed to set terminal title: %v", err)
		return
	}

	SetTerminalTitle(name)
}

func SetTerminalTitle(title string) { fmt.Printf("\033]0;process-compose: %s\007", title) }
