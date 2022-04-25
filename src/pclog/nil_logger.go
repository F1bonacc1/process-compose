package pclog

type PcNilLog struct {
}

func NewNilLogger() *PcNilLog {

	return &PcNilLog{}
}

func (l *PcNilLog) Sync() {
}

func (l PcNilLog) Info(message string, process string, replica int) {

}

func (l PcNilLog) Error(message string, process string, replica int) {

}

func (l PcNilLog) Close() {

}
