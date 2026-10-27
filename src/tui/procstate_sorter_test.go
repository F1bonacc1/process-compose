package tui

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
)

// sorterTestStates mirrors the #547 setup: one running process and several
// disabled ones that tie on every column except NAME.
func sorterTestStates() []types.ProcessState {
	states := []types.ProcessState{
		{Name: "alpha", Status: types.ProcessStateRunning, IsRunning: true, Pid: 42, Mem: 1024, CPU: 1.5, Age: 10},
	}
	for _, name := range []string{"tool-a", "tool-b", "tool-c", "tool-d", "tool-e", "tool-f"} {
		states = append(states, types.ProcessState{Name: name, Status: types.ProcessStateDisabled})
	}
	return states
}

func sortedNames(t *testing.T, column ColumnID, asc bool, states []types.ProcessState) []string {
	t.Helper()
	ps := &types.ProcessesState{States: states}
	if err := sortProcessesState(column, asc, ps); err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(ps.States))
	for i, s := range ps.States {
		names[i] = s.Name
	}
	return names
}

// The runner builds the states slice by ranging over a map, so the TUI gets
// them in a different order on every refresh. The sorted order must not
// depend on it.
func TestSortProcessesState_IndependentOfInputOrder(t *testing.T) {
	for column := ProcessStatePid; column <= ProcessStateExit; column++ {
		for _, asc := range []bool{true, false} {
			rnd := rand.New(rand.NewSource(547))
			want := sortedNames(t, column, asc, sorterTestStates())
			for range 50 {
				states := sorterTestStates()
				rnd.Shuffle(len(states), func(i, j int) { states[i], states[j] = states[j], states[i] })
				got := sortedNames(t, column, asc, states)
				if !slices.Equal(got, want) {
					t.Fatalf("sort by %s (asc=%v) depends on input order:\n got  %v\n want %v", column, asc, got, want)
				}
			}
		}
	}
}

func TestSortProcessesState_DescendingIsReverseOfAscending(t *testing.T) {
	for column := ProcessStatePid; column <= ProcessStateExit; column++ {
		asc := sortedNames(t, column, true, sorterTestStates())
		desc := sortedNames(t, column, false, sorterTestStates())
		slices.Reverse(desc)
		if !slices.Equal(asc, desc) {
			t.Errorf("sort by %s: descending is not the reverse of ascending:\n asc       %v\n desc rev. %v", column, asc, desc)
		}
	}
}

// sort.Slice requires a strict weak ordering, so an element is never less
// than itself, in either direction.
func TestReverse_IsIrreflexive(t *testing.T) {
	ps := &types.ProcessesState{States: sorterTestStates()}
	for column := ProcessStatePid; column <= ProcessStateExit; column++ {
		less := reverse(getSorter(column, ps))
		for i := range ps.States {
			if less(i, i) {
				t.Errorf("sort by %s: reversed comparator reports %s < itself", column, ps.States[i].Name)
				break
			}
		}
	}
}

// A process keeps its last MEM/CPU sample after it stops, but the table shows
// "-" for it, so it must sort with the other idle processes, not among the
// running ones by its stale sample.
func TestSortProcessesState_StoppedProcessSortsAsIdle(t *testing.T) {
	states := func() []types.ProcessState {
		return []types.ProcessState{
			{Name: "big", Status: types.ProcessStateRunning, IsRunning: true, Mem: 5800, CPU: 5.8},
			{Name: "small", Status: types.ProcessStateRunning, IsRunning: true, Mem: 3700, CPU: 3.7},
			{Name: "done", Status: types.ProcessStateCompleted, Mem: 3800, CPU: 3.8},
			{Name: "tool-a", Status: types.ProcessStateDisabled},
		}
	}
	want := []string{"done", "tool-a", "small", "big"}
	for _, column := range []ColumnID{ProcessStateMem, ProcessStateCPU} {
		if got := sortedNames(t, column, true, states()); !slices.Equal(got, want) {
			t.Errorf("sort by %s:\n got  %v\n want %v", column, got, want)
		}
	}
}
