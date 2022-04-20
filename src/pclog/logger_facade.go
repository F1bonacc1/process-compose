package pclog

import (
	"bufio"
	"os"

	"github.com/rs/zerolog"
)

type PCLog struct {
	logger zerolog.Logger
	writer *bufio.Writer
	file   *os.File
}

func NewLogger(outputPath string) *PCLog {
	f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	writer := bufio.NewWriter(f)

	return &PCLog{
		writer: writer,
		file:   f,
		logger: zerolog.New(writer),
	}
}

func (l PCLog) Info(message string, process string, replica int) {
	l.logger.Info().
		Str("process", process).
		Int("replica", replica).
		Msg(message)

}

func (l PCLog) Error(message string, process string, replica int) {
	l.logger.Error().
		Str("process", process).
		Int("replica", replica).
		Msg(message)
}

func (l PCLog) Close() {
	l.writer.Flush()
	l.file.Close()
}
