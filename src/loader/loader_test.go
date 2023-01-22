package loader

import (
	"testing"
)

func Test_autoDiscoverComposeFile(t *testing.T) {
	type args struct {
		opts *LoaderOptions
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Should not find",
			args: args{
				opts: &LoaderOptions{
					workingDir: "../../fixtures",
					FileNames:  nil,
					projects:   nil,
				},
			},
			wantErr: true,
		},
		{
			name: "Should find process-compose.yaml",
			args: args{
				opts: &LoaderOptions{
					workingDir: "../../",
					FileNames:  nil,
					projects:   nil,
				},
			},
			wantErr: false,
		},
		{
			name: "Should find process-compose.yaml no CWD",
			args: args{
				opts: &LoaderOptions{
					workingDir: "",
					FileNames:  []string{"../../process-compose.yaml"},
					projects:   nil,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := autoDiscoverComposeFile(tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("autoDiscoverComposeFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				filesNum := len(tt.args.opts.FileNames)
				if filesNum == 0 {
					t.Errorf("autoDiscoverComposeFile() filesNum = %v, want Files %v", filesNum, 1)
				}
			}
		})
	}
}
