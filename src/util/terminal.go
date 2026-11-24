package util

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"golang.org/x/term"
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

func SetTerminalTitle(title string) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	fmt.Printf("\033]0;process-compose: %s\007", title)
}