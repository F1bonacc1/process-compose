package command

import (
	"os/exec"
	"strconv"

	"github.com/rs/zerolog/log"
)

func (c *CmdWrapper) Stop(sig int, parentOnly bool) error {
	pid := c.Pid()
	if pid <= 0 {
		return nil
	}
	log.
		Debug().
		Int("pid", pid).
		Bool("parentOnly", parentOnly).
		Msg("Stop Windows process.")

	args := []string{"/F", "/PID", strconv.Itoa(pid)}
	if !parentOnly {
		args = append([]string{"/T"}, args...)
	}

	// Try taskkill
	kill := exec.Command("taskkill", args...)
	_ = kill.Run()

	// Direct kill as fallback
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}

	return nil
}

func (c *CmdWrapper) SetCmdArgs() {
	//empty for windows
}
