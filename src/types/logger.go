package types

// LogRotationConfig is the configuration for logging
type LogRotationConfig struct {
	// Directory to log to when filelogging is enabled
	Directory string `yaml:"directory,omitempty" json:"directory,omitempty"`
	// Filename is the name of the logfile which will be placed inside the directory
	Filename string `yaml:"filename,omitempty" json:"filename,omitempty"`
	// MaxSize the max size in MB of the logfile before it's rolled
	MaxSize int `yaml:"max_size_mb,omitempty" json:"maxSize,omitempty"`
	// MaxBackups the max number of rolled files to keep
	MaxBackups int `yaml:"max_backups,omitempty" json:"maxBackups,omitempty"`
	// MaxAge the max age in days to keep a logfile
	MaxAge int `yaml:"max_age_days,omitempty" json:"maxAge,omitempty"`
	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool `yaml:"compress,omitempty" json:"compress,omitempty"`
}

type LoggerConfig struct {
	// Rotation is the configuration for logging rotation
	Rotation *LogRotationConfig `yaml:"rotation,omitempty" json:"rotation,omitempty"`
	// FieldsOrder is the order in which fields are logged
	FieldsOrder []string `yaml:"fields_order,omitempty" json:"fieldsOrder,omitempty"`
	// DisableJSON disables log JSON formatting
	DisableJSON bool `yaml:"disable_json,omitempty" json:"disableJSON,omitempty"`
	// TimestampFormat is the format of the timestamp
	TimestampFormat string `yaml:"timestamp_format,omitempty" json:"timestampFormat,omitempty"`
	// NoColor disables coloring
	NoColor bool `yaml:"no_color,omitempty" json:"noColor,omitempty"`
	// NoMetadata disables log metadata (process, replica)
	NoMetadata bool `yaml:"no_metadata,omitempty" json:"noMetadata,omitempty"`
	// AddTimestamp adds timestamp to log
	AddTimestamp bool `yaml:"add_timestamp,omitempty" json:"addTimestamp,omitempty"`
	// FlushEachLine flushes the logger on each line
	FlushEachLine bool `yaml:"flush_each_line,omitempty" json:"flushEachLine,omitempty"`
}
