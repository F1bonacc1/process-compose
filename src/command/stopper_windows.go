package command

import (
	"os/exec"
	"strconv"
)

func (c *CmdWrapper) Stop(sig int, parentOnly bool) error {
	//p.command.Process.Kill()
	log.
		Debug().
		Int("pid", c.Pid()).
		Bool("parentOnly", parentOnly).
		Msg("Stop Windows process.")

	if parentOnly {
		kill := exec.Command("TASKKILL", "/F", "/PID", strconv.Itoa(c.Pid()))
		return kill.Run()
	}
	kill := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(c.Pid()))
	return kill.Run()
}

func (c *CmdWrapper) SetCmdArgs() {
	//empty for windows
}
