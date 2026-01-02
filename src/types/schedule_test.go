package types

import (
	"testing"
	"time"
)

func TestScheduleConfig_IsScheduled(t *testing.T) {
	tests := []struct {
		name   string
		config *ScheduleConfig
		want   bool
	}{
		{
			name:   "nil config",
			config: nil,
			want:   false,
		},
		{
			name:   "empty config",
			config: &ScheduleConfig{},
			want:   false,
		},
		{
			name: "cron only",
			config: &ScheduleConfig{
				Cron: "0 2 * * *",
			},
			want: true,
		},
		{
			name: "interval only",
			config: &ScheduleConfig{
				Interval: "30m",
			},
			want: true,
		},
		{
			name: "both cron and interval",
			config: &ScheduleConfig{
				Cron:     "0 2 * * *",
				Interval: "30m",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsScheduled(); got != tt.want {
				t.Errorf("IsScheduled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScheduleConfig_GetMaxConcurrent(t *testing.T) {
	tests := []struct {
		name   string
		config *ScheduleConfig
		want   int
	}{
		{
			name:   "nil config defaults to 1",
			config: nil,
			want:   1,
		},
		{
			name:   "zero defaults to 1",
			config: &ScheduleConfig{MaxConcurrent: 0},
			want:   1,
		},
		{
			name:   "negative defaults to 1",
			config: &ScheduleConfig{MaxConcurrent: -5},
			want:   1,
		},
		{
			name:   "positive value returned",
			config: &ScheduleConfig{MaxConcurrent: 3},
			want:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetMaxConcurrent(); got != tt.want {
				t.Errorf("GetMaxConcurrent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScheduleConfig_GetTimezone(t *testing.T) {
	tests := []struct {
		name      string
		config    *ScheduleConfig
		wantName  string
		wantError bool
	}{
		{
			name:      "nil config uses local",
			config:    nil,
			wantName:  time.Local.String(),
			wantError: false,
		},
		{
			name:      "empty timezone uses local",
			config:    &ScheduleConfig{Timezone: ""},
			wantName:  time.Local.String(),
			wantError: false,
		},
		{
			name:      "UTC timezone",
			config:    &ScheduleConfig{Timezone: "UTC"},
			wantName:  "UTC",
			wantError: false,
		},
		{
			name:      "America/New_York timezone",
			config:    &ScheduleConfig{Timezone: "America/New_York"},
			wantName:  "America/New_York",
			wantError: false,
		},
		{
			name:      "invalid timezone returns error",
			config:    &ScheduleConfig{Timezone: "Invalid/Timezone"},
			wantName:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := tt.config.GetTimezone()
			if (err != nil) != tt.wantError {
				t.Errorf("GetTimezone() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && loc.String() != tt.wantName {
				t.Errorf("GetTimezone() = %v, want %v", loc.String(), tt.wantName)
			}
		})
	}
}

func TestScheduleConfig_GetIntervalDuration(t *testing.T) {
	tests := []struct {
		name      string
		config    *ScheduleConfig
		want      time.Duration
		wantError bool
	}{
		{
			name:      "nil config returns zero",
			config:    nil,
			want:      0,
			wantError: false,
		},
		{
			name:      "empty interval returns zero",
			config:    &ScheduleConfig{Interval: ""},
			want:      0,
			wantError: false,
		},
		{
			name:      "30 minutes",
			config:    &ScheduleConfig{Interval: "30m"},
			want:      30 * time.Minute,
			wantError: false,
		},
		{
			name:      "1 hour",
			config:    &ScheduleConfig{Interval: "1h"},
			want:      time.Hour,
			wantError: false,
		},
		{
			name:      "5 seconds",
			config:    &ScheduleConfig{Interval: "5s"},
			want:      5 * time.Second,
			wantError: false,
		},
		{
			name:      "complex duration",
			config:    &ScheduleConfig{Interval: "1h30m"},
			want:      90 * time.Minute,
			wantError: false,
		},
		{
			name:      "invalid duration returns error",
			config:    &ScheduleConfig{Interval: "invalid"},
			want:      0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.GetIntervalDuration()
			if (err != nil) != tt.wantError {
				t.Errorf("GetIntervalDuration() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("GetIntervalDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
