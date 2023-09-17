package admitter

import "github.com/f1bonacc1/process-compose/src/types"

type DisabledProcAdmitter struct {
}

func (d *DisabledProcAdmitter) Admit(proc *types.ProcessConfig) bool {
	return !proc.Disabled
}
