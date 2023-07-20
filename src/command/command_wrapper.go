package command

import (
	"io"
	"os/exec"
)

type CmdWrapper struct {
	cmd *exec.Cmd
}

func (c *CmdWrapper) Start() error {
	return c.cmd.Start()
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

func (c *CmdWrapper) SetEnv(env []string) {
	c.cmd.Env = env
}

func (c *CmdWrapper) SetDir(dir string) {
	c.cmd.Dir = dir
}
