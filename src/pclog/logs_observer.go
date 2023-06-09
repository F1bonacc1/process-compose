package pclog

type LogObserver interface {
	WriteString(line string) (n int, err error)
	SetLines(lines []string)
	GetTailLength() int
	GetUniqueID() string
}
