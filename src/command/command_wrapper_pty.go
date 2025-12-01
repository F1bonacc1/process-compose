package command

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/creack/pty"
)

type CmdWrapperPty struct {
	*CmdWrapper
	ptmx *os.File
}

func (c *CmdWrapperPty) GetPty() *os.File {
	return c.ptmx
}

func (c *CmdWrapperPty) Start() (err error) {
	if c.ptmx != nil {
		return nil
	}
	c.ptmx, err = pty.Start(c.cmd)
	if err != nil {
		return fmt.Errorf("error starting PTY command: %w", err)
	}

	// Set initial window size to a reasonable default
	// This prevents programs like 'top' from rendering with tiny/wrong dimensions
	_ = pty.Setsize(c.ptmx, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})

	// No need to capture/restore old state, because we close the PTY when we're done.
	// _, err = term.MakeRaw(int(c.ptmx.Fd()))
	// if err != nil {
	// 	return fmt.Errorf("error putting PTY into raw mode: %w", err)
	// }
	return err
}

func (c *CmdWrapperPty) Wait() error {
	defer c.ptmx.Close()
	return c.cmd.Wait()
}

func (c *CmdWrapperPty) StdoutPipe() (io.ReadCloser, error) {
	if c.ptmx == nil {
		err := c.Start()
		if err != nil {
			return nil, err
		}
	}
	return c.ptmx, nil
}

func (c *CmdWrapperPty) StderrPipe() (io.ReadCloser, error) {
	return nil, errors.New("not supported in PTY")
}
func (c *CmdWrapperPty) StdinPipe() (io.WriteCloser, error) {
	if c.ptmx == nil {
		err := c.Start()
		if err != nil {
			return nil, err
		}
	}
	return c.ptmx, nil
}

func (c *CmdWrapperPty) SetCmdArgs() {
	//c.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
