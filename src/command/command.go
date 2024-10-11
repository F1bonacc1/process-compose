package command

import (
	"context"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func BuildCommand(cmd string, args []string) *CmdWrapper {
	return &CmdWrapper{
		cmd: exec.Command(cmd, args...),
	}
}

func BuildPtyCommand(cmd string, args []string) *CmdWrapperPty {
	return &CmdWrapperPty{
		CmdWrapper: BuildCommand(cmd, args),
	}
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

func BuildApptainerCommand(containerRuntime string, containerExecution string, containerVolumes []string, containerArgs []string, containerImage string, containerCmd string, cmdArgs []string) *CmdWrapper {
	volumes := "-B " + strings.Join(containerVolumes, ",")
	container_args := strings.Join(containerArgs, " ")
	args_string := strings.Join(cmdArgs, " ")

	args := []string{containerExecution, volumes, container_args, containerImage, containerCmd, args_string}

	return &CmdWrapper{
		cmd: exec.Command(containerRuntime, args...),
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
