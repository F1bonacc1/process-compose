package admitter

import "github.com/f1bonacc1/process-compose/src/types"

type NamespaceAdmitter struct {
	EnabledNamespaces []string
}

func (n *NamespaceAdmitter) Admit(proc *types.ProcessConfig) bool {
	if len(n.EnabledNamespaces) == 0 {
		return true
	}
	return proc.Namespace.HasAny(n.EnabledNamespaces)
}
