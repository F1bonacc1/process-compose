//go:build !windows

package app

import (
	"syscall"
)

const (
	min_sig = 1
	max_sig = 31
)

func (p *Process) stop(sig int) error {
	if p.command == nil {
		return nil
	}
	if sig < min_sig || sig > max_sig {
		sig = int(syscall.SIGTERM)
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
