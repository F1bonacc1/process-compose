package health

import (
	"context"
	"github.com/f1bonacc1/process-compose/src/command"
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

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return map[string]int{"exit_code": cmd.ExitCode()}, nil
}
