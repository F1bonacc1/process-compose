package tui

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"sort"
)

type ColumnID int

const (
	ProcessStateUndefined ColumnID = -1
	ProcessStatePid       ColumnID = 0
	ProcessStateName      ColumnID = 1
	ProcessStateNamespace ColumnID = 2
	ProcessStateStatus    ColumnID = 3
	ProcessStateAge       ColumnID = 4
	ProcessStateHealth    ColumnID = 5
	ProcessStateRestarts  ColumnID = 6
	ProcessStateExit      ColumnID = 7
)

type StateSorter struct {
	sortByColumn ColumnID
	isAsc        bool
}

type sortFn func(i, j int) bool

func sortProcessesState(sortBy ColumnID, asc bool, states *types.ProcessesState) error {

	if states == nil {
		return fmt.Errorf("empty states")
	}
	sorter := getSorter(sortBy, states)
	if !asc {
		sorter = reverse(sorter)
	}
	sort.Slice(states.States, sorter)
	return nil
}

func getSorter(sortBy ColumnID, states *types.ProcessesState) sortFn {
	switch sortBy {
	case ProcessStatePid:
		return func(i, j int) bool {
			if states.States[i].Pid == states.States[j].Pid {
				return states.States[i].Name < states.States[j].Name
			} else {
				return states.States[i].Pid < states.States[j].Pid
			}
		}
	case ProcessStateNamespace:
		return func(i, j int) bool {
			if states.States[i].Namespace == states.States[j].Namespace {
				return states.States[i].Name < states.States[j].Name
			} else {
				return states.States[i].Namespace < states.States[j].Namespace
			}
		}
	case ProcessStateStatus:
		return func(i, j int) bool {
			if states.States[i].Status == states.States[j].Status {
				return states.States[i].Name < states.States[j].Name
			} else {
				return states.States[i].Status < states.States[j].Status
			}
		}
	case ProcessStateAge:
		return func(i, j int) bool {
			if states.States[i].Age == states.States[j].Age {
				return states.States[i].Name < states.States[j].Name
			} else {
				return states.States[i].Age < states.States[j].Age
			}
		}
	case ProcessStateHealth:
		return func(i, j int) bool {
			if states.States[i].Health == states.States[j].Health {
				return states.States[i].Name < states.States[j].Name
			} else {
				return states.States[i].Health < states.States[j].Health
			}
		}
	case ProcessStateRestarts:
		return func(i, j int) bool {
			if states.States[i].Restarts == states.States[j].Restarts {
				return states.States[i].Name < states.States[j].Name
			} else {
				return states.States[i].Restarts < states.States[j].Restarts
			}
		}
	case ProcessStateExit:
		return func(i, j int) bool {
			if states.States[i].ExitCode == states.States[j].ExitCode {
				return states.States[i].Name < states.States[j].Name
			} else {
				return states.States[i].ExitCode < states.States[j].ExitCode
			}
		}
	case ProcessStateName:
		fallthrough
	default:
		return func(i, j int) bool {
			return states.States[i].Name < states.States[j].Name
		}
	}
}

func reverse(less func(i, j int) bool) func(i, j int) bool {
	return func(i, j int) bool {
		return !less(i, j)
	}
}
