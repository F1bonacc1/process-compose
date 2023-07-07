package command

import (
	"context"
	"io"
)

type MockCommand struct {
	dir            string
	env            []string
	cancel         context.CancelFunc
	stopChan       chan struct{}
	infoNoiseMaker *noiseMaker
	errNoiseMaker  *noiseMaker
}

func NewMockCommand() *MockCommand {
	return &MockCommand{
		stopChan:       make(chan struct{}),
		infoNoiseMaker: newNoiseMaker("info noise"),
		errNoiseMaker:  newNoiseMaker("error noise"),
	}
}

func (c *MockCommand) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.infoNoiseMaker.Run(ctx)
	return nil
}
func (c *MockCommand) Stop(_ int) error {
	c.stopChan <- struct{}{}
	c.cancel()
	return nil
}

func (c *MockCommand) SetCmdArgs() {

}

func (c *MockCommand) Wait() error {
	<-c.stopChan
	return nil
}

func (c *MockCommand) ExitCode() int {
	return 0
}

func (c *MockCommand) Pid() int {
	return 123456
}

func (c *MockCommand) StdoutPipe() (io.ReadCloser, error) {
	return c.infoNoiseMaker, nil
}

func (c *MockCommand) StderrPipe() (io.ReadCloser, error) {
	return c.errNoiseMaker, nil
}

func (c *MockCommand) SetEnv(env []string) {
	c.env = env
}

func (c *MockCommand) SetDir(dir string) {
	c.dir = dir
}
