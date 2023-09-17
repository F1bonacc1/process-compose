package admitter

import "github.com/f1bonacc1/process-compose/src/types"

type Admitter interface {
	Admit(config *types.ProcessConfig) bool
}
