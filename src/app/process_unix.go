//go:build !windows

package app

import (
	"github.com/rs/zerolog/log"
	"syscall"
)

const (
	min_sig = 1
	max_sig = 31
)

func (p *Process) stop(sig int, parentOnly bool) error {
	if p.command == nil {
		return nil
	}
	if sig < min_sig || sig > max_sig {
		sig = int(syscall.SIGTERM)
	}

	log.
		Debug().
		Int("pid", p.command.Process.Pid).
		Int("signal", sig).
		Bool("parentOnly", parentOnly).
		Msg("Stop Unix process.")

	if parentOnly {
		return syscall.Kill(p.command.Process.Pid, syscall.Signal(sig))
	}

	pgid, err := syscall.Getpgid(p.command.Process.Pid)
	if err == nil {
		return syscall.Kill(-pgid, syscall.Signal(sig))
	}

	return err
}

func (p *Process) setProcArgs() {
	p.command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
