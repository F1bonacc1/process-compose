package app

import (
	"os/exec"
	"strconv"
)

func (p *Process) stop(sig int, _parentOnly bool) error {
	//p.command.Process.Kill()
	kill := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(p.command.Process.Pid))
	return kill.Run()
}

func (p *Process) setProcArgs() {
	//empty for windows
}
