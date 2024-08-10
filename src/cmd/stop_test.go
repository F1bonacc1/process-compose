package cmd

import "testing"

func Test_prepareVerboseOutput(t *testing.T) {
	type args struct {
		status    map[string]string
		processes []string
	}
	tests := []struct {
		name         string
		args         args
		wantOutput   string
		wantExitCode int
	}{
		{
			name: "success",
			args: args{
				status:    map[string]string{"a": "ok", "b": "ok"},
				processes: []string{"a", "b"},
			},
			wantOutput:   "✓ Successfully stopped a\n✓ Successfully stopped b\n",
			wantExitCode: 0,
		},
		{
			name: "fail",
			args: args{
				status:    map[string]string{"a": "fail", "b": "fail"},
				processes: []string{"a", "b"},
			},
			wantOutput:   "✘ Failed to stop a: fail\n✘ Failed to stop b: fail\n",
			wantExitCode: 1,
		},
		{
			name: "fail and success",
			args: args{
				status:    map[string]string{"a": "ok", "b": "fail"},
				processes: []string{"a", "b"},
			},
			wantOutput:   "✓ Successfully stopped a\n✘ Failed to stop b: fail\n",
			wantExitCode: 1,
		},
		{
			name: "empty status",
			args: args{
				status:    map[string]string{},
				processes: []string{"a", "b"},
			},
			wantOutput:   "✘ Unknown status for process a\n✘ Unknown status for process b\n",
			wantExitCode: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOutput, gotExitCode := prepareVerboseOutput(tt.args.status, tt.args.processes)
			if gotOutput != tt.wantOutput {
				t.Errorf("prepareVerboseOutput() gotOutput = %v, want %v", gotOutput, tt.wantOutput)
			}
			if gotExitCode != tt.wantExitCode {
				t.Errorf("prepareVerboseOutput() gotExitCode = %v, want %v", gotExitCode, tt.wantExitCode)
			}
		})
	}
}

func Test_prepareConciseOutput(t *testing.T) {
	type args struct {
		stopped   map[string]string
		processes []string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 int
	}{
		{
			name: "success",
			args: args{
				stopped:   map[string]string{"a": "ok", "b": "ok"},
				processes: []string{"a", "b"},
			},
			want:  "Successfully stopped: 'a', 'b'\n",
			want1: 0,
		},
		{
			name: "fail",
			args: args{
				stopped:   map[string]string{"a": "fail", "b": "fail"},
				processes: []string{"a", "b"},
			},
			want:  "Failed to stop: 'a', 'b'\n",
			want1: 1,
		},
		{
			name: "fail and success",
			args: args{
				stopped:   map[string]string{"a": "ok", "b": "fail"},
				processes: []string{"a", "b"},
			},
			want:  "Successfully stopped 'a' but encountered failures for: 'b'\n",
			want1: 1,
		},
		{
			name: "empty status",
			args: args{
				stopped:   map[string]string{},
				processes: []string{"a", "b"},
			},
			want:  "Failed to stop: 'a', 'b'\n",
			want1: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := prepareConciseOutput(tt.args.stopped, tt.args.processes)
			if got != tt.want {
				t.Errorf("prepareConciseOutput() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("prepareConciseOutput() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
