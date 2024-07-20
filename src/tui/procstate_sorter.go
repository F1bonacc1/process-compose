package tui

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"sort"
	"strings"
)

type ColumnID int

const (
	ProcessStateUndefined ColumnID = -1
	ProcessStateIcon      ColumnID = 0
	ProcessStatePid       ColumnID = 1
	ProcessStateName      ColumnID = 2
	ProcessStateNamespace ColumnID = 3
	ProcessStateStatus    ColumnID = 4
	ProcessStateAge       ColumnID = 5
	ProcessStateHealth    ColumnID = 6
	ProcessStateMem       ColumnID = 7
	ProcessStateRestarts  ColumnID = 8
	ProcessStateExit      ColumnID = 9
)

var columnNames = map[ColumnID]string{
	ProcessStateUndefined: "",
	ProcessStatePid:       "PID",
	ProcessStateName:      "NAME",
	ProcessStateNamespace: "NAMESPACE",
	ProcessStateStatus:    "STATUS",
	ProcessStateAge:       "AGE",
	ProcessStateHealth:    "HEALTH",
	ProcessStateMem:       "MEM",
	ProcessStateRestarts:  "RESTARTS",
	ProcessStateExit:      "EXIT",
}

var columnIDs = map[string]ColumnID{
	"":          ProcessStateUndefined,
	"PID":       ProcessStatePid,
	"NAME":      ProcessStateName,
	"NAMESPACE": ProcessStateNamespace,
	"STATUS":    ProcessStateStatus,
	"AGE":       ProcessStateAge,
	"HEALTH":    ProcessStateHealth,
	"MEM":       ProcessStateMem,
	"RESTARTS":  ProcessStateRestarts,
	"EXIT":      ProcessStateExit,
}

func (c ColumnID) String() string {
	return columnNames[c]
}

func StringToColumnID(s string) (ColumnID, error) {
	id, ok := columnIDs[strings.ToUpper(s)]
	if !ok {
		return ProcessStateUndefined, fmt.Errorf("unknown column name: %s", s)
	}
	return id, nil
}
func ColumnNames() []string {
	var names []string
	for _, name := range columnNames {
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

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
			eci := getStrForExitCode(states.States[i])
			ecj := getStrForExitCode(states.States[j])
			if eci == ecj {
				return states.States[i].Name < states.States[j].Name
			} else {
				return eci < ecj
			}
		}
	case ProcessStateMem:
		return func(i, j int) bool {
			return states.States[i].Mem < states.States[j].Mem
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
