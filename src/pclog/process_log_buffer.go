package pclog

const (
	slack = 100
)

type ProcessLogBuffer struct {
	buffer []string
	size   int
}

func NewLogBuffer(size int) *ProcessLogBuffer {
	return &ProcessLogBuffer{
		size:   size,
		buffer: make([]string, 0, size),
	}
}

func (b *ProcessLogBuffer) Write(message string) {
	b.buffer = append(b.buffer, message)
	if len(b.buffer) > b.size+slack {
		b.buffer = b.buffer[slack:]
	}
}

func (b ProcessLogBuffer) GetLog(offsetFromEnd, limit int) []string {
	if len(b.buffer) == 0 {
		return []string{}
	}
	if offsetFromEnd < 0 {
		offsetFromEnd = 0
	}
	if offsetFromEnd > len(b.buffer) {
		offsetFromEnd = len(b.buffer)
	}

	if limit < 1 {
		limit = 0
	}
	if limit > len(b.buffer) {
		limit = len(b.buffer)
	}
	if offsetFromEnd+limit > len(b.buffer) {
		limit = len(b.buffer) - offsetFromEnd
	}
	if limit == 0 {
		return b.buffer[len(b.buffer)-offsetFromEnd:]
	}
	return b.buffer[len(b.buffer)-offsetFromEnd : offsetFromEnd+limit]
}
