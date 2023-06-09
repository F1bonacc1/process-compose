package pclog

type multiLineHandler func(string []string)
type lineHandler func(s string) (n int, err error)

type Connector struct {
	LogObserver
	logLinesHandler   multiLineHandler
	logMessageHandler lineHandler
	uniqueId          string
	taiLength         int
}

func NewConnector(mlHandler multiLineHandler, slHandler lineHandler, tail int) *Connector {
	return &Connector{
		logLinesHandler:   mlHandler,
		logMessageHandler: slHandler,
		uniqueId:          GenerateUniqueID(10),
		taiLength:         tail,
	}
}

func (c *Connector) WriteString(s string) (n int, err error) {
	return c.logMessageHandler(s)
}
func (c *Connector) SetLines(lines []string) {
	c.logLinesHandler(lines)
}
func (c *Connector) GetUniqueID() string {
	return c.uniqueId
}

func (c *Connector) GetTailLength() int {
	return c.taiLength
}
