package types

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/rs/zerolog/log"
	"math"
	"os"
	"reflect"
	"strings"
	"time"
)

const DefaultNamespace = "default"
const PlaceHolderValue = "-"
const DefaultLaunchTimeout = 5

type (
	Processes     map[string]ProcessConfig
	Environment   []string
	EnvCmd        map[string]string
	ProcessConfig struct {
		Name              string
		Disabled          bool                   `yaml:"disabled,omitempty"`
		IsDaemon          bool                   `yaml:"is_daemon,omitempty"`
		Command           string                 `yaml:"command,omitempty"`
		Entrypoint        []string               `yaml:"entrypoint,omitempty"`
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
		WorkingDir        string                 `yaml:"working_dir,omitempty"`
		Namespace         string                 `yaml:"namespace,omitempty"`
		Replicas          int                    `yaml:"replicas,omitempty"`
		Extensions        map[string]interface{} `yaml:",inline"`
		Description       string                 `yaml:"description,omitempty"`
		Vars              Vars                   `yaml:"vars,omitempty"`
		IsForeground      bool                   `yaml:"is_foreground,omitempty"`
		IsTty             bool                   `yaml:"is_tty,omitempty"`
		IsElevated        bool                   `yaml:"is_elevated,omitempty"`
		LaunchTimeout     int                    `yaml:"launch_timeout_seconds,omitempty"`
		OriginalConfig    string
		ReplicaNum        int
		ReplicaName       string
		Executable        string
		Args              []string
	}
)

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

// Compare returns true if two process configs are equal
func (p *ProcessConfig) Compare(another *ProcessConfig) bool {
	if p == nil || another == nil {
		return p == another
	}

	// Compare simple fields
	if p.Name != another.Name ||
		p.Disabled != another.Disabled ||
		p.IsDaemon != another.IsDaemon ||
		p.Command != another.Command ||
		p.LogLocation != another.LogLocation ||
		p.ReadyLogLine != another.ReadyLogLine ||
		p.DisableAnsiColors != another.DisableAnsiColors ||
		p.WorkingDir != another.WorkingDir ||
		p.Namespace != another.Namespace ||
		p.Replicas != another.Replicas ||
		p.Description != another.Description ||
		p.IsForeground != another.IsForeground ||
		p.IsTty != another.IsTty ||
		p.IsElevated != another.IsElevated {
		return false
	}

	if !reflect.DeepEqual(p.LoggerConfig, another.LoggerConfig) ||
		!reflect.DeepEqual(p.LivenessProbe, another.LivenessProbe) ||
		!reflect.DeepEqual(p.ReadinessProbe, another.ReadinessProbe) ||
		!reflect.DeepEqual(p.ShutDownParams, another.ShutDownParams) ||
		!reflect.DeepEqual(p.Vars, another.Vars) ||
		!reflect.DeepEqual(p.Extensions, another.Extensions) ||
		!reflect.DeepEqual(p.DependsOn, another.DependsOn) ||
		!reflect.DeepEqual(p.RestartPolicy, another.RestartPolicy) ||
		!reflect.DeepEqual(p.Environment, another.Environment) ||
		!reflect.DeepEqual(p.Args, another.Args) {
		//diffs := compareStructs(*p, *another)
		//log.Warn().Msgf("Structs are different: %s", diffs)
		return false
	}

	return true
}
func (p *ProcessConfig) AssignProcessExecutableAndArgs(shellConf *command.ShellConfig, elevatedShellArg string) {
	if p.Command != "" || len(p.Entrypoint) == 0 {
		if len(p.Entrypoint) > 0 {
			message := fmt.Sprintf("'command' and 'entrypoint' are set! Using command (process: %s)", p.Name)
			_, _ = fmt.Fprintln(os.Stderr, "process-compose:", message)
			log.Warn().Msg(message)
		}

		p.Executable = shellConf.ShellCommand

		if len(p.Command) == 0 {
			return
		}
		if p.IsElevated {
			p.Args = []string{shellConf.ShellArgument, fmt.Sprintf("%s %s %s", shellConf.ElevatedShellCmd, elevatedShellArg, p.Command)}
		} else {
			p.Args = []string{shellConf.ShellArgument, p.Command}
		}
	} else {
		if p.IsElevated {
			p.Entrypoint = append([]string{shellConf.ElevatedShellCmd, elevatedShellArg}, p.Entrypoint...)
		}
		p.Executable = p.Entrypoint[0]
		p.Args = p.Entrypoint[1:]
	}
}

func (p *ProcessConfig) ValidateProcessConfig() error {
	if len(p.Extensions) == 0 {
		return nil // no error
	}
	for extKey := range p.Extensions {
		if strings.HasPrefix(extKey, "x-") {
			continue
		}
		return fmt.Errorf("unknown key '%s' found in process '%s'", extKey, p.Name)
	}

	return nil
}

func compareStructs(a, b interface{}) []string {
	var differences []string
	aValue := reflect.ValueOf(a)
	bValue := reflect.ValueOf(b)

	if aValue.Type() != bValue.Type() {
		return []string{"Types are different"}
	}

	for i := 0; i < aValue.NumField(); i++ {
		aField := aValue.Field(i)
		bField := bValue.Field(i)
		fieldName := aValue.Type().Field(i).Name

		if !reflect.DeepEqual(aField.Interface(), bField.Interface()) {
			differences = append(differences, fmt.Sprintf("Field %s differs: %v != %v", fieldName, aField, bField))
		}
	}

	return differences
}

func NewProcessState(proc *ProcessConfig) *ProcessState {
	state := &ProcessState{
		Name:           proc.ReplicaName,
		Namespace:      proc.Namespace,
		Status:         ProcessStatePending,
		SystemTime:     PlaceHolderValue,
		Age:            time.Duration(0),
		IsRunning:      false,
		Health:         ProcessHealthUnknown,
		HasHealthProbe: proc.ReadinessProbe != nil || proc.LivenessProbe != nil,
		Restarts:       0,
		ExitCode:       0,
		Mem:            0,
		CPU:            0,
		Pid:            0,
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
	Age              time.Duration `json:"age" swaggertype:"primitive,integer"`
	Health           string        `json:"is_ready"`
	HasHealthProbe   bool          `json:"has_ready_probe"`
	Restarts         int           `json:"restarts"`
	ExitCode         int           `json:"exit_code"`
	Pid              int           `json:"pid"`
	IsElevated       bool          `json:"is_elevated"`
	PasswordProvided bool          `json:"password_provided"`
	Mem              int64         `json:"mem"`
	CPU              float64       `json:"cpu"`
	IsRunning        bool          `json:"is_running"`
}

type ProcessPorts struct {
	Name     string   `json:"name"`
	TcpPorts []uint16 `json:"tcp_ports"`
	UdpPorts []uint16 `json:"udp_ports"`
}

type ProcessesState struct {
	States []ProcessState `json:"data"`
}

func (p *ProcessesState) IsReady() bool {
	for _, state := range p.States {
		if !state.IsReady() {
			return false
		}
	}
	return true
}

// Check if a process is running and healthy.
//
// If `hasHealthProbe` is true, the process must be healthy to be considered
// ready.
func (p *ProcessState) IsReady() bool {
	isReady, _ := p.IsReadyReason()
	return isReady
}

// Check if a process is running and healthy and explain why.
//
// If `hasHealthProbe` is true, the process must be healthy to be considered
// ready.
//
// The explanation may be empty.
func (p *ProcessState) IsReadyReason() (bool, string) {
	if p.Status != ProcessStateRunning &&
		p.Status != ProcessStateForeground &&
		p.Status != ProcessStateLaunched &&
		p.Status != ProcessStateCompleted &&
		p.Status != ProcessStateSkipped &&
		p.Status != ProcessStateDisabled &&
		p.Status != ProcessStateRestarting {
		return false, fmt.Sprintf("status is %s", p.Status)
	} else if p.Status == ProcessStateDisabled {
		return true, "process is disabled"
	} else if p.HasHealthProbe && p.Health != ProcessHealthReady {
		health := p.Health
		if health == ProcessHealthUnknown {
			// `ProcessHealthUnknown` is `-`, which looks fine in the TUI's table view
			// but weird in logs.
			health = "Unknown"
		}
		return false, fmt.Sprintf("health is %s", health)
	} else if p.Health != ProcessHealthReady && p.Health != ProcessHealthUnknown {
		return false, fmt.Sprintf("health is %s", p.Health)
	} else if p.ExitCode != 0 {
		return false, fmt.Sprintf("failed with exit code %d", p.ExitCode)
	}
	return true, ""
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

const (
	ProcessUpdateUpdated = "updated"
	ProcessUpdateRemoved = "removed"
	ProcessUpdateAdded   = "added"
	ProcessUpdateError   = "error"
)
