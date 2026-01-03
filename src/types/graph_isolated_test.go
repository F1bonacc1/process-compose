package types

import (
	"testing"
)

func TestBuildDependencyGraph_Isolated(t *testing.T) {
	// A -> B. C is isolated.
	processes := Processes{
		"A": {Name: "A", ReplicaName: "A", DependsOn: DependsOnConfig{"B": {}}},
		"B": {Name: "B", ReplicaName: "B"},
		"C": {Name: "C", ReplicaName: "C"},
	}

	graph := BuildDependencyGraph(processes)

	// C should be removed
	if _, exists := graph.AllNodes["C"]; exists {
		t.Error("Isolated node C should be removed from graph")
	}

	// B is root. A is leaf.
	if node, exists := graph.AllNodes["B"]; !exists || len(node.DependsOn) != 0 {
		t.Errorf("Expected B to be root")
	}
	if len(graph.Nodes) != 1 || graph.Nodes["A"] == nil {
		t.Errorf("Expected leaf A in Nodes map")
	}
}

func TestNodesSubset_Filtering(t *testing.T) {
	// A -> B (A depends on B). A is Leaf. B is depended on.
	// C -> D. C is Leaf.
	processes := Processes{
		"A": {Name: "A", ReplicaName: "A", DependsOn: DependsOnConfig{"B": {}}},
		"B": {Name: "B", ReplicaName: "B"},
		"C": {Name: "C", ReplicaName: "C", DependsOn: DependsOnConfig{"D": {}}},
		"D": {Name: "D", ReplicaName: "D"},
	}

	graph := BuildDependencyGraph(processes)

	// Exposed Nodes map should ONLY contain leaves (A and C)
	if len(graph.Nodes) != 2 {
		t.Errorf("Expected 2 top-level nodes, got %d", len(graph.Nodes))
	}
	if _, ok := graph.Nodes["A"]; !ok {
		t.Error("Leaf A should be in Nodes map")
	}
	if _, ok := graph.Nodes["C"]; !ok {
		t.Error("Leaf C should be in Nodes map")
	}
	if _, ok := graph.Nodes["B"]; ok {
		t.Error("Non-leaf B should NOT be in Nodes map")
	}
	if _, ok := graph.Nodes["D"]; ok {
		t.Error("Non-leaf D should NOT be in Nodes map")
	}

	// AllNodes should contain everything
	if len(graph.AllNodes) != 4 {
		t.Errorf("Expected 4 total nodes, got %d", len(graph.AllNodes))
	}
}
