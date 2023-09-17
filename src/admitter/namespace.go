package admitter

import "github.com/f1bonacc1/process-compose/src/types"

type NamespaceAdmitter struct {
	EnabledNamespaces []string
}

func (n *NamespaceAdmitter) Admit(proc *types.ProcessConfig) bool {
	if len(n.EnabledNamespaces) == 0 {
		return true
	}
	for _, ns := range n.EnabledNamespaces {
		if ns == proc.Namespace {
			return true
		}
	}
	return false
}
