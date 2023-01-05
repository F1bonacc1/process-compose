package main

import (
	"github.com/f1bonacc1/process-compose/src/cmd"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"time"
)

func setupLogger(output io.Writer) {

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        output,
		TimeFormat: "06-01-02 15:04:05.000",
	})
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {

	file, err := os.OpenFile(config.LogFilePath, config.LogFileFlags, config.LogFileMode)
	if err != nil {
		panic(err)
	}
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()
	setupLogger(file)
	log.Info().Msgf("Process Compose %s", config.Version)
	cmd.Execute()
}
