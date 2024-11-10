package app

import "github.com/f1bonacc1/process-compose/src/types"

func (p *Process) waitForDaemonCompletion() {
	if !p.isDaemonLaunched() {
		return
	}

loop:
	for {
		status := <-p.procStateChan
		switch status {
		case types.ProcessStateCompleted:
			break loop
		}
	}

}

func (p *Process) notifyDaemonStopped() {
	if p.procConf.IsDaemon {
		p.procStateChan <- types.ProcessStateCompleted
	}
}

func (p *Process) isDaemonLaunched() bool {
	return p.procConf.IsDaemon && p.procState.ExitCode == 0
}
