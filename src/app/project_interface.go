package app

import (
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
)

// IProject holds all the functions from the project struct that are being consumed by the tui package
type IProject interface {
	ShutDownProject() error
	IsRemote() bool
	ErrorForSecs() int
	GetProjectName() (string, error)
	GetProjectState(checkMem bool) (*types.ProjectState, error)

	GetLogLength() int
	GetLogsAndSubscribe(name string, observer pclog.LogObserver) error
	UnSubscribeLogger(name string, observer pclog.LogObserver) error
	GetProcessLog(name string, offsetFromEnd, limit int) ([]string, error)

	GetLexicographicProcessNames() ([]string, error)
	GetProcessInfo(name string) (*types.ProcessConfig, error)
	GetProcessState(name string) (*types.ProcessState, error)
	GetProcessesState() (*types.ProcessesState, error)
	StopProcess(name string) error
	// Alwasy returns non nil(so possibly empty) map.
	// Value in map is `ok` on success, else on error to stop specific process.
	// Iterates all processes (best effort).
	// If all proceses were stopped, error is nil.
	StopProcesses(names []string) (map[string]string, error)
	StartProcess(name string) error
	RestartProcess(name string) error
	ScaleProcess(name string, scale int) error
	GetProcessPorts(name string) (*types.ProcessPorts, error)
	SetProcessPassword(name string, password string) error
	UpdateProject(project *types.Project) (map[string]string, error)
	UpdateProcess(updated *types.ProcessConfig) error
	ReloadProject() (map[string]string, error)
	TruncateProcessLogs(name string) error
}
