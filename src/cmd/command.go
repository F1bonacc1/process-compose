package cmd

import (
	"context"
	"os"
	"os/exec"
	"runtime"
)

func BuildCommand(shellCmd string) *exec.Cmd {
	return exec.Command(getRunnerShell(), getRunnerArg(), shellCmd)
}

func BuildCommandContext(ctx context.Context, shellCmd string) *exec.Cmd {
	return exec.CommandContext(ctx, getRunnerShell(), getRunnerArg(), shellCmd)
}

func getRunnerShell() string {
	shell, ok := os.LookupEnv("SHELL")
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
	if runtime.GOOS == "windows" {
		return "/C"
	} else {
		return "-c"
	}
}
