package tui

import (
	"fmt"

	"sort"
	"strings"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type graphDialog struct {
	*tview.TreeView
	root     *tview.TreeNode
	closeFn  func()
	maxWidth int
}

// newGraphDialog creates a dependency graph dialog using tview.TreeView
func newGraphDialog(closeFn func()) *graphDialog {
	root := tview.NewTreeNode("Root").SetSelectable(false)
	tree := tview.NewTreeView().
		SetRoot(root).
		SetTopLevel(1)
	tree.SetBorder(true).SetTitle("Dependencies (Esc to close)")

	// Handle node selection to toggle expansion
	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		node.SetExpanded(!node.IsExpanded())
		text := node.GetText()
		if strings.HasPrefix(text, "▾ ") {
			node.SetText(strings.Replace(text, "▾ ", "▸ ", 1))
		} else if strings.HasPrefix(text, "▸ ") {
			node.SetText(strings.Replace(text, "▸ ", "▾ ", 1))
		}
	})

	// Handle input to close dialog
	tree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			closeFn()
			return nil
		}
		return event
	})

	return &graphDialog{TreeView: tree, root: root, closeFn: closeFn}
}

// buildTree populates the tree from dependency graph with status colors, traversing Leaves -> Dependencies
func (g *graphDialog) buildTree(graph *types.DependencyGraph, states *types.ProcessesState) {
	g.root.ClearChildren()

	// Index states for faster lookup
	statesMap := make(map[string]types.ProcessState)
	for _, s := range states.States {
		statesMap[s.Name] = s
	}

	// Add leaf processes (top-level apps/processes that others don't depend on)
	var leafNames []string
	for name := range graph.Nodes {
		leafNames = append(leafNames, name)
	}
	sort.Strings(leafNames)

	for _, leafName := range leafNames {
		node := g.createProcessNode(leafName, statesMap, "")
		g.addDependencies(node, leafName, graph, statesMap, 0)
		g.root.AddChild(node)
	}

	// Set initial focus to the first leaf node
	children := g.root.GetChildren()
	if len(children) > 0 {
		g.SetCurrentNode(children[0])
	}
}

// GetWidth returns the calculated maximum width of the tree content plus some padding
func (g *graphDialog) GetWidth() int {
	// Add padding for borders and icons
	return g.maxWidth + 8
}

// createProcessNode creates a tree node with status-colored text
func (g *graphDialog) createProcessNode(name string, states map[string]types.ProcessState, condition string) *tview.TreeNode {
	node := tview.NewTreeNode(name)
	text := name
	if state, ok := states[name]; ok {
		node.SetColor(statusColor(state.Status))
		text = fmt.Sprintf("%s [%s]", name, state.Status)
	}
	if condition != "" {
		text += fmt.Sprintf(" <%s>", condition)
	}
	node.SetText(tview.Escape(text))
	node.SetReference(name)
	node.SetExpanded(true)

	// Base maxWidth on text length. Indentation is added in addDependencies.
	if len(text) > g.maxWidth {
		g.maxWidth = len(text)
	}

	return node
}

// addDependencies recursively adds dependencies to the tree
func (g *graphDialog) addDependencies(parentNode *tview.TreeNode, parentName string, graph *types.DependencyGraph, states map[string]types.ProcessState, level int) {
	node, exists := graph.AllNodes[parentName]
	if !exists {
		return
	}

	if len(node.DependsOn) > 0 {
		parentNode.SetText("▾ " + parentNode.GetText())
	}

	var depNames []string
	for name := range node.DependsOn {
		depNames = append(depNames, name)
	}
	sort.Strings(depNames)

	for _, depName := range depNames {
		link := node.DependsOn[depName]
		condition := link.Type
		childNode := g.createProcessNode(depName, states, condition)

		// Update maxWidth with indentation (tview default is 2 characters per level for TreeView)
		indent := (level + 1) * 2
		if len(childNode.GetText())+indent > g.maxWidth {
			g.maxWidth = len(childNode.GetText()) + indent
		}

		g.addDependencies(childNode, depName, graph, states, level+1)
		parentNode.AddChild(childNode)
	}
}

// statusColor returns tcell.Color based on process status
func statusColor(status string) tcell.Color {
	switch status {
	case "Running":
		return tcell.ColorGreen
	case "Completed":
		return tcell.ColorBlue
	case "Error", "Failed":
		return tcell.ColorRed
	case "Pending", "Launching":
		return tcell.ColorYellow
	default:
		return tcell.ColorWhite
	}
}
