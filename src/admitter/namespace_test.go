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
					Namespace: "",
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
					Namespace: "",
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
					Namespace: "not-test",
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
					Namespace: "test",
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
					Namespace: "test",
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
