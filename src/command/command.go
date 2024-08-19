package command

import (
	"context"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"runtime"
)

func BuildCommandContext(ctx context.Context, onCancel func() error, cmd string, args []string) *CmdWrapper {
	cmdCtx := exec.CommandContext(ctx, cmd, args...)
	cmdCtx.Cancel = onCancel
	return &CmdWrapper{cmd: cmdCtx}
}

func BuildPtyCommandContext(ctx context.Context, onCancel func() error, cmd string, args []string) *CmdWrapperPty {
	return &CmdWrapperPty{
		CmdWrapper: BuildCommandContext(ctx, onCancel, cmd, args),
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

func getElevatedRunnerCmd() string {
	shell := "sudo"
	if runtime.GOOS == "windows" {
		shell = "runas"
	}
	return shell
}

func getElevatedRunnerArg() string {
	arg := "-S"
	if runtime.GOOS == "windows" {
		arg = "/user:Administrator"
	}
	return arg
}

func DefaultShellConfig() *ShellConfig {
	return &ShellConfig{
		ShellCommand:     getRunnerShell(),
		ShellArgument:    getRunnerArg(),
		ElevatedShellCmd: getElevatedRunnerCmd(),
		ElevatedShellArg: getElevatedRunnerArg(),
	}
}

func ValidateShellConfig(shell ShellConfig) {
	_, err := exec.LookPath(shell.ShellCommand)
	if err != nil {
		log.Fatal().Err(err).Msgf("Couldn't find %s", shell.ShellCommand)
	}
}
