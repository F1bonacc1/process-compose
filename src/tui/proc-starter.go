package tui

import (
	"context"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func (pv *pcView) startProcess() {
	name := pv.getSelectedProcName()
	info, err := pv.project.GetProcessInfo(name)
	if err != nil {
		pv.showError(err.Error())
		return
	}
	if info.IsForeground {
		pv.runForeground(info)
		return
	}
	err = pv.project.StartProcess(name)
	if err != nil {
		pv.showError(err.Error())
	}
}

func (pv *pcView) runForeground(info *types.ProcessConfig) bool {
	pv.halt()
	defer pv.resume()
	return pv.appView.Suspend(func() {
		err := pv.execute(info)
		if err != nil {
			log.Err(err).Msgf("Command failed")
		}
	})
}

func (pv *pcView) execute(info *types.ProcessConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		clearScreen()
	}()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func(cancel context.CancelFunc) {
		select {
		case sig := <-sigChan:
			log.Debug().Msgf("Command canceled with signal %#v", sig)
			cancel()
		case <-ctx.Done():
			log.Debug().Msgf("Foreground process context canceled")
		}
	}(cancel)

	cmd := exec.CommandContext(ctx, info.Executable, info.Args...)
	log.Debug().Str("exec", info.Executable).Strs("args", info.Args).Msg("running start")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	log.Debug().Str("exec", info.Executable).Strs("args", info.Args).Msg("running end")

	return err
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}
