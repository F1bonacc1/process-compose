package cmd

import "github.com/rs/zerolog/log"

func runInDetachedMode() {
	log.Fatal().Msg("Running in detached mode is not supported on Windows")
}
