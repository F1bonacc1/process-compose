package health

import (
	"context"
	"github.com/f1bonacc1/process-compose/src/command"
	"time"
)

type execChecker struct {
	command string
	timeout int
}

func (c *execChecker) Status() (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.timeout)*time.Second)
	defer cancel()

	cmd := command.BuildCommandContext(ctx, c.command)

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return map[string]int{"exit_code": cmd.ProcessState.ExitCode()}, nil
}
