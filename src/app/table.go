package app

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
)

// Print a list of process states in a table.
func PrintStatesAsTable(states []types.ProcessState) {

	// Create a table
	table := []string{"PID", "NAME", "NAMESPACE", "STATUS", "AGE", "HEALTH", "RESTARTS", "EXITCODE"}
	tableColWidth := make([]int, len(table))

	for _, state := range states {
		if len(fmt.Sprintf("%d", state.Pid)) > tableColWidth[0] {
			tableColWidth[0] = len(fmt.Sprintf("%d", state.Pid))
		}
		if len(state.Name) > tableColWidth[1] {
			tableColWidth[1] = len(state.Name)
		}
		if len(state.Namespace) > tableColWidth[2] {
			tableColWidth[2] = len(state.Namespace)
		}
		if len(state.Status) > tableColWidth[3] {
			tableColWidth[3] = len(state.Status)
		}
		if len(state.SystemTime) > tableColWidth[4] {
			tableColWidth[4] = len(state.SystemTime)
		}
		if len(state.Health) > tableColWidth[5] {
			tableColWidth[5] = len(state.Health)
		}
		if len(fmt.Sprintf("%d", state.Restarts)) > tableColWidth[6] {
			tableColWidth[6] = len(fmt.Sprintf("%d", state.Restarts))
		}
		if len(fmt.Sprintf("%d", state.ExitCode)) > tableColWidth[7] {
			tableColWidth[7] = len(fmt.Sprintf("%d", state.ExitCode))
		}
	}
	for i, col := range table {
		if len(col) > tableColWidth[i] {
			tableColWidth[i] = len(col)
		}
	}
	for i, col := range table {
		fmt.Printf("%-*s   ", tableColWidth[i], col)
	}
	fmt.Println()
	for _, state := range states {
		fmt.Printf("%-*d   ", tableColWidth[0], state.Pid)
		fmt.Printf("%-*s   ", tableColWidth[1], state.Name)
		fmt.Printf("%-*s   ", tableColWidth[2], state.Namespace)
		fmt.Printf("%-*s   ", tableColWidth[3], state.Status)
		fmt.Printf("%-*s   ", tableColWidth[4], state.SystemTime)
		fmt.Printf("%-*s   ", tableColWidth[5], state.Health)
		fmt.Printf("%-*d   ", tableColWidth[6], state.Restarts)
		fmt.Printf("%-*d   ", tableColWidth[7], state.ExitCode)
		fmt.Println()
	}

}
