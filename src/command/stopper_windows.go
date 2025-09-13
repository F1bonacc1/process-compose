package command

import (
	"os/exec"
	"strconv"

	"github.com/rs/zerolog/log"
)

func (c *CmdWrapper) Stop(sig int, parentOnly bool) error {
	//p.command.Process.Kill()
	log.
		Debug().
		Int("pid", c.Pid()).
		Bool("parentOnly", parentOnly).
		Msg("Stop Windows process.")

	if parentOnly {
		kill := exec.Command("C:/Windows/System32/taskkill.exe", "/F", "/PID", strconv.Itoa(c.Pid()))
		return kill.Run()
	}
	kill := exec.Command("C:/Windows/System32/taskkill.exe", "/T", "/F", "/PID", strconv.Itoa(c.Pid()))
	return kill.Run()
}

func (c *CmdWrapper) SetCmdArgs() {
	//empty for windows
}
