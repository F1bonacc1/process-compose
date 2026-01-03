package types

import (
	"fmt"
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
