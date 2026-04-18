package cmd

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var analyzeCriticalChainCmd = &cobra.Command{
	Use:   "critical-chain [process...]",
	Short: "Print the critical process startup chain",
	Long: `Print a tree of processes ordered from the processes that nothing
depends on (top-level) down through their dependencies, annotated with
startup timings -- similar to 'systemd-analyze critical-chain'.

For each process two times are printed:

  @<offset>   Time after the project started that the process became ready.
              (For processes without a readiness probe this is the time the
              process was launched.)
  +<duration> Time the process spent between launch and becoming ready.
              Only shown for processes with a readiness signal (readiness
              probe, liveness probe, or 'ready_log_line').

If process names are given as arguments, only those processes (and their
dependency sub-chains) are printed; otherwise every top-level process is
printed.`,
	Run: runAnalyzeCriticalChain,
}

func init() {
	analyzeCmd.AddCommand(analyzeCriticalChainCmd)
}

func runAnalyzeCriticalChain(cmd *cobra.Command, args []string) {
	c := getClient()

	projectState, err := c.GetProjectState(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get project state: %v\n", err)
		os.Exit(1)
	}

	procStates, err := c.GetRemoteProcessesState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get process states: %v\n", err)
		os.Exit(1)
	}

	graph, err := c.GetDependencyGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get dependency graph: %v\n", err)
		os.Exit(1)
	}
	graph.RebuildInternalIndices()

	stateByName := make(map[string]*types.ProcessState, len(procStates.States))
	for i := range procStates.States {
		s := &procStates.States[i]
		stateByName[s.Name] = s
	}

	// Determine the starting set of nodes.
	var roots []*types.DependencyNode
	if len(args) > 0 {
		for _, name := range args {
			node, ok := graph.AllNodes[name]
			if !ok {
				fmt.Fprintf(os.Stderr, "unknown process (or process has no dependencies): %s\n", name)
				os.Exit(1)
			}
			roots = append(roots, node)
		}
	} else {
		// Top-level = processes that nothing depends on. graph.Nodes already
		// contains the leaves (see BuildDependencyGraph).
		for _, n := range graph.Nodes {
			roots = append(roots, n)
		}
		// Also include processes from the state that aren't in the graph at
		// all (isolated) so the user can see their timings too.
		inGraph := make(map[string]bool, len(graph.AllNodes))
		for name := range graph.AllNodes {
			inGraph[name] = true
		}
		for _, s := range procStates.States {
			if !inGraph[s.Name] {
				roots = append(roots, &types.DependencyNode{Name: s.Name})
			}
		}
	}

	sort.Slice(roots, func(i, j int) bool {
		ai := readyOffsetForSort(stateByName[roots[i].Name], projectState.StartTime)
		aj := readyOffsetForSort(stateByName[roots[j].Name], projectState.StartTime)
		if ai == aj {
			return roots[i].Name < roots[j].Name
		}
		return ai > aj
	})

	// Header
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Println("The time when unit became ready is printed after the \"@\" character.")
	fmt.Println("The time the unit took to become ready is printed after the \"+\" character.")
	fmt.Println()
	fmt.Printf("%s: %s\n", green("Project"), projectState.ProjectName)
	fmt.Printf("%s: %s\n", green("Started"), projectState.StartTime.Format(time.RFC3339))
	fmt.Printf("%s: %s\n", green("Up time"), projectState.UpTime.Round(time.Millisecond))
	fmt.Println()

	for i, root := range roots {
		last := i == len(roots)-1
		printCriticalChain(root, stateByName, projectState.StartTime, "", true, last)
	}
}

// readyOffsetForSort returns the duration from project start until the process
// became ready. Processes that never became ready sort to the top (return
// max duration) so the "slowest" chain is printed first.
func readyOffsetForSort(s *types.ProcessState, projectStart time.Time) time.Duration {
	if s == nil {
		return time.Duration(1<<62 - 1)
	}
	if s.ProcessReadyTime != nil {
		return s.ProcessReadyTime.Sub(projectStart)
	}
	if s.ProcessStartTime != nil {
		return s.ProcessStartTime.Sub(projectStart)
	}
	return time.Duration(1<<62 - 1)
}

func printCriticalChain(
	node *types.DependencyNode,
	stateByName map[string]*types.ProcessState,
	projectStart time.Time,
	prefix string,
	isRoot bool,
	isLast bool,
) {
	// Tree branch characters.
	branch := ""
	nextPrefix := prefix
	if !isRoot {
		if isLast {
			branch = "└─"
			nextPrefix = prefix + "  "
		} else {
			branch = "├─"
			nextPrefix = prefix + "│ "
		}
	}

	line := formatNodeLine(node, stateByName[node.Name], projectStart)
	fmt.Printf("%s%s%s\n", prefix, branch, line)

	// Sort dependencies by when they became ready (latest first), mirroring
	// systemd-analyze critical-chain.
	deps := make([]*types.DependencyNode, 0, len(node.DependsOn))
	for _, link := range node.DependsOn {
		if link.DependencyNode != nil {
			deps = append(deps, link.DependencyNode)
		}
	}
	sort.Slice(deps, func(i, j int) bool {
		ai := readyOffsetForSort(stateByName[deps[i].Name], projectStart)
		aj := readyOffsetForSort(stateByName[deps[j].Name], projectStart)
		if ai == aj {
			return deps[i].Name < deps[j].Name
		}
		return ai > aj
	})

	for i, dep := range deps {
		printCriticalChain(dep, stateByName, projectStart, nextPrefix, false, i == len(deps)-1)
	}
}

func formatNodeLine(
	node *types.DependencyNode,
	state *types.ProcessState,
	projectStart time.Time,
) string {
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	faint := color.New(color.Faint).SprintFunc()

	name := cyan(node.Name)

	if state == nil {
		return fmt.Sprintf("%s %s", name, faint("(no state)"))
	}

	var parts []string
	parts = append(parts, name)

	if state.ProcessReadyTime != nil {
		offset := state.ProcessReadyTime.Sub(projectStart)
		parts = append(parts, fmt.Sprintf("@%s", yellow(formatOffset(offset))))
	} else if state.ProcessStartTime != nil {
		offset := state.ProcessStartTime.Sub(projectStart)
		parts = append(parts,
			fmt.Sprintf("@%s", yellow(formatOffset(offset))),
			faint("(not ready)"))
	} else {
		parts = append(parts, faint("(not started)"))
	}

	// +<duration> only meaningful when there is a gap between start and ready.
	if state.ProcessStartTime != nil && state.ProcessReadyTime != nil {
		ready := state.ProcessReadyTime.Sub(*state.ProcessStartTime)
		if ready > 0 {
			parts = append(parts, fmt.Sprintf("+%s", yellow(formatDuration(ready))))
		}
	}

	// Status annotation if anything looks off.
	switch state.Status {
	case types.ProcessStateError, types.ProcessStateSkipped:
		parts = append(parts, red(fmt.Sprintf("[%s]", state.Status)))
	case types.ProcessStateCompleted, types.ProcessStateRunning:
		// expected -- no extra annotation
	default:
		if state.Status != "" {
			parts = append(parts, faint(fmt.Sprintf("[%s]", state.Status)))
		}
	}

	// Join with spaces.
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}

// formatOffset formats a duration representing "time since project start".
// Uses a more compact form for shorter durations and a minute-aware form for
// longer ones, mirroring `systemd-analyze`.
func formatOffset(d time.Duration) string {
	if d < 0 {
		return "?"
	}
	if d < time.Minute {
		return formatDuration(d)
	}
	mins := int(d / time.Minute)
	rem := d - time.Duration(mins)*time.Minute
	return fmt.Sprintf("%dmin %s", mins, formatDuration(rem))
}

// formatDuration prints a short human-friendly duration such as "5ms",
// "1.234s", or "2.500s".
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "?"
	}
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	secs := float64(d) / float64(time.Second)
	return fmt.Sprintf("%.3fs", secs)
}
