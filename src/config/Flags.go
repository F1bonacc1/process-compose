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
)

const (
	EnvVarNamePort     = "PC_PORT_NUM"
	EnvVarNameTui      = "PC_DISABLE_TUI"
	EnvVarNameConfig   = "PC_CONFIG_FILES"
	EnvVarNameNoServer = "PC_NO_SERVER"
)

// Flags represents PC configuration flags.
type Flags struct {
	RefreshRate    *int
	PortNum        *int
	Address        *string
	LogLevel       *string
	LogFile        *string
	LogLength      *int
	LogFollow      *bool
	LogTailLength  *int
	Headless       *bool
	Command        *string
	Write          *bool
	NoDependencies *bool
	HideDisabled   *bool
	SortColumn     *string
	IsReverseSort  *bool
	NoServer       *bool
}

// NewFlags returns new configuration flags.
func NewFlags() *Flags {
	return &Flags{
		RefreshRate:    intPtr(DefaultRefreshRate),
		Headless:       boolPtr(getTuiDefault()),
		PortNum:        intPtr(getPortDefault()),
		Address:        strPtr(DefaultAddress),
		LogLength:      intPtr(DefaultLogLength),
		LogLevel:       strPtr(DefaultLogLevel),
		LogFile:        strPtr(GetLogFilePath()),
		LogFollow:      boolPtr(false),
		LogTailLength:  intPtr(math.MaxInt),
		NoDependencies: boolPtr(false),
		HideDisabled:   boolPtr(false),
		SortColumn:     strPtr(DefaultSortColumn),
		IsReverseSort:  boolPtr(false),
		NoServer:       boolPtr(getNoServerDefault()),
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
