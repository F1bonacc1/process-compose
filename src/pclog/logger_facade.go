package pclog

import (
	"bufio"
	"github.com/rs/zerolog/log"
	"os"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
)

type PCLog struct {
	logger       zerolog.Logger
	writer       *bufio.Writer
	file         *os.File
	logEventChan chan logEvent
	wg           sync.WaitGroup
	closer       sync.Once
	isClosed     atomic.Bool
}

type logEvent struct {
	message string
	process string
	replica int
	isErr   bool
}

func NewLogger() *PCLog {
	log := &PCLog{
		logEventChan: make(chan logEvent, 100),
	}

	return log
}

func (l *PCLog) Open(filePath string) {
	if l.file != nil {
		log.Error().Msgf("log file for %s is already open", filePath)
		return
	}
	if filePath == "" {
		log.Error().Msg("empty file path")
		return
	}
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		l.isClosed.Store(true)
		log.Err(err).Msgf("failed to open log file %s", filePath)
	}
	writer := bufio.NewWriter(f)
	l.writer = writer
	l.file = f
	l.logger = zerolog.New(writer)
	l.wg.Add(1)
	go l.runCollector()
}

func (l *PCLog) Info(message string, process string, replica int) {
	if l.isClosed.Load() {
		return
	}
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
	if l.isClosed.Load() {
		return
	}
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
	if l.file == nil {
		return
	}
	l.closer.Do(func() {
		l.isClosed.Store(true)
		close(l.logEventChan)
		l.wg.Wait()
		l.writer.Flush()
		l.file.Close()
	})
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
