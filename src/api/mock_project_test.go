package api

import (
	"os"

	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
)

// mockProject implements app.IProject for testing API handlers.
type mockProject struct {
	shutDownProjectFn       func() error
	isRemoteFn              func() bool
	errorForSecsFn          func() int
	getProjectNameFn        func() (string, error)
	getProjectStateFn       func(bool) (*types.ProjectState, error)
	getLogLengthFn          func() int
	getLogsAndSubscribeFn   func(string, pclog.LogObserver) error
	unSubscribeLoggerFn     func(string, pclog.LogObserver) error
	getProcessLogFn         func(string, int, int) ([]string, error)
	getLexicographicNamesFn func() ([]string, error)
	getProcessInfoFn        func(string) (*types.ProcessConfig, error)
	getProcessStateFn       func(string) (*types.ProcessState, error)
	getProcessesStateFn     func() (*types.ProcessesState, error)
	stopProcessFn           func(string) error
	stopProcessesFn         func([]string) (map[string]string, error)
	startNamespaceFn        func(string) error
	stopNamespaceFn         func(string) error
	restartNamespaceFn      func(string) error
	getNamespacesFn         func() ([]string, error)
	startProcessFn          func(string) error
	restartProcessFn        func(string) error
	scaleProcessFn          func(string, int) error
	getProcessPortsFn       func(string) (*types.ProcessPorts, error)
	setProcessPasswordFn    func(string, string) error
	updateProjectFn         func(*types.Project) (map[string]string, error)
	updateProcessFn         func(*types.ProcessConfig) error
	reloadProjectFn         func() (map[string]string, error)
	truncateProcessLogsFn   func(string) error
	getProcessPtyFn         func(string) *os.File
	getFullProcessEnvFn     func(*types.ProcessConfig) []string
	getDependencyGraphFn    func() (*types.DependencyGraph, error)
	sendSignalFn            func(string, int) error
}

func (m *mockProject) ShutDownProject() error {
	if m.shutDownProjectFn != nil {
		return m.shutDownProjectFn()
	}
	return nil
}

func (m *mockProject) IsRemote() bool {
	if m.isRemoteFn != nil {
		return m.isRemoteFn()
	}
	return false
}

func (m *mockProject) ErrorForSecs() int {
	if m.errorForSecsFn != nil {
		return m.errorForSecsFn()
	}
	return 0
}

func (m *mockProject) GetProjectName() (string, error) {
	if m.getProjectNameFn != nil {
		return m.getProjectNameFn()
	}
	return "", nil
}

func (m *mockProject) GetProjectState(checkMem bool) (*types.ProjectState, error) {
	if m.getProjectStateFn != nil {
		return m.getProjectStateFn(checkMem)
	}
	return &types.ProjectState{}, nil
}

func (m *mockProject) GetLogLength() int {
	if m.getLogLengthFn != nil {
		return m.getLogLengthFn()
	}
	return 0
}

func (m *mockProject) GetLogsAndSubscribe(name string, observer pclog.LogObserver) error {
	if m.getLogsAndSubscribeFn != nil {
		return m.getLogsAndSubscribeFn(name, observer)
	}
	return nil
}

func (m *mockProject) UnSubscribeLogger(name string, observer pclog.LogObserver) error {
	if m.unSubscribeLoggerFn != nil {
		return m.unSubscribeLoggerFn(name, observer)
	}
	return nil
}

func (m *mockProject) GetProcessLog(name string, offsetFromEnd, limit int) ([]string, error) {
	if m.getProcessLogFn != nil {
		return m.getProcessLogFn(name, offsetFromEnd, limit)
	}
	return nil, nil
}

func (m *mockProject) GetLexicographicProcessNames() ([]string, error) {
	if m.getLexicographicNamesFn != nil {
		return m.getLexicographicNamesFn()
	}
	return nil, nil
}

func (m *mockProject) GetProcessInfo(name string) (*types.ProcessConfig, error) {
	if m.getProcessInfoFn != nil {
		return m.getProcessInfoFn(name)
	}
	return nil, nil
}

func (m *mockProject) GetProcessState(name string) (*types.ProcessState, error) {
	if m.getProcessStateFn != nil {
		return m.getProcessStateFn(name)
	}
	return nil, nil
}

func (m *mockProject) GetProcessesState() (*types.ProcessesState, error) {
	if m.getProcessesStateFn != nil {
		return m.getProcessesStateFn()
	}
	return nil, nil
}

func (m *mockProject) StopProcess(name string) error {
	if m.stopProcessFn != nil {
		return m.stopProcessFn(name)
	}
	return nil
}

func (m *mockProject) SendSignal(name string, sig int) error {
	if m.sendSignalFn != nil {
		return m.sendSignalFn(name, sig)
	}
	return nil
}

func (m *mockProject) StopProcesses(names []string) (map[string]string, error) {
	if m.stopProcessesFn != nil {
		return m.stopProcessesFn(names)
	}
	return nil, nil
}

func (m *mockProject) StartNamespace(namespace string) error {
	if m.startNamespaceFn != nil {
		return m.startNamespaceFn(namespace)
	}
	return nil
}

func (m *mockProject) StopNamespace(namespace string) error {
	if m.stopNamespaceFn != nil {
		return m.stopNamespaceFn(namespace)
	}
	return nil
}

func (m *mockProject) RestartNamespace(namespace string) error {
	if m.restartNamespaceFn != nil {
		return m.restartNamespaceFn(namespace)
	}
	return nil
}

func (m *mockProject) GetNamespaces() ([]string, error) {
	if m.getNamespacesFn != nil {
		return m.getNamespacesFn()
	}
	return nil, nil
}

func (m *mockProject) StartProcess(name string) error {
	if m.startProcessFn != nil {
		return m.startProcessFn(name)
	}
	return nil
}

func (m *mockProject) RestartProcess(name string) error {
	if m.restartProcessFn != nil {
		return m.restartProcessFn(name)
	}
	return nil
}

func (m *mockProject) ScaleProcess(name string, scale int) error {
	if m.scaleProcessFn != nil {
		return m.scaleProcessFn(name, scale)
	}
	return nil
}

func (m *mockProject) GetProcessPorts(name string) (*types.ProcessPorts, error) {
	if m.getProcessPortsFn != nil {
		return m.getProcessPortsFn(name)
	}
	return nil, nil
}

func (m *mockProject) SetProcessPassword(name string, password string) error {
	if m.setProcessPasswordFn != nil {
		return m.setProcessPasswordFn(name, password)
	}
	return nil
}

func (m *mockProject) UpdateProject(project *types.Project) (map[string]string, error) {
	if m.updateProjectFn != nil {
		return m.updateProjectFn(project)
	}
	return nil, nil
}

func (m *mockProject) UpdateProcess(updated *types.ProcessConfig) error {
	if m.updateProcessFn != nil {
		return m.updateProcessFn(updated)
	}
	return nil
}

func (m *mockProject) ReloadProject() (map[string]string, error) {
	if m.reloadProjectFn != nil {
		return m.reloadProjectFn()
	}
	return nil, nil
}

func (m *mockProject) TruncateProcessLogs(name string) error {
	if m.truncateProcessLogsFn != nil {
		return m.truncateProcessLogsFn(name)
	}
	return nil
}

func (m *mockProject) GetProcessPty(name string) *os.File {
	if m.getProcessPtyFn != nil {
		return m.getProcessPtyFn(name)
	}
	return nil
}

func (m *mockProject) GetFullProcessEnvironment(proc *types.ProcessConfig) []string {
	if m.getFullProcessEnvFn != nil {
		return m.getFullProcessEnvFn(proc)
	}
	return nil
}

func (m *mockProject) GetDependencyGraph() (*types.DependencyGraph, error) {
	if m.getDependencyGraphFn != nil {
		return m.getDependencyGraphFn()
	}
	return nil, nil
}
