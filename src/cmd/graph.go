package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Display process dependency graph",
	Long:  `Output the process dependency graph in various formats (mermaid, json, yaml, ascii)`,
	Run: func(cmd *cobra.Command, args []string) {
		graph, err := getClient().GetDependencyGraph()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get dependency graph")
		}
		printGraph(graph)
	},
}

func printGraph(graph *types.DependencyGraph) {
	switch *pcFlags.OutputFormat {
	case "mermaid":
		fmt.Println(graph.ToMermaid())
	case "json":
		b, err := json.MarshalIndent(graph, "", "  ")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to marshal graph to JSON")
		}
		os.Stdout.Write(b)
	case "yaml":
		b, err := yaml.Marshal(graph)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to marshal graph to YAML")
		}
		os.Stdout.Write(b)
	default:
		// Default to ASCII tree
		printAsciiTree(graph)
	}
}

func printAsciiTree(graph *types.DependencyGraph) {
	fmt.Println("Dependency Graph")
	var leafNames []string
	for name := range graph.Nodes {
		leafNames = append(leafNames, name)
	}
	sort.Strings(leafNames)

	for i, leaf := range leafNames {
		printNode(graph, leaf, "", "", i == len(leafNames)-1)
	}
}

func printNode(graph *types.DependencyGraph, nodeName string, prefix string, condition string, isLast bool) {
	node, exists := graph.AllNodes[nodeName]
	if !exists {
		return
	}
	marker := "├── "
	if isLast {
		marker = "└── "
	}

	status := ""
	if node.Status != "" {
		status = fmt.Sprintf(" [%s]", node.Status)
	}

	fmt.Printf("%s%s%s%s%s\n", prefix, marker, nodeName, condition, status)

	newPrefix := prefix
	if isLast {
		newPrefix += "    "
	} else {
		newPrefix += "│   "
	}

	// Traverse dependencies (DependsOn) for Leaf -> Dependency order
	var depNames []string
	for name := range node.DependsOn {
		depNames = append(depNames, name)
	}
	sort.Strings(depNames)

	for i, dep := range depNames {
		link := node.DependsOn[dep]
		depCondition := fmt.Sprintf(" <%s>", link.Type)
		printNode(graph, dep, newPrefix, depCondition, i == len(depNames)-1)
	}
}

func init() {
	rootCmd.AddCommand(graphCmd)
	graphCmd.Flags().StringVarP(pcFlags.OutputFormat, "format", "f", "",
		"Output format: mermaid, json, yaml, or ascii (default)")
}
