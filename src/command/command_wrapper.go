package command

import (
	"io"
	"os"
	"os/exec"
	"sync/atomic"
)

type CmdWrapper struct {
	cmd            *exec.Cmd
	processGroupID atomic.Int64
}

func (c *CmdWrapper) GetPty() *os.File {
	return nil
}

func (c *CmdWrapper) Start() error {
	if err := c.cmd.Start(); err != nil {
		return err
	}
	c.captureProcessGroup()
	return nil
}

func (c *CmdWrapper) Run() error {
	return c.cmd.Run()
}

func (c *CmdWrapper) Wait() error {
	return c.cmd.Wait()
}

func (c *CmdWrapper) ExitCode() int {
	return c.cmd.ProcessState.ExitCode()
}

func (c *CmdWrapper) Pid() int {
	return c.cmd.Process.Pid
}

func (c *CmdWrapper) StdoutPipe() (io.ReadCloser, error) {
	return c.cmd.StdoutPipe()
}

func (c *CmdWrapper) StderrPipe() (io.ReadCloser, error) {
	return c.cmd.StderrPipe()
}

func (c *CmdWrapper) StdinPipe() (io.WriteCloser, error) {
	return c.cmd.StdinPipe()
}

func (c *CmdWrapper) AttachIo() {
	c.cmd.Stdin = os.Stdin
	c.cmd.Stdout = os.Stdout
	c.cmd.Stderr = os.Stderr
}

func (c *CmdWrapper) SetEnv(env []string) {
	c.cmd.Env = env
}

func (c *CmdWrapper) SetDir(dir string) {
	c.cmd.Dir = dir
}

func (c *CmdWrapper) Output() ([]byte, error) {
	return c.cmd.Output()
}

func (c *CmdWrapper) CombinedOutput() ([]byte, error) {
	return c.cmd.CombinedOutput()
}
