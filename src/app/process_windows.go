package app

import (
	"os/exec"
	"strconv"
)

func (p *Process) stop(sig int) error {
	//p.cmd.Process.Kill()
	kill := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(p.cmd.Process.Pid))
	return kill.Run()
}

func (p *Process) setProcArgs() {
	//empty for windows
}
