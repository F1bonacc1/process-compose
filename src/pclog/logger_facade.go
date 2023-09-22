package pclog

import (
	"bufio"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
)

type PCLog struct {
	logger       zerolog.Logger
	writer       *bufio.Writer
	file         io.WriteCloser
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
	l := &PCLog{
		logEventChan: make(chan logEvent, 100),
	}

	return l
}

func (l *PCLog) Open(filePath string, rotation *types.LogRotationConfig) {
	if l.file != nil {
		log.Error().Msgf("log file for %s is already open", filePath)
		return
	}
	if filePath == "" {
		log.Error().Msg("empty file path")
		return
	}

	f, err := l.getWriter(filePath, rotation)
	if err != nil {
		l.isClosed.Store(true)
		log.Err(err).Msgf("failed to create file %s", filePath)
	}
	l.writer = bufio.NewWriter(f)
	l.file = f
	l.logger = zerolog.New(l.writer)
	l.wg.Add(1)
	go l.runCollector()
}

func (l *PCLog) getWriter(filePath string, rotation *types.LogRotationConfig) (io.WriteCloser, error) {
	dirName := path.Dir(filePath)
	if err := os.MkdirAll(dirName, 0700); err != nil && !os.IsExist(err) {
		l.isClosed.Store(true)
		log.Err(err).Msgf("failed to create log file directory %s", dirName)
		return nil, err
	}
	if rotation == nil {
		log.Debug().Str("filePath", filePath).Msg("no rotation config")
		return l.getFileWriter(filePath)
	} else {
		log.Debug().Str("filePath", filePath).Msg("rotation config")
		return l.getRollingWriter(filePath, rotation)
	}
}

func (l *PCLog) getFileWriter(filePath string) (io.WriteCloser, error) {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		l.isClosed.Store(true)
		log.Err(err).Msgf("failed to open log file %s", filePath)
		return nil, err
	}
	return f, nil
}

func (l *PCLog) getRollingWriter(filePath string, rotation *types.LogRotationConfig) (io.WriteCloser, error) {
	return &lumberjack.Logger{
		Filename:   filePath,
		MaxBackups: rotation.MaxBackups, // files
		MaxSize:    rotation.MaxSize,    // megabytes
		MaxAge:     rotation.MaxAge,     // days
		LocalTime:  true,
		Compress:   rotation.Compress,
	}, nil
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
