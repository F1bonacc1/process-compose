package command

import (
	"io"
	"os"
)

type Commander interface {
	Stop(sig int, _parentOnly bool) error
	SetCmdArgs()
	Start() error
	Run() error
	Wait() error
	ExitCode() int
	Pid() int
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	StdinPipe() (io.WriteCloser, error)
	AttachIo()
	SetEnv(env []string)
	SetDir(dir string)
	Output() ([]byte, error)
	CombinedOutput() ([]byte, error)
	GetPty() *os.File
}
