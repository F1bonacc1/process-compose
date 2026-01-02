package loader

import (
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
)

func Test_validateScheduledProcessScaling(t *testing.T) {
	type args struct {
		p *types.Project
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid scheduled process",
			args: args{
				p: &types.Project{
					Processes: types.Processes{
						"sched": {
							Name:     "sched",
							Replicas: 1,
							Schedule: &types.ScheduleConfig{
								Cron: "* * * * *",
							},
						},
					},
					IsStrict: true,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid scaled scheduled process (non-strict)",
			args: args{
				p: &types.Project{
					Processes: types.Processes{
						"sched": {
							Name:     "sched",
							Replicas: 2,
							Schedule: &types.ScheduleConfig{
								Cron: "* * * * *",
							},
						},
					},
					IsStrict: false,
				},
			},
			wantErr: false, // Should warn but not error
		},
		{
			name: "Invalid scaled scheduled process (strict)",
			args: args{
				p: &types.Project{
					Processes: types.Processes{
						"sched": {
							Name:     "sched",
							Replicas: 2,
							Schedule: &types.ScheduleConfig{
								Cron: "* * * * *",
							},
						},
					},
					IsStrict: true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateScheduledProcessScaling(tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("validateScheduledProcessScaling() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
