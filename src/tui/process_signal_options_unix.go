//go:build !windows

package tui

import "syscall"

func availableSignalOptions() []processSignalOption {
	return []processSignalOption{
		{Signal: int(syscall.SIGHUP), Name: "SIGHUP", Description: "hang up controlling terminal or process"},
		{Signal: int(syscall.SIGINT), Name: "SIGINT", Description: "interrupt from keyboard"},
		{Signal: int(syscall.SIGQUIT), Name: "SIGQUIT", Description: "quit from keyboard"},
		{Signal: int(syscall.SIGILL), Name: "SIGILL", Description: "illegal instruction"},
		{Signal: int(syscall.SIGABRT), Name: "SIGABRT", Description: "abnormal termination"},
		{Signal: int(syscall.SIGFPE), Name: "SIGFPE", Description: "floating-point exception"},
		{Signal: int(syscall.SIGKILL), Name: "SIGKILL", Description: "forced process termination"},
		{Signal: int(syscall.SIGUSR1), Name: "SIGUSR1", Description: "user-defined signal 1"},
		{Signal: int(syscall.SIGSEGV), Name: "SIGSEGV", Description: "invalid memory reference"},
		{Signal: int(syscall.SIGUSR2), Name: "SIGUSR2", Description: "user-defined signal 2"},
		{Signal: int(syscall.SIGPIPE), Name: "SIGPIPE", Description: "write to pipe with no readers"},
		{Signal: int(syscall.SIGALRM), Name: "SIGALRM", Description: "real-time clock alarm"},
		{Signal: int(syscall.SIGTERM), Name: "SIGTERM", Description: "process termination"},
		{Signal: int(syscall.SIGCHLD), Name: "SIGCHLD", Description: "child process stopped or terminated"},
		{Signal: int(syscall.SIGCONT), Name: "SIGCONT", Description: "resume execution if stopped"},
		{Signal: int(syscall.SIGSTOP), Name: "SIGSTOP", Description: "stop process execution"},
		{Signal: int(syscall.SIGTSTP), Name: "SIGTSTP", Description: "terminal stop"},
		{Signal: int(syscall.SIGTTIN), Name: "SIGTTIN", Description: "background process requires input"},
		{Signal: int(syscall.SIGTTOU), Name: "SIGTTOU", Description: "background process requires output"},
	}
}
