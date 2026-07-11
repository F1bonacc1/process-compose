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

	// NoNamespace represents no namespace selection
	NoNamespace = ""
)

const (
	EnvVarNamePort             = "PC_PORT_NUM"
	EnvVarNameTui              = "PC_DISABLE_TUI"
	EnvVarNameConfig           = "PC_CONFIG_FILES"
	EnvVarNameNamespace        = "PC_NAMESPACES"
	EnvVarStrictNamespace      = "PC_STRICT_NAMESPACE"
	EnvVarNameShortcuts        = "PC_SHORTCUTS_FILES"
	EnvVarNameRecipes          = "PC_RECIPE_FILES"
	EnvVarNameNoServer         = "PC_NO_SERVER"
	EnvVarUnixSocketPath       = "PC_SOCKET_PATH"
	EnvVarReadOnlyMode         = "PC_READ_ONLY"
	EnvVarDisableDotEnv        = "PC_DISABLE_DOTENV"
	EnvVarTuiFullScreen        = "PC_TUI_FULL_SCREEN"
	EnvVarHideDisabled         = "PC_HIDE_DISABLED_PROC"
	EnvVarNameOrderedShutdown  = "PC_ORDERED_SHUTDOWN"
	EnvVarWithRecursiveMetrics = "PC_RECURSIVE_METRICS"
	EnvVarDisabledProcs        = "PC_DISABLED_PROCESSES"
	EnvVarNameAddress          = "PC_ADDRESS"
	EnvVarLogNoColor           = "PC_LOG_NO_COLOR"
)

// Flags represents PC configuration flags.
type Flags struct {
	RefreshRate          *time.Duration
	SlowRefreshRate      *time.Duration
	PortNum              *int
	Address              *string
	LogLevel             *string
	LogFile              *string
	LogLength            *int
	LogFollow            *bool
	LogTailLength        *int
	IsRawLogOutput       *bool
	IsTuiEnabled         *bool
	Command              *string
	Write                *bool
	NoDependencies       *bool
	HideDisabled         *bool
	SortColumn           *string
	SortColumnChanged    bool
	IsReverseSort        *bool
	NoServer             *bool
	KeepTuiOn            *bool
	KeepProjectOn        *bool
	IsOrderedShutdown    *bool
	PcTheme              *string
	PcThemeChanged       bool
	ShortcutPaths        *[]string
	UnixSocketPath       *string
	IsUnixSocket         *bool
	IsReadOnlyMode       *bool
	OutputFormat         *string
	DisableDotEnv        *bool
	IsTuiFullScreen      *bool
	IsDetached           *bool
	IsDetachedWithTui    *bool
	Namespace            *string
	DetachOnSuccess      *bool
	WaitReady            *bool
	ShortVersion         *bool
	LogsTruncate         *bool
	WithRecursiveMetrics *bool
	ApiTokenPath         *string
	LogNoColor           *bool
}

// NewFlags returns new configuration flags.
func NewFlags() *Flags {
	return &Flags{
		RefreshRate:          new(DefaultRefreshRate),
		SlowRefreshRate:      new(DefaultRefreshRate),
		IsTuiEnabled:         new(getDisableTuiDefault()),
		PortNum:              new(getPortDefault()),
		Address:              new(getAddressDefault()),
		LogLength:            new(DefaultLogLength),
		LogLevel:             new(DefaultLogLevel),
		LogFile:              new(GetLogFilePath()),
		LogFollow:            new(false),
		LogTailLength:        new(math.MaxInt),
		NoDependencies:       new(false),
		HideDisabled:         new(getHideDisabledDefault()),
		SortColumn:           new(DefaultSortColumn),
		IsReverseSort:        new(false),
		NoServer:             new(getNoServerDefault()),
		KeepTuiOn:            new(false),
		KeepProjectOn:        new(false),
		IsOrderedShutdown:    new(getOrderedShutdownDefault()),
		PcTheme:              new(DefaultThemeName),
		ShortcutPaths:        new(GetShortCutsPaths(nil)),
		UnixSocketPath:       new(""),
		IsUnixSocket:         new(false),
		IsReadOnlyMode:       new(getReadOnlyDefault()),
		OutputFormat:         new(""),
		DisableDotEnv:        new(getDisableDotEnvDefault()),
		IsTuiFullScreen:      new(getTuiFullScreenDefault()),
		IsDetached:           new(false),
		IsDetachedWithTui:    new(false),
		IsRawLogOutput:       new(false),
		Namespace:            new(NoNamespace),
		DetachOnSuccess:      new(false),
		WaitReady:            new(false),
		ShortVersion:         new(false),
		LogsTruncate:         new(false),
		WithRecursiveMetrics: new(getWithRecursiveMetricsEnvDefault()),
		ApiTokenPath:         new(getApiTokenPathDefault()),
		LogNoColor:           new(getLogNoColorDefault()),
	}
}
