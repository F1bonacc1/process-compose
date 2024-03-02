package app

import "github.com/f1bonacc1/process-compose/src/types"

type ProjectOpts struct {
	project           *types.Project
	processesToRun    []string
	noDeps            bool
	mainProcess       string
	mainProcessArgs   []string
	isTuiOn           bool
	isOrderedShutDown bool
}

func (p *ProjectOpts) WithProject(project *types.Project) *ProjectOpts {
	p.project = project
	return p
}

func (p *ProjectOpts) WithProcessesToRun(processesToRun []string) *ProjectOpts {
	p.processesToRun = processesToRun
	return p
}
func (p *ProjectOpts) WithNoDeps(noDeps bool) *ProjectOpts {
	p.noDeps = noDeps
	return p
}

func (p *ProjectOpts) WithMainProcess(mainProcess string) *ProjectOpts {
	p.mainProcess = mainProcess
	return p
}

func (p *ProjectOpts) WithMainProcessArgs(mainProcessArgs []string) *ProjectOpts {
	p.mainProcessArgs = mainProcessArgs
	return p
}

func (p *ProjectOpts) WithIsTuiOn(isTuiOn bool) *ProjectOpts {
	p.isTuiOn = isTuiOn
	return p
}

func (p *ProjectOpts) WithOrderedShutDown(isOrderedShutDown bool) *ProjectOpts {
	p.isOrderedShutDown = isOrderedShutDown
	return p
}
