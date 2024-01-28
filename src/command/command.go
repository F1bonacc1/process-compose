package command

import (
	"context"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"runtime"
)

func BuildCommand(cmd string, args []string) *CmdWrapper {
	return &CmdWrapper{
		cmd: exec.Command(cmd, args...),
	}
	//return NewMockCommand()
}

func BuildCommandContext(ctx context.Context, shellCmd string) *CmdWrapper {
	return &CmdWrapper{
		cmd: exec.CommandContext(ctx, getRunnerShell(), getRunnerArg(), shellCmd),
	}
}

func BuildCommandShellArgContext(ctx context.Context, shell ShellConfig, cmd string) *CmdWrapper {
	return &CmdWrapper{
		cmd: exec.CommandContext(ctx, shell.ShellCommand, shell.ShellArgument, cmd),
	}
}

func getRunnerShell() string {
	shell, ok := os.LookupEnv("COMPOSE_SHELL")
	if !ok {
		if runtime.GOOS == "windows" {
			shell = "cmd"
		} else {
			shell = "bash"
		}
	}
	return shell
}

func getRunnerArg() string {
	arg := "-c"
	if runtime.GOOS == "windows" {
		arg = "/C"
	}
	return arg
}

func DefaultShellConfig() *ShellConfig {
	return &ShellConfig{
		ShellCommand:  getRunnerShell(),
		ShellArgument: getRunnerArg(),
	}
}

func ValidateShellConfig(shell ShellConfig) {
	_, err := exec.LookPath(shell.ShellCommand)
	if err != nil {
		log.Fatal().Err(err).Msgf("Couldn't find %s", shell.ShellCommand)
	}
}
