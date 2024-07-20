package tui

import (
	"testing"
)

func TestMemToString(t *testing.T) {
	type args struct {
		mem int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "negative",
			args: args{
				mem: -1,
			},
			want: "Unknown",
		},
		{
			name: "zero",
			args: args{
				mem: 0,
			},
			want: "-",
		},
		{
			name: "bytes",
			args: args{
				mem: 345,
			},
			want: "0.0 MiB",
		},
		{
			name: "kilobytes",
			args: args{
				mem: 1024 * 513,
			},
			want: "0.5 MiB",
		},
		{
			name: "megabytes",
			args: args{
				mem: 1024*1024*7 + 1024*513,
			},
			want: "7.5 MiB",
		},
		{
			name: "gigabytes",
			args: args{
				mem: 1024*1024*1024*4 + 1024*1024*740,
			},
			want: "4.7 GiB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStrForMem(tt.args.mem); got != tt.want {
				t.Errorf("DurationToString() = %v, want %v", got, tt.want)
			}
		})
	}
}
