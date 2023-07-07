//go:build !windows

package command

import (
	"syscall"
)

const (
	min_sig = 1
	max_sig = 31
)

func (c *CmdWrapper) Stop(sig int) error {
	if c.Cmd == nil {
		return nil
	}
	if sig < min_sig || sig > max_sig {
		sig = int(syscall.SIGTERM)
	}
	pgid, err := syscall.Getpgid(c.Pid())
	if err == nil {
		return syscall.Kill(-pgid, syscall.Signal(sig))
	}
	return err
}

func (c *CmdWrapper) SetCmdArgs() {
	c.Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
