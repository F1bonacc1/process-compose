package health

import (
	"context"
	"github.com/f1bonacc1/process-compose/src/command"
	"strconv"
	"time"
)

type execChecker struct {
	command    string
	timeout    int
	workingDir string
}

func (c *execChecker) Status() (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.timeout)*time.Second)
	defer cancel()

	cmd := command.BuildCommandContext(ctx, c.command)
	cmd.SetDir(c.workingDir)

	rcMap := make(map[string]string)
	out, err := cmd.CombinedOutput()
	if err != nil {
		rcMap["error"] = err.Error()
	}
	rcMap["output"] = string(out)
	rcMap["exit_code"] = strconv.Itoa(cmd.ExitCode())
	return rcMap, err
}
