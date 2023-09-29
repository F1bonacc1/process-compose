package pclog

import "github.com/f1bonacc1/process-compose/src/types"

type PcNilLog struct {
}

func NewNilLogger() *PcNilLog {

	return &PcNilLog{}
}

func (l *PcNilLog) Open(filePath string, rotation *types.LoggerConfig) {
}

func (l *PcNilLog) Sync() {
}

func (l *PcNilLog) Info(message string, process string, replica int) {

}

func (l *PcNilLog) Error(message string, process string, replica int) {

}

func (l *PcNilLog) Close() {

}
