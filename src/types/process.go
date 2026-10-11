package types

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const DefaultNamespace = "default"
const PlaceHolderValue = "-"
const DefaultLaunchTimeout = 5

type (
	Processes     map[string]ProcessConfig
	Environment   []string
	EnvCmd        map[string]string
	ProcessConfig struct {
		Name                    string              `yaml:",omitempty" json:"name,omitempty"`
		Extends                 string              `yaml:"extends,omitempty" json:"extends,omitempty"`
		Disabled                bool                `yaml:"disabled,omitempty" json:"disabled,omitempty"`
		IsDaemon                bool                `yaml:"is_daemon,omitempty" json:"isDaemon,omitempty"`
		Command                 string              `yaml:"command,omitempty" json:"command,omitempty"`
		Entrypoint              []string            `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
		LogLocation             string              `yaml:"log_location,omitempty" json:"logLocation,omitempty"`
		LoggerConfig            *LoggerConfig       `yaml:"log_configuration,omitempty" json:"loggerConfig,omitempty"`
		Environment             Environment         `yaml:"environment,omitempty" json:"environment,omitempty"`
		EnvFile                 string              `yaml:"env_file,omitempty" json:"envFile,omitempty"`
		RestartPolicy           RestartPolicyConfig `yaml:"availability,omitempty" json:"restartPolicy"`
		DependsOn               DependsOnConfig     `yaml:"depends_on,omitempty" json:"dependsOn,omitempty"`
		LivenessProbe           *health.Probe       `yaml:"liveness_probe,omitempty" json:"livenessProbe,omitempty"`
		ReadinessProbe          *health.Probe       `yaml:"readiness_probe,omitempty" json:"readinessProbe,omitempty"`
		ReadyLogLine            string              `yaml:"ready_log_line,omitempty" json:"readyLogLine,omitempty"`
		ShutDownParams          ShutDownParams      `yaml:"shutdown,omitempty" json:"shutDownParams"`
		DisableAnsiColors       bool                `yaml:"disable_ansi_colors,omitempty" json:"disableAnsiColors,omitempty"`
		WorkingDir              string              `yaml:"working_dir,omitempty" json:"workingDir,omitempty"`
		Namespace               Namespaces          `yaml:"namespace,omitempty" json:"namespace,omitempty"`
		Replicas                int                 `yaml:"replicas,omitempty" json:"replicas,omitempty"`
		Extensions              map[string]any      `yaml:",inline" json:"extensions,omitempty"`
		Description             string              `yaml:"description,omitempty" json:"description,omitempty"`
		Vars                    Vars                `yaml:"vars,omitempty" json:"vars,omitempty"`
		IsForeground            bool                `yaml:"is_foreground,omitempty" json:"isForeground,omitempty"`
		IsTty                   bool                `yaml:"is_tty,omitempty" json:"isTty,omitempty"`
		IsElevated              bool                `yaml:"is_elevated,omitempty" json:"isElevated,omitempty"`
		IsInteractive           bool                `yaml:"is_interactive,omitempty" json:"isInteractive,omitempty"`
		LaunchTimeout           int                 `yaml:"launch_timeout_seconds,omitempty" json:"launchTimeout,omitempty"`
		IsDisabled              string              `yaml:"is_disabled,omitempty" json:"isDisabled,omitempty"`
		DisableDotEnv           bool                `yaml:"is_dotenv_disabled,omitempty" json:"disableDotEnv,omitempty"`
		OriginalConfig          string              `yaml:"original_config,omitempty" json:"originalConfig,omitempty"`
		ReplicaNum              int                 `yaml:"replica_num,omitempty" json:"replicaNum,omitempty"`
		ReplicaName             string              `yaml:"replica_name,omitempty" json:"replicaName,omitempty"`
		Executable              string              `yaml:"executable,omitempty" json:"executable,omitempty"`
		Args                    []string            `yaml:"args,omitempty" json:"args,omitempty"`
		Schedule                *ScheduleConfig     `yaml:"schedule,omitempty" json:"schedule,omitempty"`
		Watch                   *WatchConfig        `yaml:"watch,omitempty" json:"watch,omitempty"`
		MCP                     *MCPProcessConfig   `yaml:"mcp,omitempty" json:"mcp,omitempty"`
		TruncateLog             bool                `yaml:"truncate_log,omitempty" json:"truncateLog,omitempty"`
		DisableCommandRendering bool                `yaml:"is_template_disabled,omitempty" json:"disableCommandRendering,omitempty"`
		MonitorFor              MonitorFor          `yaml:"monitor_for,omitempty" json:"monitorFor,omitempty" jsonschema:"type=string,enum=none,enum=activity,enum=silence"`
		MonitorSilenceThreshold time.Duration       `yaml:"monitor_silence_threshold,omitempty" json:"monitorSilenceThreshold,omitempty"`
		SuccessExitCodes        []int               `yaml:"success_exit_codes,omitempty" json:"successExitCodes,omitempty"`
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

// IsMCP returns true if this process is MCP-enabled
func (p *ProcessConfig) IsMCP() bool {
	return p.MCP != nil
}

// IsMCPTool returns true if this is an MCP tool
func (p *ProcessConfig) IsMCPTool() bool {
	return p.MCP != nil && p.MCP.IsTool()
}

// IsMCPResource returns true if this is an MCP resource
func (p *ProcessConfig) IsMCPResource() bool {
	return p.MCP != nil && p.MCP.IsResource()
}

// isExitCodeSuccess reports whether a process exit code should be treated as a
// successful exit. Code 0 is always successful; any code listed in successCodes
// is additionally treated as success. This is how `success_exit_codes` lets
// signal-induced exits (128+signal, e.g. 130 for SIGINT, 143 for SIGTERM) count
// as healthy outcomes rather than failures (systemd's `SuccessExitStatus`).
func isExitCodeSuccess(code int, successCodes []int) bool {
	return code == 0 || slices.Contains(successCodes, code)
}

// IsExitCodeSuccess reports whether code is a successful exit for this process,
// honoring its configured SuccessExitCodes allowlist.
func (p *ProcessConfig) IsExitCodeSuccess(code int) bool {
	return isExitCodeSuccess(code, p.SuccessExitCodes)
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
		p.EnvFile != another.EnvFile ||
		p.WorkingDir != another.WorkingDir ||
		!p.Namespace.Equal(another.Namespace) ||
		p.Replicas != another.Replicas ||
		p.Description != another.Description ||
		p.IsForeground != another.IsForeground ||
		p.IsTty != another.IsTty ||
		p.IsInteractive != another.IsInteractive ||
		p.IsElevated != another.IsElevated {
		return false
	}

	return p.compareCompositeFields(another)
}

// compareCompositeFields compares the fields that need a deep comparison. It is
// split out of Compare to keep that function within the project's cyclomatic
// complexity limit; add new composite fields here.
func (p *ProcessConfig) compareCompositeFields(another *ProcessConfig) bool {
	composites := []struct{ a, b any }{
		{p.LoggerConfig, another.LoggerConfig},
		{p.LivenessProbe, another.LivenessProbe},
		{p.ReadinessProbe, another.ReadinessProbe},
		{p.ShutDownParams, another.ShutDownParams},
		{p.Vars, another.Vars},
		{p.Extensions, another.Extensions},
		{p.DependsOn, another.DependsOn},
		{p.RestartPolicy, another.RestartPolicy},
		{p.Environment, another.Environment},
		{p.Args, another.Args},
		{p.Watch, another.Watch},
		{p.SuccessExitCodes, another.SuccessExitCodes},
	}
	for _, field := range composites {
		if !reflect.DeepEqual(field.a, field.b) {
			//diffs := compareStructs(*p, *another)
			//log.Warn().Msgf("Structs are different: %s", diffs)
			return false
		}
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
	for _, code := range p.SuccessExitCodes {
		if code < 0 || code > 255 {
			return fmt.Errorf("invalid success_exit_codes value %d in process '%s': exit codes must be in the range 0-255", code, p.Name)
		}
	}
	if p.ShutDownParams.SendKeys != "" && !p.IsInteractive && !p.IsTty {
		return fmt.Errorf("process '%s': shutdown.send_keys requires is_interactive (or is_tty)", p.Name)
	}
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

func compareStructs(a, b any) []string {
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
		Name:             proc.ReplicaName,
		Namespace:        proc.Namespace.OrDefault(),
		Status:           ProcessStatePending,
		SystemTime:       PlaceHolderValue,
		Age:              time.Duration(0),
		IsRunning:        false,
		Health:           ProcessHealthUnknown,
		HasHealthProbe:   proc.ReadinessProbe != nil || proc.LivenessProbe != nil,
		Restarts:         0,
		ExitCode:         0,
		SuccessExitCodes: proc.SuccessExitCodes,
		Mem:              0,
		CPU:              0,
		Pid:              0,
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
	Namespace        Namespaces    `json:"namespace" swaggertype:"array,string"`
	Status           string        `json:"status"`
	SystemTime       string        `json:"system_time"`
	Age              time.Duration `json:"age" swaggertype:"primitive,integer"`
	Health           string        `json:"is_ready"`
	HasHealthProbe   bool          `json:"has_ready_probe"`
	Restarts         int           `json:"restarts"`
	ExitCode         int           `json:"exit_code"`
	SuccessExitCodes []int         `json:"success_exit_codes,omitempty"`
	Pid              int           `json:"pid"`
	IsElevated       bool          `json:"is_elevated"`
	PasswordProvided bool          `json:"password_provided"`
	Mem              int64         `json:"mem"`
	CPU              float64       `json:"cpu"`
	IsRunning        bool          `json:"is_running"`
	NextRunTime      *time.Time    `json:"next_run_time,omitempty"`
	LastActivityTime *time.Time    `json:"last_activity_time,omitempty"`
	// IsWatched reports whether an active file watcher is armed for this
	// process. It is the `watch` counterpart of NextRunTime and, like it, must
	// stay in the JSON payload - the remote client deserializes this struct, so
	// attached sessions would otherwise never see the Watching state.
	IsWatched bool `json:"is_watched,omitempty"`
	// WatchTriggerPath and WatchTriggerTime describe the file change that last
	// restarted this process. They travel in the state, rather than through a
	// local callback, so that an attached (remote) TUI can report them too.
	WatchTriggerPath string     `json:"watch_trigger_path,omitempty"`
	WatchTriggerTime *time.Time `json:"watch_trigger_time,omitempty"`
	MaxLogicalLine   int64      `json:"-"` // TUI-only: furthest logical line reached in terminal
	// ProcessStartTime is the wall-clock time the process (first) entered a
	// running/launched state. Used by `process-compose analyze critical-chain`.
	ProcessStartTime *time.Time `json:"process_start_time,omitempty"`
	// ProcessReadyTime is the wall-clock time the process became ready. For
	// processes with a readiness probe or a `ready_log_line`, this is set when
	// the probe succeeds / the line is observed. For processes without any
	// readiness probe, it equals ProcessStartTime.
	ProcessReadyTime *time.Time `json:"process_ready_time,omitempty"`
	// ProcessEndTime is the wall-clock time the process ended (completed,
	// errored, terminated, or was skipped).
	ProcessEndTime *time.Time `json:"process_end_time,omitempty"`
}

type ProcessPorts struct {
	Name     string   `json:"name"`
	TcpPorts []uint16 `json:"tcp_ports"`
	UdpPorts []uint16 `json:"udp_ports"`
}

type ProcessesState struct {
	States []ProcessState `json:"data"`
}

// ProcessStateEvent is published whenever a process's observable state
// (Status, Health, or terminal exit info) changes. It is also emitted as a
// snapshot for every existing process when a subscriber first connects.
type ProcessStateEvent struct {
	// Snapshot is true for events emitted as part of the initial replay on
	// subscribe, false for live transitions.
	Snapshot bool `json:"snapshot,omitempty"`
	// State is a self-contained copy of the process state at the moment of
	// the event.
	State ProcessState `json:"state"`
}

// StateObserver consumes process state events. Implementations must be safe
// for concurrent calls; Notify is invoked while the broadcaster's mutex is
// held, so observers should not block on slow I/O.
//
// The interface lives next to ProcessStateEvent so consumers (api, client,
// tui) can implement and reference it without importing the heavier app
// package.
type StateObserver interface {
	// Notify is called for every event delivered to this observer. If the
	// observer cannot accept the event (e.g. its buffer is full), it should
	// drop or close itself rather than block.
	Notify(ev ProcessStateEvent)
	// UniqueID identifies this observer for subscribe/unsubscribe.
	UniqueID() string
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

// IsExitCodeSuccess reports whether this process's exit code is considered a
// success, honoring its SuccessExitCodes allowlist (carried from the config).
func (p *ProcessState) IsExitCodeSuccess() bool {
	return isExitCodeSuccess(p.ExitCode, p.SuccessExitCodes)
}

// IsWatchIdle reports whether this process has finished cleanly and is only
// waiting for a watched file to change.
//
// A failed process is deliberately excluded. "Watching" must never stand in for
// "Failed": a broken build is the single most important thing a watch loop has
// to tell the user, and a state that hid it would be worse than no state at
// all. The IsWatched flag still travels in the state, so a caller that wants to
// show "armed" alongside a failure can.
//
// Callers must combine this with IsWatched - it deliberately says nothing about
// whether a watch exists.
func (p *ProcessState) IsWatchIdle() bool {
	return !p.IsRunning &&
		p.Status == ProcessStateCompleted &&
		p.IsExitCodeSuccess()
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
	} else if !p.IsExitCodeSuccess() {
		return false, fmt.Sprintf("failed with exit code %d", p.ExitCode)
	}
	return true, ""
}

//go:generate stringer -type=RestartPolicy
type RestartPolicy int

const (
	RestartPolicyNo RestartPolicy = iota
	RestartPolicyAlways
	RestartPolicyOnFailure
	RestartPolicyExitOnFailure
)

func (p *RestartPolicy) UnmarshalYAML(node *yaml.Node) error {
	var value string
	if err := node.Decode(&value); err != nil {
		return err
	}
	switch value {
	case "always":
		*p = RestartPolicyAlways
	case "on_failure":
		*p = RestartPolicyOnFailure
	case "exit_on_failure":
		*p = RestartPolicyExitOnFailure
	case "no":
		*p = RestartPolicyNo
	default:
		return fmt.Errorf("invalid restart policy: %q", value)
	}
	return nil
}

func (p RestartPolicy) MarshalYAML() (any, error) {
	switch p {
	case RestartPolicyNo:
		return "no", nil
	case RestartPolicyAlways:
		return "always", nil
	case RestartPolicyOnFailure:
		return "on_failure", nil
	case RestartPolicyExitOnFailure:
		return "exit_on_failure", nil
	default:
		return nil, fmt.Errorf("invalid restart policy: %d", p)
	}
}

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
	ProcessStateScheduled   = "Scheduled"
	ProcessStateWatching    = "Watching"
)

// Display a process status for the UI.
//
// In particular, this displays "Failed" if the process has completed with a
// non-zero exit code. This makes it clearer when a process has failed, as
// opposed to exiting successfully.
//
// We can't change the `Status` field to "Failed" directly because that would
// change the JSON API behavior, but we can change it in the TUI.
func DisplayProcessStatus(state ProcessState) string {
	if state.NextRunTime != nil && !state.IsRunning {
		return ProcessStateScheduled
	}
	// A process that exited cleanly but still has an active file watcher is
	// armed rather than finished - showing "Completed" would suggest nothing
	// more can happen, and would leave the project's continued running
	// unexplained. A failed one keeps its failure; see IsWatchIdle.
	if state.IsWatched && state.IsWatchIdle() {
		return ProcessStateWatching
	}
	if state.Status == ProcessStateCompleted && !state.IsExitCodeSuccess() {
		return "Failed"
	}
	return state.Status
}

const (
	ProcessHealthReady    = "Ready"
	ProcessHealthNotReady = "Not Ready"
	ProcessHealthUnknown  = PlaceHolderValue
)

type RestartPolicyConfig struct {
	Restart        RestartPolicy `yaml:",omitempty" json:"restart,omitempty" jsonschema:"type=string,enum=always,enum=on_failure,enum=exit_on_failure,enum=no"`
	BackoffSeconds int           `yaml:"backoff_seconds,omitempty" json:"backoffSeconds,omitempty"`
	MaxRestarts    int           `yaml:"max_restarts,omitempty" json:"maxRestarts,omitempty"`
	ExitOnEnd      bool          `yaml:"exit_on_end,omitempty" json:"exitOnEnd,omitempty"`
	ExitOnSkipped  bool          `yaml:"exit_on_skipped,omitempty" json:"exitOnSkipped,omitempty"`
}

type ShutDownParams struct {
	ShutDownCommand string `yaml:"command,omitempty" json:"shutDownCommand,omitempty"`
	ShutDownTimeout int    `yaml:"timeout_seconds,omitempty" json:"shutDownTimeout,omitempty"`
	Signal          int    `yaml:"signal,omitempty" json:"signal,omitempty"`
	ParentOnly      bool   `yaml:"parent_only,omitempty" json:"parentOnly,omitempty"`
	SendKeys        string `yaml:"send_keys,omitempty" json:"sendKeys,omitempty"`
}

//go:generate stringer -type=ProcessCondition
type ProcessCondition int

const (
	// ProcessConditionCompleted is the type for waiting until a process has completed (any exit code).
	ProcessConditionCompleted ProcessCondition = iota
	// ProcessConditionCompletedSuccessfully is the type for waiting until a process has completed successfully (exit code 0).
	ProcessConditionCompletedSuccessfully
	// ProcessConditionHealthy is the type for waiting until a process is healthy.
	ProcessConditionHealthy
	// ProcessConditionStarted is the type for waiting until a process has started (default).
	ProcessConditionStarted
	// ProcessConditionLogReady is the type for waiting until a process has printed a predefined log line
	ProcessConditionLogReady
)

func (c *ProcessCondition) UnmarshalYAML(node *yaml.Node) error {
	var value string
	if err := node.Decode(&value); err != nil {
		return err
	}
	switch value {
	case "process_completed":
		*c = ProcessConditionCompleted
	case "process_completed_successfully":
		*c = ProcessConditionCompletedSuccessfully
	case "process_healthy":
		*c = ProcessConditionHealthy
	case "process_started":
		*c = ProcessConditionStarted
	case "process_log_ready":
		*c = ProcessConditionLogReady
	default:
		return fmt.Errorf("invalid process dependency condition: %q", value)
	}
	return nil
}

func (c ProcessCondition) MarshalYAML() (any, error) {
	switch c {
	case ProcessConditionCompleted:
		return "process_completed", nil
	case ProcessConditionCompletedSuccessfully:
		return "process_completed_successfully", nil
	case ProcessConditionHealthy:
		return "process_healthy", nil
	case ProcessConditionStarted:
		return "process_started", nil
	case ProcessConditionLogReady:
		return "process_log_ready", nil
	default:
		return nil, fmt.Errorf("invalid process condition: %d", c)
	}
}

//go:generate stringer -type=MonitorFor
type MonitorFor int

const (
	MonitorForNone     MonitorFor = iota // default - no monitoring
	MonitorForActivity                   // notify on new output while unfocused
	MonitorForSilence                    // notify on no output while unfocused
)

func (m *MonitorFor) UnmarshalYAML(node *yaml.Node) error {
	var value string
	if err := node.Decode(&value); err != nil {
		return err
	}
	switch value {
	case "none", "":
		*m = MonitorForNone
	case "activity":
		*m = MonitorForActivity
	case "silence":
		*m = MonitorForSilence
	default:
		return fmt.Errorf("invalid monitor_for value: %q", value)
	}
	return nil
}

func (m MonitorFor) MarshalYAML() (any, error) {
	switch m {
	case MonitorForNone:
		return "none", nil
	case MonitorForActivity:
		return "activity", nil
	case MonitorForSilence:
		return "silence", nil
	default:
		return nil, fmt.Errorf("invalid monitor_for value: %d", m)
	}
}

// Where key is process name.
type DependsOnConfig map[string]ProcessDependency

type ProcessDependency struct {
	Condition  ProcessCondition `yaml:",omitempty" json:"condition,omitempty" jsonschema:"type=string,enum=process_started,enum=process_healthy,enum=process_completed,enum=process_completed_successfully,enum=process_log_ready"`
	Extensions map[string]any   `yaml:",inline" json:"extensions,omitempty"`
}

const (
	ProcessUpdateUpdated = "updated"
	ProcessUpdateRemoved = "removed"
	ProcessUpdateAdded   = "added"
	ProcessUpdateError   = "error"
)
