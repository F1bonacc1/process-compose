package types

// LogRotationConfig is the configuration for logging
type LogRotationConfig struct {
	// Directory to log to when filelogging is enabled
	Directory string `yaml:"directory"`
	// Filename is the name of the logfile which will be placed inside the directory
	Filename string `yaml:"filename"`
	// MaxSize the max size in MB of the logfile before it's rolled
	MaxSize int `yaml:"max_size_mb"`
	// MaxBackups the max number of rolled files to keep
	MaxBackups int `yaml:"max_backups"`
	// MaxAge the max age in days to keep a logfile
	MaxAge int `yaml:"max_age_days"`
	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool `json:"compress" yaml:"compress"`
}
