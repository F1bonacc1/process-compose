package types

import (
	"reflect"
	"testing"
)

// proc builds a process that depends on the given names. depends_on maps a
// process to what it needs, so TransitiveDependents walks these edges backwards.
func proc(name string, dependsOn ...string) ProcessConfig {
	cfg := ProcessConfig{Name: name, ReplicaName: name}
	if len(dependsOn) > 0 {
		cfg.DependsOn = make(DependsOnConfig, len(dependsOn))
		for _, dep := range dependsOn {
			cfg.DependsOn[dep] = ProcessDependency{Condition: ProcessConditionStarted}
		}
	}
	return cfg
}

func procs(configs ...ProcessConfig) Processes {
	out := make(Processes, len(configs))
	for _, cfg := range configs {
		out[cfg.ReplicaName] = cfg
	}
	return out
}

func TestTransitiveDependents(t *testing.T) {
	tests := []struct {
		name      string
		processes Processes
		root      string
		want      []string
	}{
		{
			name:      "no dependents",
			processes: procs(proc("assets"), proc("api")),
			root:      "assets",
			want:      nil,
		},
		{
			name:      "unknown root",
			processes: procs(proc("assets")),
			root:      "nope",
			want:      nil,
		},
		{
			name:      "single direct dependent",
			processes: procs(proc("assets"), proc("api", "assets")),
			root:      "assets",
			want:      []string{"api"},
		},
		{
			// A -> B -> C: changing A must reach C, which no single-level
			// reverse lookup would find.
			name:      "transitive chain",
			processes: procs(proc("a"), proc("b", "a"), proc("c", "b")),
			root:      "a",
			want:      []string{"b", "c"},
		},
		{
			// Diamond: d depends on b and c, both of which depend on a.
			// d must appear exactly once, and after both b and c.
			name:      "diamond dedupes and orders",
			processes: procs(proc("a"), proc("b", "a"), proc("c", "a"), proc("d", "b", "c")),
			root:      "a",
			want:      []string{"b", "c", "d"},
		},
		{
			// Ordering is load-bearing: a dependent restarted before its
			// dependency would resolve against the dependency's stale
			// completed incarnation. Names are chosen so that a naive
			// alphabetical sort would produce the wrong order.
			name:      "dependency precedes dependent regardless of name",
			processes: procs(proc("z"), proc("a", "z")),
			root:      "z",
			want:      []string{"a"},
		},
		{
			name:      "deep chain ordered leaf last",
			processes: procs(proc("root"), proc("z", "root"), proc("y", "z"), proc("x", "y")),
			root:      "root",
			want:      []string{"z", "y", "x"},
		},
		{
			name:      "only the root's subtree is returned",
			processes: procs(proc("a"), proc("b", "a"), proc("other"), proc("unrelated", "other")),
			root:      "a",
			want:      []string{"b"},
		},
		{
			// A cascade must not start what the user turned off.
			name: "deferred dependents are excluded",
			processes: func() Processes {
				disabled := proc("b", "a")
				disabled.Disabled = true
				foreground := proc("c", "a")
				foreground.IsForeground = true
				return procs(proc("a"), disabled, foreground)
			}(),
			root: "a",
			want: nil,
		},
		{
			// Traversal still passes through a disabled process so a running
			// one behind it is reached.
			name: "traversal passes through a deferred process",
			processes: func() Processes {
				disabled := proc("b", "a")
				disabled.Disabled = true
				return procs(proc("a"), disabled, proc("c", "b"))
			}(),
			root: "a",
			want: []string{"c"},
		},
		{
			name:      "root is never included in its own cascade",
			processes: procs(proc("a"), proc("b", "a"), proc("a2", "b")),
			root:      "a",
			want:      []string{"b", "a2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TransitiveDependents(tt.processes, tt.root)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TransitiveDependents() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTransitiveDependents_Deterministic guards the lexicographic tie-break.
// Go randomizes map iteration, so without it the cascade order would vary
// between runs and the restart sequence would be untestable.
func TestTransitiveDependents_Deterministic(t *testing.T) {
	processes := procs(
		proc("root"),
		proc("delta", "root"),
		proc("alpha", "root"),
		proc("charlie", "root"),
		proc("bravo", "root"),
	)
	want := []string{"alpha", "bravo", "charlie", "delta"}

	for i := range 50 {
		if got := TransitiveDependents(processes, "root"); !reflect.DeepEqual(got, want) {
			t.Fatalf("TransitiveDependents() = %v, want %v (iteration %d)", got, want, i)
		}
	}
}

// TestTransitiveDependents_ReplicaNames confirms the traversal works in
// replica-name space. cloneReplicas rewrites depends_on to replica names and
// keys Processes by them, so replicas need no special handling here.
func TestTransitiveDependents_ReplicaNames(t *testing.T) {
	processes := procs(
		proc("db"),
		proc("api-0", "db"),
		proc("api-1", "db"),
		proc("worker", "api-0"),
	)
	want := []string{"api-0", "api-1", "worker"}

	if got := TransitiveDependents(processes, "db"); !reflect.DeepEqual(got, want) {
		t.Errorf("TransitiveDependents() = %v, want %v", got, want)
	}
}
