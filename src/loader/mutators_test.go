package loader

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/types"
	"testing"
)

func Test_assignDefaultProcessValues(t *testing.T) {
	type args struct {
		p *types.Project
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{
				p: &types.Project{
					Processes: types.Processes{
						"test2": {
							Name: "test2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assignDefaultProcessValues(tt.args.p)
			for _, p := range tt.args.p.Processes {
				if p.Namespace == "" {
					t.Errorf("Expected namespace to be set")
				}
				if p.Replicas == 0 {
					t.Errorf("Expected replicas to be set")
				}
				if p.Name == "" {
					t.Errorf("Expected name to be set")
				}
			}
		})
	}
}

func Test_setDefaultShell(t *testing.T) {
	type args struct {
		p *types.Project
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "valid",
			args: args{
				p: &types.Project{
					ShellConfig: &command.ShellConfig{
						ShellCommand:  "bash",
						ShellArgument: "-c",
					},
				},
			},
		},
		{
			name: "use default bash",
			args: args{
				p: &types.Project{
					ShellConfig: nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setDefaultShell(tt.args.p)
			if tt.args.p.ShellConfig.ShellCommand != "bash" {
				t.Errorf("Expected shell command to be bash")
			}
			if tt.args.p.ShellConfig.ShellArgument != "-c" {
				t.Errorf("Expected shell argument to be '-c'")
			}
		})
	}
}

func Test_copyWorkingDirToProbes(t *testing.T) {

	procNoWorkingDir := "noWorkingDir"
	procWithWorkingDir := "withWorkingDir"

	type args struct {
		p *types.Project
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{
				p: &types.Project{
					Processes: types.Processes{
						procNoWorkingDir: {
							Name:       procNoWorkingDir,
							WorkingDir: "/tmp",
							LivenessProbe: &health.Probe{
								Exec: &health.ExecProbe{
									Command: "echo",
								},
							},
							ReadinessProbe: &health.Probe{
								Exec: &health.ExecProbe{
									Command: "echo",
								},
							},
						},
						procWithWorkingDir: {
							Name:       procWithWorkingDir,
							WorkingDir: "/tmp",
							LivenessProbe: &health.Probe{
								Exec: &health.ExecProbe{
									WorkingDir: "/another",
								},
							},
							ReadinessProbe: &health.Probe{
								Exec: &health.ExecProbe{
									WorkingDir: "/another",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copyWorkingDirToProbes(tt.args.p)
			for _, p := range tt.args.p.Processes {
				switch p.Name {
				case procWithWorkingDir:
					if p.LivenessProbe.Exec.WorkingDir != "/another" {
						t.Errorf("Expected liveness probe working dir to be another")
					}
					if p.ReadinessProbe.Exec.WorkingDir != "/another" {
						t.Errorf("Expected readiness probe working dir to be another")
					}
				case procNoWorkingDir:
					if p.LivenessProbe.Exec.WorkingDir != "/tmp" {
						t.Errorf("Expected lieveness probe working dir to be tmp")
					}
					if p.ReadinessProbe.Exec.WorkingDir != "/tmp" {
						t.Errorf("Expected readiness probe working dir to be tmp")
					}
				default:
					t.Errorf("Expected process to exist")
				}
			}
		})
	}
}

func Test_cloneReplicas(t *testing.T) {
	type replica struct {
		name        string
		num         int
		replicaName string
	}

	type procParams struct {
		Name     string
		Replicas int
	}

	procs := []procParams{
		{
			Name:     "p0",
			Replicas: 0,
		},
		{
			Name:     "p1",
			Replicas: 1,
		},
		{
			Name:     "p2",
			Replicas: 2,
		},
		{
			Name:     "p10",
			Replicas: 10,
		},
	}

	for _, p := range procs {
		p1 := &types.Project{
			Processes: types.Processes{
				p.Name: {
					Name:     p.Name,
					Replicas: p.Replicas,
				},
			},
		}
		assignDefaultProcessValues(p1)
		cloneReplicas(p1)
		replicas := []replica{}
		switch p.Replicas {
		case 0:
			replicas = append(replicas, replica{
				name:        p.Name,
				num:         0,
				replicaName: "p0",
			})
		case 1:
			replicas = append(replicas, replica{
				name:        p.Name,
				num:         0,
				replicaName: p.Name,
			})
		case 2:
			replicas = append(replicas, replica{
				name:        fmt.Sprintf("%s-0", p.Name),
				num:         0,
				replicaName: fmt.Sprintf("%s-0", p.Name),
			},
				replica{
					name:        fmt.Sprintf("%s-1", p.Name),
					num:         1,
					replicaName: fmt.Sprintf("%s-1", p.Name),
				},
			)
		case 10:
			replicas = append(replicas, replica{
				name:        fmt.Sprintf("%s-00", p.Name),
				num:         0,
				replicaName: fmt.Sprintf("%s-00", p.Name),
			},
				replica{
					name:        fmt.Sprintf("%s-01", p.Name),
					num:         1,
					replicaName: fmt.Sprintf("%s-01", p.Name),
				},
				replica{
					name:        fmt.Sprintf("%s-09", p.Name),
					num:         9,
					replicaName: fmt.Sprintf("%s-09", p.Name),
				},
			)

		}
		for _, r := range replicas {
			proc, ok := p1.Processes[r.name]
			if !ok {
				t.Errorf("Expected process %s to exist", r.name)
			}
			if proc.ReplicaNum != r.num {
				t.Errorf("Expected replica num to be %d, got %d", r.num, proc.ReplicaNum)
			}
			if proc.ReplicaName != r.replicaName {
				t.Errorf("Expected replica name to be %s, got %s", r.replicaName, proc.ReplicaName)
			}
		}
	}
}
