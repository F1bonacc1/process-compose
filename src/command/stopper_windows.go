package command

import (
	"os/exec"
	"strconv"

	"github.com/rs/zerolog/log"
)

func (c *CmdWrapper) Stop(sig int, parentOnly bool) error {
	pid := c.Pid()
	if pid == 0 {
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

	kill := exec.Command("taskkill", args...)
	err := kill.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// 128: The process with PID XXX could not be terminated. Reason: There is no running instance of the task.
			if exitErr.ExitCode() == 128 {
				return nil
			}
		}
	}
	return err
}

func (c *CmdWrapper) SetCmdArgs() {
	//empty for windows
}
