package pclog

import (
	"bufio"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path"
	"sync"
	"sync/atomic"
)

type PCLog struct {
	logger        zerolog.Logger
	writer        *bufio.Writer
	file          io.WriteCloser
	logEventChan  chan logEvent
	wg            sync.WaitGroup
	closer        sync.Once
	isClosed      atomic.Bool
	noMetaData    bool
	flushEachLine bool
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

func (l *PCLog) Open(filePath string, config *types.LoggerConfig) {
	if l.file != nil {
		log.Error().Msgf("log file for %s is already open", filePath)
		return
	}
	if filePath == "" {
		log.Error().Msg("empty file path")
		return
	}

	f, err := l.getWriter(filePath, config)
	if err != nil {
		l.isClosed.Store(true)
		log.Err(err).Msgf("failed to create file %s", filePath)
	}
	l.writer = bufio.NewWriter(f)
	l.file = f
	if config == nil || !config.DisableJSON {
		l.logger = zerolog.New(l.writer)
	} else {
		out := zerolog.NewConsoleWriter(
			func(w *zerolog.ConsoleWriter) {
				w.Out = l.writer
				if len(config.FieldsOrder) > 0 {
					w.PartsOrder = config.FieldsOrder
				}
				if len(config.TimestampFormat) > 0 {
					w.TimeFormat = config.TimestampFormat
				}
				w.NoColor = config.NoColor
			},
		)
		l.logger = zerolog.New(out)
	}
	if config != nil {
		l.noMetaData = config.NoMetadata
		l.flushEachLine = config.FlushEachLine
		if config.AddTimestamp {
			l.logger = l.logger.With().Timestamp().Logger()
			if len(config.TimestampFormat) > 0 {
				zerolog.TimeFieldFormat = config.TimestampFormat
			}
		}
	}

	l.wg.Add(1)
	go l.runCollector()
}

func (l *PCLog) getWriter(filePath string, config *types.LoggerConfig) (io.WriteCloser, error) {
	isRotationEnabled := config != nil && config.Rotation != nil
	log.Debug().Str("filePath", filePath).Bool("rotation", isRotationEnabled).Send()
	if !isRotationEnabled {
		return l.getFileWriter(filePath, config)
	} else {
		return l.getRollingWriter(filePath, config.Rotation)
	}
}

func (l *PCLog) getFileWriter(filePath string, config *types.LoggerConfig) (io.WriteCloser, error) {
	dirName := path.Dir(filePath)
	if err := os.MkdirAll(dirName, 0755); err != nil && !os.IsExist(err) {
		l.isClosed.Store(true)
		log.Err(err).Msgf("failed to create log file directory %s", dirName)
		return nil, err
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_APPEND | os.O_TRUNC
	f, err := os.OpenFile(filePath, flags, 0600)
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

		level := l.logger.Info()
		if event.isErr {
			level = l.logger.Error()
		}
		if !l.noMetaData {
			level = level.Str("process", event.process).Int("replica", event.replica)
		}
		level.Msg(event.message)
		if l.flushEachLine {
			l.writer.Flush()
		}
	}
	l.wg.Done()
}
