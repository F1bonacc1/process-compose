package command

import (
	"io"
	"os/exec"
)

type CmdWrapper struct {
	Cmd *exec.Cmd
}

func (c *CmdWrapper) Start() error {
	return c.Cmd.Start()
}

func (c *CmdWrapper) Wait() error {
	return c.Cmd.Wait()
}

func (c *CmdWrapper) ExitCode() int {
	return c.Cmd.ProcessState.ExitCode()
}

func (c *CmdWrapper) Pid() int {
	return c.Cmd.Process.Pid
}

func (c *CmdWrapper) StdoutPipe() (io.ReadCloser, error) {
	return c.Cmd.StdoutPipe()
}

func (c *CmdWrapper) StderrPipe() (io.ReadCloser, error) {
	return c.Cmd.StderrPipe()
}

func (c *CmdWrapper) SetEnv(env []string) {
	c.Cmd.Env = env
}

func (c *CmdWrapper) SetDir(dir string) {
	c.Cmd.Dir = dir
}
