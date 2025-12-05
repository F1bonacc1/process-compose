package health

import (
	"context"
	"strconv"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
)

type execChecker struct {
	command     string
	timeout     int
	workingDir  string
	env         []string
	shellConfig command.ShellConfig
}

func (c *execChecker) Status() (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.timeout)*time.Second)
	defer cancel()

	cmd := command.BuildCommandShellArgContext(ctx, c.shellConfig, c.command)
	cmd.SetDir(c.workingDir)
	cmd.SetEnv(c.env)

	rcMap := make(map[string]string)
	out, err := cmd.CombinedOutput()
	if err != nil {
		rcMap["error"] = err.Error()
	}
	rcMap["output"] = string(out)
	rcMap["exit_code"] = strconv.Itoa(cmd.ExitCode())
	return rcMap, err
}
