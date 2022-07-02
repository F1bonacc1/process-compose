package app

func (p *Process) getStartingStateName() string {
	if p.procConf.IsDaemon {
		return ProcessStateLaunching
	}
	return ProcessStateRunning
}
