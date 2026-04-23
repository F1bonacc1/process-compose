package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/fatih/color"
)

// disableColor turns off ANSI color escapes so that the test can make exact
// string comparisons against the output of the renderer.
func disableColor(t *testing.T) {
	t.Helper()
	prev := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = prev })
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{"negative", -time.Second, "?"},
		{"sub-microsecond", 500 * time.Nanosecond, "500ns"},
		{"sub-millisecond", 500 * time.Microsecond, "500us"},
		{"sub-second", 234 * time.Millisecond, "234ms"},
		{"single-second", time.Second, "1.000s"},
		{"seconds-fractional", 1*time.Second + 234*time.Millisecond, "1.234s"},
		{"many-seconds", 2500 * time.Millisecond, "2.500s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.in); got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatOffset(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{"negative", -time.Millisecond, "?"},
		{"sub-minute", 900 * time.Millisecond, "900ms"},
		{"exactly-one-minute", time.Minute, "1min 0ns"},
		{"one-minute-plus", time.Minute + 250*time.Millisecond, "1min 250ms"},
		{"multi-minute", 4*time.Minute + 30*time.Second + 148*time.Millisecond, "4min 30.148s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatOffset(tt.in); got != tt.want {
				t.Errorf("formatOffset(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestReadyOffsetForSort(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ready := start.Add(3 * time.Second)
	launched := start.Add(1 * time.Second)

	tests := []struct {
		name string
		s    *types.ProcessState
		want time.Duration
	}{
		{"nil-state", nil, unreadySortOrder},
		{"ready-wins-over-start", &types.ProcessState{ProcessReadyTime: &ready, ProcessStartTime: &launched}, 3 * time.Second},
		{"start-only", &types.ProcessState{ProcessStartTime: &launched}, 1 * time.Second},
		{"neither", &types.ProcessState{}, unreadySortOrder},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readyOffsetForSort(tt.s, start); got != tt.want {
				t.Errorf("readyOffsetForSort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatNodeLine(t *testing.T) {
	disableColor(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	startedAt := start.Add(500 * time.Millisecond)
	readyAt := start.Add(1500 * time.Millisecond)

	tests := []struct {
		name  string
		node  *types.DependencyNode
		state *types.ProcessState
		want  string
	}{
		{
			name:  "nil-state",
			node:  &types.DependencyNode{Name: "orphan"},
			state: nil,
			want:  "orphan (no state)",
		},
		{
			name: "ready-with-probe-gap",
			node: &types.DependencyNode{Name: "postgres"},
			state: &types.ProcessState{
				Status:           types.ProcessStateRunning,
				ProcessStartTime: &startedAt,
				ProcessReadyTime: &readyAt,
			},
			want: "postgres @1.500s +1.000s",
		},
		{
			name: "ready-no-gap", // no readiness probe -- ready == start
			node: &types.DependencyNode{Name: "bare"},
			state: &types.ProcessState{
				Status:           types.ProcessStateRunning,
				ProcessStartTime: &startedAt,
				ProcessReadyTime: &startedAt,
			},
			want: "bare @500ms",
		},
		{
			name: "launched-but-not-ready",
			node: &types.DependencyNode{Name: "stuck"},
			state: &types.ProcessState{
				Status:           types.ProcessStateRunning,
				ProcessStartTime: &startedAt,
			},
			want: "stuck @500ms (not ready)",
		},
		{
			name:  "never-started-pending",
			node:  &types.DependencyNode{Name: "pend"},
			state: &types.ProcessState{Status: types.ProcessStatePending},
			want:  "pend (not started) [Pending]",
		},
		{
			name:  "error-status",
			node:  &types.DependencyNode{Name: "err"},
			state: &types.ProcessState{Status: types.ProcessStateError},
			want:  "err (not started) [Error]",
		},
		{
			name:  "skipped-status",
			node:  &types.DependencyNode{Name: "skip"},
			state: &types.ProcessState{Status: types.ProcessStateSkipped},
			want:  "skip (not started) [Skipped]",
		},
		{
			name: "completed-no-annotation",
			node: &types.DependencyNode{Name: "done"},
			state: &types.ProcessState{
				Status:           types.ProcessStateCompleted,
				ProcessStartTime: &startedAt,
				ProcessReadyTime: &startedAt,
			},
			want: "done @500ms",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNodeLine(tt.node, tt.state, start)
			if got != tt.want {
				t.Errorf("formatNodeLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSortRootsByReadyTime(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	mkReady := func(d time.Duration) *types.ProcessState {
		t := start.Add(d)
		return &types.ProcessState{ProcessReadyTime: &t, ProcessStartTime: &t}
	}

	stateByName := map[string]*types.ProcessState{
		"fast":    mkReady(100 * time.Millisecond),
		"slow":    mkReady(5 * time.Second),
		"medium":  mkReady(1 * time.Second),
		"unready": {}, // no times -> sorts to top
		// "tiebreak-b" and "tiebreak-a" share identical ready-time; alphabetical wins.
		"tiebreak-b": mkReady(2 * time.Second),
		"tiebreak-a": mkReady(2 * time.Second),
	}

	nodes := []*types.DependencyNode{
		{Name: "fast"},
		{Name: "slow"},
		{Name: "medium"},
		{Name: "unready"},
		{Name: "tiebreak-b"},
		{Name: "tiebreak-a"},
	}

	sortRootsByReadyTime(nodes, stateByName, start)

	got := make([]string, len(nodes))
	for i, n := range nodes {
		got[i] = n.Name
	}
	want := []string{"unready", "slow", "tiebreak-a", "tiebreak-b", "medium", "fast"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("sortRootsByReadyTime() order = %v, want %v", got, want)
	}
}

func TestRenderCriticalChainTree(t *testing.T) {
	disableColor(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mkReady := func(d time.Duration) *types.ProcessState {
		t := start.Add(d)
		return &types.ProcessState{
			Status:           types.ProcessStateCompleted,
			ProcessStartTime: &t,
			ProcessReadyTime: &t,
		}
	}
	mkReadyWithGap := func(launchOffset, readyOffset time.Duration) *types.ProcessState {
		launch := start.Add(launchOffset)
		ready := start.Add(readyOffset)
		return &types.ProcessState{
			Status:           types.ProcessStateRunning,
			ProcessStartTime: &launch,
			ProcessReadyTime: &ready,
		}
	}

	// Graph:
	//   app -> db, cache
	//   web -> app
	// Only leaves (processes nothing depends on) are roots: "web".
	dbNode := &types.DependencyNode{Name: "db"}
	cacheNode := &types.DependencyNode{Name: "cache"}
	appNode := &types.DependencyNode{
		Name: "app",
		DependsOn: map[string]types.DependencyLink{
			"db":    {DependencyNode: dbNode, Type: "healthy"},
			"cache": {DependencyNode: cacheNode, Type: "healthy"},
		},
	}
	webNode := &types.DependencyNode{
		Name: "web",
		DependsOn: map[string]types.DependencyLink{
			"app": {DependencyNode: appNode, Type: "healthy"},
		},
	}

	stateByName := map[string]*types.ProcessState{
		"web":   mkReadyWithGap(400*time.Millisecond, 5*time.Second),
		"app":   mkReadyWithGap(200*time.Millisecond, 3*time.Second),
		"db":    mkReady(2 * time.Second),   // slower -> first dep of app
		"cache": mkReady(500 * time.Millisecond),
	}

	projectState := &types.ProjectState{
		ProjectName: "test-project",
		StartTime:   start,
		UpTime:      10 * time.Second,
	}

	var buf bytes.Buffer
	renderCriticalChain(&buf, projectState, []*types.DependencyNode{webNode}, stateByName)
	out := buf.String()

	// Header assertions.
	if !strings.Contains(out, "Project: test-project") {
		t.Errorf("header missing project name:\n%s", out)
	}
	if !strings.Contains(out, "Up time: 10s") {
		t.Errorf("header missing up time:\n%s", out)
	}

	// Core tree assertions: web -> app -> (db before cache).
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// Find the tree lines (after the header blank line).
	treeStart := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "web ") {
			treeStart = i
			break
		}
	}
	if treeStart < 0 {
		t.Fatalf("couldn't locate web root line in output:\n%s", out)
	}

	got := lines[treeStart:]
	want := []string{
		"web @5.000s +4.600s",
		"└─app @3.000s +2.800s",
		"  ├─db @2.000s",
		"  └─cache @500ms",
	}
	if len(got) != len(want) {
		t.Fatalf("tree line count = %d, want %d\nfull output:\n%s", len(got), len(want), out)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tree line %d:\n  got:  %q\n  want: %q", i, got[i], want[i])
		}
	}
}

func TestRenderCriticalChainSortsSiblingsSlowestFirst(t *testing.T) {
	disableColor(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mk := func(d time.Duration) *types.ProcessState {
		t := start.Add(d)
		return &types.ProcessState{
			Status:           types.ProcessStateCompleted,
			ProcessStartTime: &t,
			ProcessReadyTime: &t,
		}
	}
	a := &types.DependencyNode{Name: "a"}
	b := &types.DependencyNode{Name: "b"}
	c := &types.DependencyNode{Name: "c"}
	parent := &types.DependencyNode{
		Name: "parent",
		DependsOn: map[string]types.DependencyLink{
			"a": {DependencyNode: a},
			"b": {DependencyNode: b},
			"c": {DependencyNode: c},
		},
	}

	stateByName := map[string]*types.ProcessState{
		"parent": mk(4 * time.Second),
		"a":      mk(1 * time.Second),
		"b":      mk(3 * time.Second),
		"c":      mk(2 * time.Second),
	}

	var buf bytes.Buffer
	renderCriticalChain(&buf, &types.ProjectState{
		ProjectName: "p",
		StartTime:   start,
		UpTime:      5 * time.Second,
	}, []*types.DependencyNode{parent}, stateByName)

	// Expected sibling order: b (3s) > c (2s) > a (1s). Using descending ready time.
	depLines := []string{}
	for _, l := range strings.Split(buf.String(), "\n") {
		// tree lines rendered for direct children use "├─" or "└─".
		if strings.HasPrefix(l, "├─") || strings.HasPrefix(l, "└─") {
			depLines = append(depLines, l)
		}
	}
	want := []string{
		"├─b @3.000s",
		"├─c @2.000s",
		"└─a @1.000s",
	}
	if strings.Join(depLines, "\n") != strings.Join(want, "\n") {
		t.Errorf("sibling order = %v, want %v", depLines, want)
	}
}
