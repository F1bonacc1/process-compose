package config

import (
	"math"
	"time"
)

const (
	// DefaultRefreshRate represents the refresh interval.
	DefaultRefreshRate = 1 * time.Second

	// DefaultLogLevel represents the default log level.
	DefaultLogLevel = "info"

	// DefaultPortNum represents the default port number.
	DefaultPortNum = 8080

	// DefaultAddress represents the default address.
	DefaultAddress = "localhost"

	// DefaultLogLength represents the default log length.
	DefaultLogLength = 1000

	// DefaultSortColumn represents the default sort column.
	DefaultSortColumn = "NAME"

	// DefaultThemeName represents the default theme
	DefaultThemeName = "Default"
)

const (
	EnvVarNamePort          = "PC_PORT_NUM"
	EnvVarNameTui           = "PC_DISABLE_TUI"
	EnvVarNameConfig        = "PC_CONFIG_FILES"
	EnvVarNameShortcuts     = "PC_SHORTCUTS_FILES"
	EnvVarNameNoServer      = "PC_NO_SERVER"
	EnvVarUnixSocketPath    = "PC_SOCKET_PATH"
	EnvVarReadOnlyMode      = "PC_READ_ONLY"
	EnvVarDisableDotEnv     = "PC_DISABLE_DOTENV"
	EnvVarTuiFullScreen     = "PC_TUI_FULL_SCREEN"
	EnvVarHideDisabled      = "PC_HIDE_DISABLED_PROC"
	EnvVarDisabledProcesses = "PC_DISABLE"
	EnvVarEnabledProcesses  = "PC_ENABLE"
)

// Flags represents PC configuration flags.
type Flags struct {
	RefreshRate       *time.Duration
	PortNum           *int
	Address           *string
	LogLevel          *string
	LogFile           *string
	LogLength         *int
	LogFollow         *bool
	LogTailLength     *int
	IsRawLogOutput    *bool
	IsTuiEnabled      *bool
	Command           *string
	Write             *bool
	NoDependencies    *bool
	HideDisabled      *bool
	DisabledProcesses *[]string
	EnabledProcesses  *[]string
	SortColumn        *string
	SortColumnChanged bool
	IsReverseSort     *bool
	NoServer          *bool
	KeepTuiOn         *bool
	KeepProjectOn     *bool
	IsOrderedShutDown *bool
	PcTheme           *string
	PcThemeChanged    bool
	ShortcutPaths     *[]string
	UnixSocketPath    *string
	IsUnixSocket      *bool
	IsReadOnlyMode    *bool
	OutputFormat      *string
	DisableDotEnv     *bool
	IsTuiFullScreen   *bool
	IsDetached        *bool
	IsDetachedWithTui *bool
	DetachOnSuccess   *bool
}

// NewFlags returns new configuration flags.
func NewFlags() *Flags {
	return &Flags{
		RefreshRate:       toPtr(DefaultRefreshRate),
		IsTuiEnabled:      toPtr(getDisableTuiDefault()),
		PortNum:           toPtr(getPortDefault()),
		Address:           toPtr(DefaultAddress),
		LogLength:         toPtr(DefaultLogLength),
		LogLevel:          toPtr(DefaultLogLevel),
		LogFile:           toPtr(GetLogFilePath()),
		LogFollow:         toPtr(false),
		LogTailLength:     toPtr(math.MaxInt),
		NoDependencies:    toPtr(false),
		HideDisabled:      toPtr(getHideDisabledDefault()),
		DisabledProcesses: toPtr(getDisabledProcessesDefault()),
		EnabledProcesses:  toPtr(getEnabledProcessesDefault()),
		SortColumn:        toPtr(DefaultSortColumn),
		IsReverseSort:     toPtr(false),
		NoServer:          toPtr(getNoServerDefault()),
		KeepTuiOn:         toPtr(false),
		KeepProjectOn:     toPtr(false),
		IsOrderedShutDown: toPtr(false),
		PcTheme:           toPtr(DefaultThemeName),
		ShortcutPaths:     toPtr(GetShortCutsPaths(nil)),
		UnixSocketPath:    toPtr(""),
		IsUnixSocket:      toPtr(false),
		IsReadOnlyMode:    toPtr(getReadOnlyDefault()),
		OutputFormat:      toPtr(""),
		DisableDotEnv:     toPtr(getDisableDotEnvDefault()),
		IsTuiFullScreen:   toPtr(getTuiFullScreenDefault()),
		IsDetached:        toPtr(false),
		IsDetachedWithTui: toPtr(false),
		IsRawLogOutput:    toPtr(false),
		DetachOnSuccess:   toPtr(false),
	}
}

func toPtr[T any](t T) *T {
	return &t
}
