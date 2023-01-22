package types

import "github.com/f1bonacc1/process-compose/src/health"

type Processes map[string]ProcessConfig
type Environment []string
type ProcessConfig struct {
	Name              string
	Disabled          bool                   `yaml:"disabled,omitempty"`
	IsDaemon          bool                   `yaml:"is_daemon,omitempty"`
	Command           string                 `yaml:"command"`
	LogLocation       string                 `yaml:"log_location,omitempty"`
	Environment       Environment            `yaml:"environment,omitempty"`
	RestartPolicy     RestartPolicyConfig    `yaml:"availability,omitempty"`
	DependsOn         DependsOnConfig        `yaml:"depends_on,omitempty"`
	LivenessProbe     *health.Probe          `yaml:"liveness_probe,omitempty"`
	ReadinessProbe    *health.Probe          `yaml:"readiness_probe,omitempty"`
	ShutDownParams    ShutDownParams         `yaml:"shutdown,omitempty"`
	DisableAnsiColors bool                   `yaml:"disable_ansi_colors,omitempty"`
	WorkingDir        string                 `yaml:"working_dir"`
	Extensions        map[string]interface{} `yaml:",inline"`
}

func (p ProcessConfig) GetDependencies() []string {
	dependencies := make([]string, len(p.DependsOn))

	i := 0
	for k := range p.DependsOn {
		dependencies[i] = k
		i++
	}
	return dependencies
}

type ProcessState struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	SystemTime string `json:"system_time"`
	Health     string `json:"is_ready"`
	Restarts   int    `json:"restarts"`
	ExitCode   int    `json:"exit_code"`
	Pid        int    `json:"pid"`
	IsRunning  bool
}

type ProcessStates struct {
	States []ProcessState `json:"data"`
}

const (
	RestartPolicyAlways              = "always"
	RestartPolicyOnFailureDeprecated = "on-failure"
	RestartPolicyOnFailure           = "on_failure"
	RestartPolicyExitOnFailure       = "exit_on_failure"
	RestartPolicyNo                  = "no"
)

const (
	ProcessStateDisabled    = "Disabled"
	ProcessStatePending     = "Pending"
	ProcessStateRunning     = "Running"
	ProcessStateLaunching   = "Launching"
	ProcessStateLaunched    = "Launched"
	ProcessStateRestarting  = "Restarting"
	ProcessStateTerminating = "Terminating"
	ProcessStateCompleted   = "Completed"
	ProcessStateError       = "Error"
)

const (
	ProcessHealthReady    = "Ready"
	ProcessHealthNotReady = "Not Ready"
	ProcessHealthUnknown  = "N/A"
)

type RestartPolicyConfig struct {
	Restart        string `yaml:",omitempty"`
	BackoffSeconds int    `yaml:"backoff_seconds,omitempty"`
	MaxRestarts    int    `yaml:"max_restarts,omitempty"`
}

type ShutDownParams struct {
	ShutDownCommand string `yaml:"command,omitempty"`
	ShutDownTimeout int    `yaml:"timeout_seconds,omitempty"`
	Signal          int    `yaml:"signal,omitempty"`
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
)

type DependsOnConfig map[string]ProcessDependency

type ProcessDependency struct {
	Condition  string                 `yaml:",omitempty"`
	Extensions map[string]interface{} `yaml:",inline"`
}
