package loader

import (
	"github.com/f1bonacc1/process-compose/src/types"
	"testing"
)

func TestLoaderOptions_getWorkingDir(t *testing.T) {
	type fields struct {
		workingDir string
		FileNames  []string
		projects   []*types.Project
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "Empty Working Dir",
			fields: fields{
				workingDir: "",
				FileNames: []string{
					"/home/user/dir/process-compose.yaml",
				},
				projects: nil,
			},
			want:    "/home/user/dir",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := LoaderOptions{
				workingDir: tt.fields.workingDir,
				FileNames:  tt.fields.FileNames,
				projects:   tt.fields.projects,
			}
			got, err := o.getWorkingDir()
			if (err != nil) != tt.wantErr {
				t.Errorf("getWorkingDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getWorkingDir() got = %v, want %v", got, tt.want)
			}
		})
	}
}
