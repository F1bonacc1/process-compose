//go:build !windows

package command

import (
	"github.com/rs/zerolog/log"
	"syscall"
)

const (
	min_sig = 1
	max_sig = 31
)

func (c *CmdWrapper) Stop(sig int, parentOnly bool) error {
	if c.cmd == nil {
		return nil
	}
	if sig < min_sig || sig > max_sig {
		sig = int(syscall.SIGTERM)
	}

	log.
		Debug().
		Int("pid", c.Pid()).
		Int("signal", sig).
		Bool("parentOnly", parentOnly).
		Msg("Stop Unix process.")

	if parentOnly {
		return c.cmd.Process.Signal(syscall.Signal(sig))
	}

	pgid, err := syscall.Getpgid(c.Pid())
	if err == nil {
		return syscall.Kill(-pgid, syscall.Signal(sig))
	}

	return err
}

func (c *CmdWrapper) SetCmdArgs() {
	c.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
