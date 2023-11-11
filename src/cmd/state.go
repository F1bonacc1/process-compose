/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
	"time"
)

// stateCmd represents the state command
var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Get Process Compose project state",
	Run: func(cmd *cobra.Command, args []string) {
		pcClient := client.NewClient(*pcFlags.Address, *pcFlags.PortNum, *pcFlags.LogLength)
		state, err := pcClient.GetProjectState()
		if err != nil {
			logFatal(err, "failed to get project state")
		}
		//pretty print state
		printState(state)
	},
}

func printState(state *types.ProjectState) {

	longestKey := len("Running Processes")
	printStateLine("Hostname", state.HostName, longestKey)
	printStateLine("User", state.UserName, longestKey)
	printStateLine("Version", state.Version, longestKey)
	printStateLine("Up Time", state.UpTime.Round(time.Second).String(), longestKey)
	printStateLine("Processes", strconv.Itoa(state.ProcessNum), longestKey)
	printStateLine("Running Processes", strconv.Itoa(state.RunningProcessNum), longestKey)

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s:\n", green("File Names"))
	for _, file := range state.FileNames {
		fmt.Printf("\t - %s\n", file)
	}
}
func printStateLine(key, value string, longestKey int) {
	dotPads := longestKey - len(key)
	padding := strings.Repeat(" ", dotPads)

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s:%s %s\n", green(key), padding, value)
}

func init() {
	projectCmd.AddCommand(stateCmd)
}
