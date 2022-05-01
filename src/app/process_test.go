package app

import (
	"testing"
	"time"
)

func TestDurationToString(t *testing.T) {
	type args struct {
		dur time.Duration
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "milis",
			args: args{
				dur: 20 * time.Millisecond,
			},
			want: "0s",
		},
		{
			name: "under 1m",
			args: args{
				dur: 20 * time.Second,
			},
			want: "20s",
		},
		{
			name: "under 3m",
			args: args{
				dur: 150 * time.Second,
			},
			want: "2m30s",
		},
		{
			name: "under 1h",
			args: args{
				dur: 30 * time.Minute,
			},
			want: "30m",
		},
		{
			name: "under 24h",
			args: args{
				dur: 280 * time.Minute,
			},
			want: "4h40m",
		},
		{
			name: "above 24h",
			args: args{
				dur: 25*time.Hour + 50*time.Minute,
			},
			want: "25h",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := durationToString(tt.args.dur); got != tt.want {
				t.Errorf("DurationToString() = %v, want %v", got, tt.want)
			}
		})
	}
}
