package types

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/health"
	"math"
	"time"
)

const DefaultNamespace = "default"
const PlaceHolderValue = "-"

type Processes map[string]ProcessConfig
type Environment []string
type ProcessConfig struct {
	Name              string
	Disabled          bool                   `yaml:"disabled,omitempty"`
	IsDaemon          bool                   `yaml:"is_daemon,omitempty"`
	Command           string                 `yaml:"command"`
	Entrypoint        []string               `yaml:"entrypoint"`
	LogLocation       string                 `yaml:"log_location,omitempty"`
	LoggerConfig      *LoggerConfig          `yaml:"log_configuration,omitempty"`
	Environment       Environment            `yaml:"environment,omitempty"`
	RestartPolicy     RestartPolicyConfig    `yaml:"availability,omitempty"`
	DependsOn         DependsOnConfig        `yaml:"depends_on,omitempty"`
	LivenessProbe     *health.Probe          `yaml:"liveness_probe,omitempty"`
	ReadinessProbe    *health.Probe          `yaml:"readiness_probe,omitempty"`
	ReadyLogLine      string                 `yaml:"ready_log_line,omitempty"`
	ShutDownParams    ShutDownParams         `yaml:"shutdown,omitempty"`
	DisableAnsiColors bool                   `yaml:"disable_ansi_colors,omitempty"`
	WorkingDir        string                 `yaml:"working_dir"`
	Namespace         string                 `yaml:"namespace"`
	Replicas          int                    `yaml:"replicas"`
	Extensions        map[string]interface{} `yaml:",inline"`
	Description       string                 `yaml:"description,omitempty"`
	Vars              Vars                   `yaml:"vars"`
	IsForeground      bool                   `yaml:"is_foreground"`
	IsTty             bool                   `yaml:"is_tty"`
	IsElevated        bool                   `yaml:"is_elevated"`
	ReplicaNum        int
	ReplicaName       string
	Executable        string
	Args              []string
}

func (p *ProcessConfig) GetDependencies() []string {
	dependencies := make([]string, len(p.DependsOn))

	i := 0
	for k := range p.DependsOn {
		dependencies[i] = k
		i++
	}
	return dependencies
}

func (p *ProcessConfig) CalculateReplicaName() string {
	if p.Replicas <= 1 {
		return p.Name
	}
	myWidth := 1 + int(math.Log10(float64(p.Replicas)))
	return fmt.Sprintf("%s-%0*d", p.Name, myWidth, p.ReplicaNum)
}

func (p *ProcessConfig) IsDeferred() bool {
	return p.IsForeground || p.Disabled
}

func NewProcessState(proc *ProcessConfig) *ProcessState {
	state := &ProcessState{
		Name:       proc.ReplicaName,
		Namespace:  proc.Namespace,
		Status:     ProcessStatePending,
		SystemTime: PlaceHolderValue,
		Age:        time.Duration(0),
		IsRunning:  false,
		Health:     ProcessHealthUnknown,
		Restarts:   0,
		ExitCode:   0,
		Mem:        0,
		Pid:        0,
	}
	if proc.Disabled {
		state.Status = ProcessStateDisabled
	} else if proc.IsForeground {
		state.Status = ProcessStateForeground
	}
	return state
}

type ProcessState struct {
	Name             string        `json:"name"`
	Namespace        string        `json:"namespace"`
	Status           string        `json:"status"`
	SystemTime       string        `json:"system_time"`
	Age              time.Duration `json:"age"`
	Health           string        `json:"is_ready"`
	Restarts         int           `json:"restarts"`
	ExitCode         int           `json:"exit_code"`
	Pid              int           `json:"pid"`
	IsElevated       bool          `json:"is_elevated"`
	PasswordProvided bool          `json:"password_provided"`
    Mem              int64         `json:"mem"`
    IsRunning        bool

}

type ProcessPorts struct {
	Name     string   `json:"name"`
	TcpPorts []uint16 `json:"tcp_ports"`
	UdpPorts []uint16 `json:"udp_ports"`
}

type ProcessesState struct {
	States []ProcessState `json:"data"`
}

const (
	RestartPolicyAlways        = "always"
	RestartPolicyOnFailure     = "on_failure"
	RestartPolicyExitOnFailure = "exit_on_failure"
	RestartPolicyNo            = "no"
)

const (
	ProcessStateDisabled    = "Disabled"
	ProcessStateForeground  = "Foreground"
	ProcessStatePending     = "Pending"
	ProcessStateRunning     = "Running"
	ProcessStateLaunching   = "Launching"
	ProcessStateLaunched    = "Launched"
	ProcessStateRestarting  = "Restarting"
	ProcessStateTerminating = "Terminating"
	ProcessStateCompleted   = "Completed"
	ProcessStateSkipped     = "Skipped"
	ProcessStateError       = "Error"
)

const (
	ProcessHealthReady    = "Ready"
	ProcessHealthNotReady = "Not Ready"
	ProcessHealthUnknown  = PlaceHolderValue
)

type RestartPolicyConfig struct {
	Restart        string `yaml:",omitempty"`
	BackoffSeconds int    `yaml:"backoff_seconds,omitempty"`
	MaxRestarts    int    `yaml:"max_restarts,omitempty"`
	ExitOnEnd      bool   `yaml:"exit_on_end,omitempty"`
	ExitOnSkipped  bool   `yaml:"exit_on_skipped,omitempty"`
}

type ShutDownParams struct {
	ShutDownCommand string `yaml:"command,omitempty"`
	ShutDownTimeout int    `yaml:"timeout_seconds,omitempty"`
	Signal          int    `yaml:"signal,omitempty"`
	ParentOnly      bool   `yaml:"parent_only,omitempty"`
}

const (
	// ProcessConditionCompleted is the type for waiting until a process has completed (any exit code).
	ProcessConditionCompleted = "process_completed"

	// ProcessConditionCompletedSuccessfully is the type for waiting until a process has completed successfully (exit code 0).
	ProcessConditionCompletedSuccessfully = "process_completed_successfully"

	// ProcessConditionHealthy is the type for waiting until a process is healthy.
	ProcessConditionHealthy = "process_healthy"

	// ProcessConditionStarted is the type for waiting until a process has started (default).
	ProcessConditionStarted = "process_started"

	// ProcessConditionLogReady is the type for waiting until a process has printed a predefined log line
	ProcessConditionLogReady = "process_log_ready"
)

type DependsOnConfig map[string]ProcessDependency

type ProcessDependency struct {
	Condition  string                 `yaml:",omitempty"`
	Extensions map[string]interface{} `yaml:",inline"`
}
