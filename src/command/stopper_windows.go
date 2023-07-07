package command

import (
	"os/exec"
	"strconv"
)

func (c *CmdWrapper) Stop(sig int, _parentOnly bool) error {
	//p.command.Process.Kill()
	kill := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(c.Pid()))
	return kill.Run()
}

func (c *CmdWrapper) SetCmdArgs() {
	//empty for windows
}
