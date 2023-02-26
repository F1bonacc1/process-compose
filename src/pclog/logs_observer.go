package pclog

type LogObserver interface {
	AddLine(line string)
	SetLines(lines []string)
	GetTailLength() int
	GetUniqueID() string
}
