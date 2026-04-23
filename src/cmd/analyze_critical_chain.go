package cmd

import (
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// unreadySortOrder is the sort-key returned for processes that never became
// ready (or have no recorded state). Using the maximum duration means these
// processes float to the top of the critical-chain listing, mirroring
// systemd-analyze which puts the slowest/stuck paths first.
const unreadySortOrder = time.Duration(math.MaxInt64)

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
				// The graph only contains processes that participate in a
				// dependency edge. If the name refers to an isolated process
				// that is still known to the project, synthesize a bare node
				// so its timings are still rendered.
				if _, exists := stateByName[name]; exists {
					node = &types.DependencyNode{Name: name}
				} else {
					fmt.Fprintf(os.Stderr, "unknown process: %s\n", name)
					os.Exit(1)
				}
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

	renderCriticalChain(os.Stdout, projectState, roots, stateByName)
}

// renderCriticalChain writes the critical-chain report for the given roots to
// w. Split out from runAnalyzeCriticalChain so it can be unit-tested without a
// running server.
func renderCriticalChain(
	w io.Writer,
	projectState *types.ProjectState,
	roots []*types.DependencyNode,
	stateByName map[string]*types.ProcessState,
) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Fprintln(w, "The time when unit became ready is printed after the \"@\" character.")
	fmt.Fprintln(w, "The time the unit took to become ready is printed after the \"+\" character.")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s: %s\n", green("Project"), projectState.ProjectName)
	fmt.Fprintf(w, "%s: %s\n", green("Started"), projectState.StartTime.Format(time.RFC3339))
	fmt.Fprintf(w, "%s: %s\n", green("Up time"), projectState.UpTime.Round(time.Millisecond))
	fmt.Fprintln(w)

	sortRootsByReadyTime(roots, stateByName, projectState.StartTime)

	for i, root := range roots {
		last := i == len(roots)-1
		printCriticalChain(w, root, stateByName, projectState.StartTime, "", true, last)
	}
}

// sortRootsByReadyTime sorts nodes in-place by descending ready-time (slowest
// first); ties are broken by ascending name.
func sortRootsByReadyTime(
	nodes []*types.DependencyNode,
	stateByName map[string]*types.ProcessState,
	projectStart time.Time,
) {
	sort.Slice(nodes, func(i, j int) bool {
		ai := readyOffsetForSort(stateByName[nodes[i].Name], projectStart)
		aj := readyOffsetForSort(stateByName[nodes[j].Name], projectStart)
		if ai == aj {
			return nodes[i].Name < nodes[j].Name
		}
		return ai > aj
	})
}

// readyOffsetForSort returns the duration from project start until the process
// became ready. Processes that never became ready return unreadySortOrder so
// the "slowest" / stuck chain is printed first.
func readyOffsetForSort(s *types.ProcessState, projectStart time.Time) time.Duration {
	if s == nil {
		return unreadySortOrder
	}
	if s.ProcessReadyTime != nil {
		return s.ProcessReadyTime.Sub(projectStart)
	}
	if s.ProcessStartTime != nil {
		return s.ProcessStartTime.Sub(projectStart)
	}
	return unreadySortOrder
}

func printCriticalChain(
	w io.Writer,
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
	fmt.Fprintf(w, "%s%s%s\n", prefix, branch, line)

	// Sort dependencies by when they became ready (latest first), mirroring
	// systemd-analyze critical-chain.
	deps := make([]*types.DependencyNode, 0, len(node.DependsOn))
	for _, link := range node.DependsOn {
		if link.DependencyNode != nil {
			deps = append(deps, link.DependencyNode)
		}
	}
	sortRootsByReadyTime(deps, stateByName, projectStart)

	for i, dep := range deps {
		printCriticalChain(w, dep, stateByName, projectStart, nextPrefix, false, i == len(deps)-1)
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

	return strings.Join(parts, " ")
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
