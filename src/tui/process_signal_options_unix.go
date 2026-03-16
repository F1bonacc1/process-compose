//go:build !windows

package tui

import "syscall"

func availableSignalOptions() []processSignalOption {
	return []processSignalOption{
		{Signal: int(syscall.SIGHUP), Name: "SIGHUP", Description: "hang up or reload"},
		{Signal: int(syscall.SIGINT), Name: "SIGINT", Description: "interrupt"},
		{Signal: int(syscall.SIGQUIT), Name: "SIGQUIT", Description: "quit and dump core"},
		{Signal: int(syscall.SIGUSR1), Name: "SIGUSR1", Description: "application-defined signal 1"},
		{Signal: int(syscall.SIGUSR2), Name: "SIGUSR2", Description: "application-defined signal 2"},
		{Signal: int(syscall.SIGABRT), Name: "SIGABRT", Description: "abort"},
		{Signal: int(syscall.SIGALRM), Name: "SIGALRM", Description: "alarm timer"},
		{Signal: int(syscall.SIGTERM), Name: "SIGTERM", Description: "terminate"},
		{Signal: int(syscall.SIGKILL), Name: "SIGKILL", Description: "force kill"},
		{Signal: int(syscall.SIGCONT), Name: "SIGCONT", Description: "continue"},
		{Signal: int(syscall.SIGSTOP), Name: "SIGSTOP", Description: "stop execution"},
		{Signal: int(syscall.SIGTSTP), Name: "SIGTSTP", Description: "terminal stop"},
	}
}
