package types

import (
	"strings"
	"testing"
)

func TestBuildDependencyGraph_LinearChain(t *testing.T) {
	// A -> B -> C -> D (D is root, A is leaf)
	processes := Processes{
		"A": {Name: "A", ReplicaName: "A", DependsOn: DependsOnConfig{"B": {}}},
		"B": {Name: "B", ReplicaName: "B", DependsOn: DependsOnConfig{"C": {}}},
		"C": {Name: "C", ReplicaName: "C", DependsOn: DependsOnConfig{"D": {}}},
		"D": {Name: "D", ReplicaName: "D"},
	}

	graph := BuildDependencyGraph(processes)

	// Check roots (no dependencies)
	if node, exists := graph.AllNodes["D"]; !exists || len(node.DependsOn) != 0 {
		t.Errorf("Expected D to be a root (no dependencies)")
	}

	// Check leaves (in Nodes map)
	if len(graph.Nodes) != 1 || graph.Nodes["A"] == nil {
		t.Errorf("Expected only leaf [A] in Nodes map, got %v", graph.Nodes)
	}

	// Check dependencies
	if len(graph.AllNodes["A"].DependsOn) != 1 {
		t.Errorf("A should have 1 dependency, got %d", len(graph.AllNodes["A"].DependsOn))
	}
	if _, ok := graph.AllNodes["A"].DependsOn["B"]; !ok {
		t.Errorf("A should depend on B")
	}

}

func TestBuildDependencyGraph_MultipleRoots(t *testing.T) {
	// postgres -> api, redis -> api, api -> frontend
	processes := Processes{
		"postgres": {Name: "postgres", ReplicaName: "postgres"},
		"redis":    {Name: "redis", ReplicaName: "redis"},
		"api":      {Name: "api", ReplicaName: "api", DependsOn: DependsOnConfig{"postgres": {}, "redis": {}}},
		"frontend": {Name: "frontend", ReplicaName: "frontend", DependsOn: DependsOnConfig{"api": {}}},
	}

	graph := BuildDependencyGraph(processes)

	// Check roots
	if node, exists := graph.AllNodes["postgres"]; !exists || len(node.DependsOn) != 0 {
		t.Errorf("postgres should be a root")
	}
	if node, exists := graph.AllNodes["redis"]; !exists || len(node.DependsOn) != 0 {
		t.Errorf("redis should be a root")
	}

	// Check leaves
	if len(graph.Nodes) != 1 || graph.Nodes["frontend"] == nil {
		t.Errorf("Expected only leaf [frontend] in Nodes map, got %v", graph.Nodes)
	}

	// Check api depends on both postgres and redis
	if len(graph.AllNodes["api"].DependsOn) != 2 {
		t.Errorf("api should depend on 2 processes, got %v", graph.AllNodes["api"].DependsOn)
	}
	if _, ok := graph.AllNodes["api"].DependsOn["postgres"]; !ok {
		t.Errorf("api should depend on postgres")
	}
	if _, ok := graph.AllNodes["api"].DependsOn["redis"]; !ok {
		t.Errorf("api should depend on redis")
	}
}

func TestBuildDependencyGraph_DiamondDependency(t *testing.T) {
	// A -> B, A -> C, B -> D, C -> D (diamond pattern)
	processes := Processes{
		"A": {Name: "A", ReplicaName: "A", DependsOn: DependsOnConfig{"B": {}, "C": {}}},
		"B": {Name: "B", ReplicaName: "B", DependsOn: DependsOnConfig{"D": {}}},
		"C": {Name: "C", ReplicaName: "C", DependsOn: DependsOnConfig{"D": {}}},
		"D": {Name: "D", ReplicaName: "D"},
	}

	graph := BuildDependencyGraph(processes)

	if node, exists := graph.AllNodes["D"]; !exists || len(node.DependsOn) != 0 {
		t.Errorf("Expected D to be root")
	}

	if len(graph.Nodes) != 1 || graph.Nodes["A"] == nil {
		t.Errorf("Expected leaf A in Nodes map, got %v", graph.Nodes)
	}

}

func TestBuildDependencyGraph_Empty(t *testing.T) {
	processes := Processes{}
	graph := BuildDependencyGraph(processes)

	if len(graph.AllNodes) != 0 {
		t.Errorf("Expected empty nodes, got %d", len(graph.AllNodes))
	}
	if len(graph.AllNodes) != 0 {
		t.Errorf("Expected empty nodes, got %d", len(graph.AllNodes))
	}
	if len(graph.Nodes) != 0 {
		t.Errorf("Expected empty Nodes map, got %d", len(graph.Nodes))
	}
}

func TestDependencyGraph_ToMermaid(t *testing.T) {
	processes := Processes{
		"postgres": {Name: "postgres", ReplicaName: "postgres"},
		"api":      {Name: "api", ReplicaName: "api", DependsOn: DependsOnConfig{"postgres": {}}},
	}

	graph := BuildDependencyGraph(processes)
	mermaid := graph.ToMermaid()

	if !strings.Contains(mermaid, "flowchart LR") {
		t.Error("Mermaid output should start with 'flowchart LR'")
	}
	if !strings.Contains(mermaid, "api --> postgres") {
		t.Errorf("Mermaid output should contain 'api --> postgres', got:\n%s", mermaid)
	}
}
