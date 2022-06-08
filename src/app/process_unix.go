//go:build !windows

package app

import "syscall"

func (p *Process) stop() error {
	pgid, err := syscall.Getpgid(p.cmd.Process.Pid)
	if err == nil {
		return syscall.Kill(-pgid, syscall.SIGKILL)
	}
	return err
}

func (p *Process) setProcArgs() {
	p.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
