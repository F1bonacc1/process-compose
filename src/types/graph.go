package types

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// DependencyNode represents a process in the dependency graph
type DependencyNode struct {
	Name      string                    `json:"name" yaml:"name"`
	Status    string                    `json:"process_status" yaml:"process_status"`
	IsReady   string                    `json:"is_ready" yaml:"is_ready"`
	DependsOn map[string]DependencyLink `json:"depends_on,omitempty" yaml:"depends_on,omitempty" swaggertype:"object"`
}

// DependencyLink wraps a node with the dependency condition
type DependencyLink struct {
	*DependencyNode `yaml:",inline"`
	Type            string `json:"dependency_type" yaml:"dependency_type"`
}

// DependencyGraph represents the full process dependency structure
type DependencyGraph struct {
	AllNodes map[string]*DependencyNode `json:"-" yaml:"-"`
	Nodes    map[string]*DependencyNode `json:"nodes" yaml:"nodes"`
}

// NewDependencyGraph creates an empty dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		AllNodes: make(map[string]*DependencyNode),
		Nodes:    make(map[string]*DependencyNode),
	}
}

// BuildDependencyGraph constructs a dependency graph from process configurations
func BuildDependencyGraph(processes Processes) *DependencyGraph {
	graph := NewDependencyGraph()

	// First pass: create all nodes
	for name := range processes {
		node := &DependencyNode{
			Name:      name,
			DependsOn: make(map[string]DependencyLink),
			IsReady:   "-",
			Status:    "Pending",
		}
		graph.AllNodes[name] = node
	}

	// Second pass: link dependencies recursively
	isDependedOn := make(map[string]bool)
	for name, proc := range processes {
		node := graph.AllNodes[name]
		for depName, depConfig := range proc.DependsOn {
			condition := "started" // default
			switch depConfig.Condition {
			case ProcessConditionCompleted:
				condition = "completed"
			case ProcessConditionCompletedSuccessfully:
				condition = "completed_successfully"
			case ProcessConditionHealthy:
				condition = "healthy"
			case ProcessConditionStarted:
				condition = "started"
			case ProcessConditionLogReady:
				condition = "log_ready"
			}

			if depNode, exists := graph.AllNodes[depName]; exists {
				node.DependsOn[depName] = DependencyLink{
					DependencyNode: depNode,
					Type:           condition,
				}
				isDependedOn[depName] = true
			}
		}
	}

	// Identify roots and leaves, filtering out isolated nodes
	for name, node := range graph.AllNodes {
		isRoot := len(node.DependsOn) == 0
		isLeaf := !isDependedOn[name]

		if isRoot && isLeaf {
			// Isolated node - remove from graph
			delete(graph.AllNodes, name)
			continue
		}

		if isLeaf {
			// Add to Nodes map for top-level JSON output (only leaves)
			graph.Nodes[name] = node
		}
	}

	return graph
}

// TransitiveDependents returns every process that depends on root, directly or
// transitively, ordered so that a process always appears after everything it
// depends on. root itself is excluded, as are deferred processes (disabled or
// foreground) - traversal still passes through them, so a running process is
// reached even when an intermediate one is disabled.
//
// This is the cascade order used by `watch`. Ordering is load-bearing: a
// dependent restarted before its dependency would resolve against the
// dependency's stale, already-completed incarnation and run against old output.
//
// The traversal operates in replica-name space, which is what processes is
// keyed by and what cloneReplicas rewrites depends_on to, so replicas need no
// special handling. Ties are broken lexicographically to keep the order
// deterministic and testable.
func TransitiveDependents(processes Processes, root string) []string {
	// Reverse adjacency: dependency -> processes that depend on it.
	dependents := make(map[string][]string, len(processes))
	for name, proc := range processes {
		for dep := range proc.DependsOn {
			dependents[dep] = append(dependents[dep], name)
		}
	}

	// Collect the closure reachable from root.
	inSet := make(map[string]bool)
	queue := []string{root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, dependent := range dependents[current] {
			if inSet[dependent] || dependent == root {
				continue
			}
			inSet[dependent] = true
			queue = append(queue, dependent)
		}
	}
	if len(inSet) == 0 {
		return nil
	}

	// Kahn's algorithm over the induced subgraph. root is excluded from the
	// set, so an edge back to it imposes no constraint - it is restarted first
	// by the caller regardless.
	inDegree := make(map[string]int, len(inSet))
	for name := range inSet {
		for dep := range processes[name].DependsOn {
			if inSet[dep] {
				inDegree[name]++
			}
		}
	}

	ready := make([]string, 0, len(inSet))
	for name := range inSet {
		if inDegree[name] == 0 {
			ready = append(ready, name)
		}
	}
	sort.Strings(ready)

	ordered := make([]string, 0, len(inSet))
	for len(ready) > 0 {
		name := ready[0]
		ready = ready[1:]
		ordered = append(ordered, name)

		promoted := make([]string, 0)
		for _, dependent := range dependents[name] {
			if !inSet[dependent] {
				continue
			}
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				promoted = append(promoted, dependent)
			}
		}
		if len(promoted) > 0 {
			ready = append(ready, promoted...)
			sort.Strings(ready)
		}
	}

	// A cycle is impossible - validateNoCircularDependencies rejects one at load
	// time - but never silently drop processes if that guarantee ever breaks.
	if len(ordered) < len(inSet) {
		for name := range inSet {
			if !slices.Contains(ordered, name) {
				ordered = append(ordered, name)
			}
		}
	}

	// Deferred processes must not be started by a cascade.
	result := make([]string, 0, len(ordered))
	for _, name := range ordered {
		proc := processes[name]
		if proc.IsDeferred() {
			continue
		}
		result = append(result, name)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// ToMermaid outputs the dependency graph in Mermaid flowchart format
func (g *DependencyGraph) ToMermaid() string {
	var sb strings.Builder
	sb.WriteString("flowchart LR\n")

	// Collect all edges and nodes
	edges := make([]string, 0)
	isDependedOn := make(map[string]bool)
	for name, node := range g.AllNodes {
		for depName := range node.DependsOn {
			edges = append(edges, fmt.Sprintf("    %s --> %s", sanitizeMermaidId(name), sanitizeMermaidId(depName)))
			isDependedOn[depName] = true
		}
	}

	for name, node := range g.AllNodes {
		if len(node.DependsOn) == 0 && !isDependedOn[name] {
			// Isolated node (though BuildDependencyGraph filters them, we keep this for consistency)
			edges = append(edges, fmt.Sprintf("    %s", sanitizeMermaidId(name)))
		}
	}

	sort.Strings(edges)
	for _, edge := range edges {
		sb.WriteString(edge)
		sb.WriteString("\n")
	}

	return sb.String()
}

// sanitizeMermaidId replaces characters that are invalid in Mermaid node IDs
func sanitizeMermaidId(name string) string {
	// Replace hyphens and dots with underscores for valid Mermaid IDs
	replacer := strings.NewReplacer("-", "_", ".", "_", " ", "_")
	return replacer.Replace(name)
}

// RebuildInternalIndices reconstructs AllNodes from the recursive Nodes map.
// This is useful on the client side after receiving the graph via JSON/YAML.
func (g *DependencyGraph) RebuildInternalIndices() {
	if g.AllNodes == nil {
		g.AllNodes = make(map[string]*DependencyNode)
	}

	var visit func(node *DependencyNode)
	visit = func(node *DependencyNode) {
		if node == nil {
			return
		}
		if _, exists := g.AllNodes[node.Name]; exists {
			return
		}
		g.AllNodes[node.Name] = node

		for _, link := range node.DependsOn {
			visit(link.DependencyNode)
		}
	}

	for _, node := range g.Nodes {
		visit(node)
	}
}
