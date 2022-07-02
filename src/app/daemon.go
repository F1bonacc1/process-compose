package app

func (p *Process) waitForDaemonCompletion() {
	if !p.isDaemonLaunched() {
		return
	}

loop:
	for {
		status := <-p.procStateChan
		switch status {
		case ProcessStateCompleted:
			break loop
		}
	}

}

func (p *Process) notifyDaemonStopped() {
	if p.isDaemonLaunched() {
		p.procStateChan <- ProcessStateCompleted
	}
}

func (p *Process) isDaemonLaunched() bool {
	return p.procConf.IsDaemon && p.procState.ExitCode == 0
}
