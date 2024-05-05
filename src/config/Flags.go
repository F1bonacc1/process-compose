package config

import (
	"math"
)

const (
	// DefaultRefreshRate represents the refresh interval.
	DefaultRefreshRate = 1 // secs

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
	EnvVarNamePort       = "PC_PORT_NUM"
	EnvVarNameTui        = "PC_DISABLE_TUI"
	EnvVarNameConfig     = "PC_CONFIG_FILES"
	EnvVarNameNoServer   = "PC_NO_SERVER"
	EnvVarUnixSocketPath = "PC_SOCKET_PATH"
	EnvVarReadOnlyMode   = "PC_READ_ONLY"
)

// Flags represents PC configuration flags.
type Flags struct {
	RefreshRate       *int
	PortNum           *int
	Address           *string
	LogLevel          *string
	LogFile           *string
	LogLength         *int
	LogFollow         *bool
	LogTailLength     *int
	Headless          *bool
	Command           *string
	Write             *bool
	NoDependencies    *bool
	HideDisabled      *bool
	SortColumn        *string
	SortColumnChanged bool
	IsReverseSort     *bool
	NoServer          *bool
	KeepTuiOn         *bool
	IsOrderedShutDown *bool
	PcTheme           *string
	PcThemeChanged    bool
	UnixSocketPath    *string
	IsUnixSocket      *bool
	IsReadOnlyMode    *bool
}

// NewFlags returns new configuration flags.
func NewFlags() *Flags {
	return &Flags{
		RefreshRate:       toPtr(DefaultRefreshRate),
		Headless:          toPtr(getTuiDefault()),
		PortNum:           toPtr(getPortDefault()),
		Address:           toPtr(DefaultAddress),
		LogLength:         toPtr(DefaultLogLength),
		LogLevel:          toPtr(DefaultLogLevel),
		LogFile:           toPtr(GetLogFilePath()),
		LogFollow:         toPtr(false),
		LogTailLength:     toPtr(math.MaxInt),
		NoDependencies:    toPtr(false),
		HideDisabled:      toPtr(false),
		SortColumn:        toPtr(DefaultSortColumn),
		IsReverseSort:     toPtr(false),
		NoServer:          toPtr(getNoServerDefault()),
		KeepTuiOn:         toPtr(false),
		IsOrderedShutDown: toPtr(false),
		PcTheme:           toPtr(DefaultThemeName),
		UnixSocketPath:    toPtr(""),
		IsUnixSocket:      toPtr(false),
		IsReadOnlyMode:    toPtr(getReadOnlyDefault()),
	}
}

func toPtr[T any](t T) *T {
	return &t
}
