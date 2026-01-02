package types

import (
	"time"
)

// ScheduleConfig defines the scheduling configuration for a process.
type ScheduleConfig struct {
	// Cron expression for cron-based scheduling (e.g., "0 2 * * *")
	Cron string `yaml:"cron,omitempty" json:"cron,omitempty"`

	// Timezone for cron expression (e.g., "UTC", "America/New_York")
	Timezone string `yaml:"timezone,omitempty" json:"timezone,omitempty"`

	// Interval for interval-based scheduling (e.g., "30m", "1h", "5s")
	Interval string `yaml:"interval,omitempty" json:"interval,omitempty"`

	// RunOnStart determines whether to run immediately when process-compose starts
	RunOnStart bool `yaml:"run_on_start,omitempty" json:"run_on_start,omitempty"`

	// MaxConcurrent limits concurrent executions (default: 1)
	MaxConcurrent int `yaml:"max_concurrent,omitempty" json:"max_concurrent,omitempty"`
}

// IsScheduled returns true if this config has any scheduling defined.
func (s *ScheduleConfig) IsScheduled() bool {
	return s != nil && (s.Cron != "" || s.Interval != "")
}

// GetMaxConcurrent returns the max concurrent value, defaulting to 1.
func (s *ScheduleConfig) GetMaxConcurrent() int {
	if s == nil || s.MaxConcurrent <= 0 {
		return 1
	}
	return s.MaxConcurrent
}

// GetTimezone returns the configured timezone location.
func (s *ScheduleConfig) GetTimezone() (*time.Location, error) {
	if s == nil || s.Timezone == "" {
		return time.Local, nil //nolint:gosmopolitan
	}
	return time.LoadLocation(s.Timezone)
}

// GetIntervalDuration parses and returns the interval as a duration.
func (s *ScheduleConfig) GetIntervalDuration() (time.Duration, error) {
	if s == nil || s.Interval == "" {
		return 0, nil
	}
	return time.ParseDuration(s.Interval)
}
