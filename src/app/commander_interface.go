package app

import (
	"io"
)

type Commander interface {
	Start() error
	Wait() error
	ExitCode() int
	Pid() int
	SetEnv([]string)
	SetDir(string)
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Stop(int, bool) error
	SetCmdArgs()
	AttachIo()
}
