package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	updateVerboseOutput = false
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an already running process-compose instance by passing an updated process-compose.yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		runUpdateCmd(args) // TODO: add error
	},
}

func runUpdateCmd(args []string) {
	project, err := loader.Load(opts)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load project")
	}
	status, err := getClient().UpdateProject(project)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to update project")
	}
	printStatus(status)
}

func printStatus(updateStatus map[string]string) {
	if len(updateStatus) == 0 {
		fmt.Println("No processes were updated")
		return
	}
	if updateVerboseOutput {
		printStatusAsTable(updateStatus)
	} else {
		fmt.Println("Project updated successfully")
	}
	for name, status := range updateStatus {
		log.Debug().Msgf("%s: %s", name, status)
	}
}

func printStatusAsTable(updateStatus map[string]string) {
	colStatus := "STATUS"
	colName := "PROCESS"
	// Calculate column widths
	maxNameWidth := len(colName)
	for proc := range updateStatus {
		if len(proc) > maxNameWidth {
			maxNameWidth = len(proc)
		}
	}
	fmt.Printf("  %-*s  %s\n", maxNameWidth, colName, colStatus)
	for proc, status := range updateStatus {
		procWithIcon := getStatusIcon(status) + " " + proc
		fmt.Printf("%-*s  %s\n", maxNameWidth+2, procWithIcon, status)
	}
}

func getStatusIcon(status string) string {
	switch status {
	case types.ProcessUpdateUpdated:
		return "↺"
	case types.ProcessUpdateAdded:
		return "▲"
	case types.ProcessUpdateRemoved:
		return "▼"
	case types.ProcessUpdateError:
		return "✘"
	default:
		return "?"
	}
}

func init() {
	projectCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringArrayVarP(&opts.FileNames, "config", "f", config.GetConfigDefault(), "path to config files to load (env: "+config.EnvVarNameConfig+")")
	updateCmd.Flags().BoolVarP(&updateVerboseOutput, "verbose", "v", updateVerboseOutput, "verbose output")
	updateCmd.MarkFlagRequired("config")
}
