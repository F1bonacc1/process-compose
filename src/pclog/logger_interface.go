package pclog

type PcLogger interface {
	Open(filePath string)
	Info(message string, process string, replica int)
	Error(message string, process string, replica int)
	Close()
}
