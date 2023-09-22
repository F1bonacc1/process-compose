package pclog

import "github.com/f1bonacc1/process-compose/src/types"

type PcLogger interface {
	Open(filePath string, rotation *types.LogRotationConfig)
	Info(message string, process string, replica int)
	Error(message string, process string, replica int)
	Close()
}
