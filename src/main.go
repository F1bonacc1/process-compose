package main

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/cmd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

var version = "undefined"

const (
	LogFileFlags = os.O_CREATE | os.O_APPEND | os.O_WRONLY | os.O_TRUNC
	LogFileMode  = os.FileMode(0600)
)

func setupLogger(output io.Writer) {

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        output,
		TimeFormat: "06-01-02 15:04:05.000",
	})
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func mustUser() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to retrieve user info")
	}
	return usr.Username
}

func main() {
	logFile := filepath.Join(os.TempDir(), fmt.Sprintf("process-compose-%s.log", mustUser()))
	file, err := os.OpenFile(logFile, LogFileFlags, LogFileMode)
	if err != nil {
		panic(err)
	}
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()
	setupLogger(file)
	cmd.Execute(version)
}
