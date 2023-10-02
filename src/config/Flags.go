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
	PortEnvVarName   = "PC_PORT_NUM"
	TuiEnvVarName    = "PC_DISABLE_TUI"
	ConfigEnvVarName = "PC_CONFIG_FILES"
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
}

// NewFlags returns new configuration flags.
func NewFlags() *Flags {
	return &Flags{
		RefreshRate:    intPtr(DefaultRefreshRate),
		Headless:       boolPtr(GetTuiDefault()),
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
