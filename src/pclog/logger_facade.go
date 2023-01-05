package pclog

import (
	"bufio"
	"os"
	"sync"

	"github.com/rs/zerolog"
)

type PCLog struct {
	logger       zerolog.Logger
	writer       *bufio.Writer
	file         *os.File
	logEventChan chan logEvent
	wg           sync.WaitGroup
}

type logEvent struct {
	message string
	process string
	replica int
	isErr   bool
}

func NewLogger(outputPath string) *PCLog {
	f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	writer := bufio.NewWriter(f)

	log := &PCLog{
		writer:       writer,
		file:         f,
		logger:       zerolog.New(writer),
		logEventChan: make(chan logEvent, 100),
	}
	log.wg.Add(1)
	go log.runCollector()
	return log
}

func (l *PCLog) Info(message string, process string, replica int) {
	l.logEventChan <- logEvent{
		message: message,
		process: process,
		replica: replica,
		isErr:   false,
	}
}
func (l *PCLog) info(message string, process string, replica int) {
	l.logger.Info().
		Str("process", process).
		Int("replica", replica).
		Msg(message)
}

func (l *PCLog) Error(message string, process string, replica int) {
	l.logEventChan <- logEvent{
		message: message,
		process: process,
		replica: replica,
		isErr:   true,
	}
}

func (l *PCLog) error(message string, process string, replica int) {
	l.logger.Error().
		Str("process", process).
		Int("replica", replica).
		Msg(message)
}

func (l *PCLog) Close() {
	close(l.logEventChan)
	l.wg.Wait()
	l.writer.Flush()
	l.file.Close()
}

func (l *PCLog) runCollector() {
	for {
		event, open := <-l.logEventChan
		if !open {
			break
		}
		if event.isErr {
			l.error(event.message, event.process, event.replica)
		} else {
			l.info(event.message, event.process, event.replica)
		}
	}
	l.wg.Done()
}
