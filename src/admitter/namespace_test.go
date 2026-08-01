package admitter

import (
	"github.com/f1bonacc1/process-compose/src/types"
	"testing"
)

func TestNamespaceAdmitter_Admit(t *testing.T) {
	type fields struct {
		EnabledNamespaces []string
	}
	type args struct {
		proc *types.ProcessConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "no namespace",
			fields: fields{
				EnabledNamespaces: []string{},
			},
			args: args{
				proc: &types.ProcessConfig{
					Namespace: nil,
				},
			},
			want: true,
		},
		{
			name: "nil namespace",
			fields: fields{
				EnabledNamespaces: nil,
			},
			args: args{
				proc: &types.ProcessConfig{
					Namespace: nil,
				},
			},
			want: true,
		},
		{
			name: "mismatched namespace",
			fields: fields{
				EnabledNamespaces: []string{"test"},
			},
			args: args{
				proc: &types.ProcessConfig{
					Namespace: types.Namespaces{"not-test"},
				},
			},
			want: false,
		},
		{
			name: "matched namespace",
			fields: fields{
				EnabledNamespaces: []string{"test"},
			},
			args: args{
				proc: &types.ProcessConfig{
					Namespace: types.Namespaces{"test"},
				},
			},
			want: true,
		},
		{
			name: "matched namespaces",
			fields: fields{
				EnabledNamespaces: []string{"not-test", "test"},
			},
			args: args{
				proc: &types.ProcessConfig{
					Namespace: types.Namespaces{"test"},
				},
			},
			want: true,
		},
		{
			name: "process in multiple namespaces - one matches",
			fields: fields{
				EnabledNamespaces: []string{"test"},
			},
			args: args{
				proc: &types.ProcessConfig{
					Namespace: types.Namespaces{"not-test", "test"},
				},
			},
			want: true,
		},
		{
			name: "process in multiple namespaces - none matches",
			fields: fields{
				EnabledNamespaces: []string{"test"},
			},
			args: args{
				proc: &types.ProcessConfig{
					Namespace: types.Namespaces{"not-test", "also-not-test"},
				},
			},
			want: false,
		},
		{
			name: "no namespace admitted by default",
			fields: fields{
				EnabledNamespaces: []string{types.DefaultNamespace},
			},
			args: args{
				proc: &types.ProcessConfig{
					Namespace: nil,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NamespaceAdmitter{
				EnabledNamespaces: tt.fields.EnabledNamespaces,
			}
			if got := n.Admit(tt.args.proc); got != tt.want {
				t.Errorf("Admit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyToProject(t *testing.T) {
	project := &types.Project{
		Processes: types.Processes{
			"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1"},
				DependsOn: types.DependsOnConfig{"p2": {}, "p3": {}}},
			"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"ns2"}},
			"p3": {Name: "p3", ReplicaName: "p3", Namespace: types.Namespaces{"ns1"}},
		},
	}
	ApplyToProject(project, []Admitter{&NamespaceAdmitter{EnabledNamespaces: []string{"ns1"}}})

	if _, ok := project.Processes["p2"]; ok {
		t.Error("excluded process should be removed from the project")
	}
	p1 := project.Processes["p1"]
	if _, ok := p1.DependsOn["p2"]; ok {
		t.Error("dependency on the excluded process should be pruned")
	}
	if _, ok := p1.DependsOn["p3"]; !ok {
		t.Error("dependency on an admitted process should be kept")
	}
}

func TestApplyToProject_NoAdmittersKeepsProjectIntact(t *testing.T) {
	project := &types.Project{
		Processes: types.Processes{
			"p1": {Name: "p1", ReplicaName: "p1", Namespace: types.Namespaces{"ns1"},
				DependsOn: types.DependsOnConfig{"p2": {}}},
			"p2": {Name: "p2", ReplicaName: "p2", Namespace: types.Namespaces{"ns2"}},
		},
	}
	ApplyToProject(project, nil)

	if len(project.Processes) != 2 {
		t.Errorf("without admitters no process should be removed, got %d", len(project.Processes))
	}
	if _, ok := project.Processes["p1"].DependsOn["p2"]; !ok {
		t.Error("without admitters no dependency should be pruned")
	}
}
