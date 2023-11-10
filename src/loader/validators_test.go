package loader

import (
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/types"
	"testing"
)

func Test_validateProcessConfig(t *testing.T) {
	type args struct {
		p *types.Project
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid",
			args: args{
				p: &types.Project{
					Processes: types.Processes{
						"test": {
							Name:    "test",
							Command: "echo",
						},
						"test2": {
							Name:    "test2",
							Command: "echo",
						},
					},
					IsStrict: true,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid non strict",
			args: args{
				p: &types.Project{
					Processes: types.Processes{
						"test2": {
							Name:    "test2",
							Command: "echo",
							Extensions: map[string]interface{}{
								"invalid": "invalid",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid strict",
			args: args{
				p: &types.Project{
					Processes: types.Processes{
						"test2": {
							Name:    "test2",
							Command: "echo",
							Extensions: map[string]interface{}{
								"invalid": "invalid",
							},
						},
					},
					IsStrict: true,
				},
			},
			wantErr: true,
		},
		{
			name: "Valid extension strict",
			args: args{
				p: &types.Project{
					Processes: types.Processes{

						"test2": {
							Name:    "test2",
							Command: "echo",
							Extensions: map[string]interface{}{
								"x-valid": "valid",
							},
						},
					},
					IsStrict: true,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateProcessConfig(tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("validateProcessConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateLogLevel(t *testing.T) {
	type args struct {
		p *types.Project
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid",
			args: args{
				p: &types.Project{
					LogLevel: "debug",
					IsStrict: true,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid non strict",
			args: args{
				p: &types.Project{
					LogLevel: "invalid",
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid strict",
			args: args{
				p: &types.Project{
					LogLevel: "invalid",
					IsStrict: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateLogLevel(tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("validateLogLevel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateShellConfig(t *testing.T) {
	type args struct {
		p *types.Project
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid",
			args: args{
				p: &types.Project{
					ShellConfig: &command.ShellConfig{
						ShellCommand: "sh",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid",
			args: args{
				p: &types.Project{
					ShellConfig: &command.ShellConfig{
						ShellCommand: "wrong",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateShellConfig(tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("validateShellConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
