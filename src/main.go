package main

import (
	"github.com/f1bonacc1/process-compose/src/cmd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

var version = "undefined"

func setupLogger() {

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "06-01-02 15:04:05",
	})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func init() {
	setupLogger()
}

func main() {
	cmd.Execute(version)
}
