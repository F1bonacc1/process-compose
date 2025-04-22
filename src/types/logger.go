package types

// LogRotationConfig is the configuration for logging
type LogRotationConfig struct {
	// Directory to log to when filelogging is enabled
	Directory string `yaml:"directory,omitempty"`
	// Filename is the name of the logfile which will be placed inside the directory
	Filename string `yaml:"filename,omitempty"`
	// MaxSize the max size in MB of the logfile before it's rolled
	MaxSize int `yaml:"max_size_mb,omitempty"`
	// MaxBackups the max number of rolled files to keep
	MaxBackups int `yaml:"max_backups,omitempty"`
	// MaxAge the max age in days to keep a logfile
	MaxAge int `yaml:"max_age_days,omitempty"`
	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool `json:"compress" yaml:"compress,omitempty"`
}

type LoggerConfig struct {
	// Rotation is the configuration for logging rotation
	Rotation *LogRotationConfig `yaml:"rotation,omitempty"`
	// FieldsOrder is the order in which fields are logged
	FieldsOrder []string `yaml:"fields_order,omitempty"`
	// DisableJSON disables log JSON formatting
	DisableJSON bool `yaml:"disable_json,omitempty"`
	// TimestampFormat is the format of the timestamp
	TimestampFormat string `yaml:"timestamp_format,omitempty"`
	// NoColor disables coloring
	NoColor bool `yaml:"no_color,omitempty"`
	// NoMetadata disables log metadata (process, replica)
	NoMetadata bool `yaml:"no_metadata,omitempty"`
	// AddTimestamp adds timestamp to log
	AddTimestamp bool `yaml:"add_timestamp,omitempty"`
	// FlushEachLine flushes the logger on each line
	FlushEachLine bool `yaml:"flush_each_line,omitempty"`
}
