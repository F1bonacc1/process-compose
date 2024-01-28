package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
	"time"
)

var (
	withMemoryUsage = false
)

// stateCmd represents the state command
var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Get Process Compose project state",
	Run: func(cmd *cobra.Command, args []string) {
		pcClient := client.NewClient(*pcFlags.Address, *pcFlags.PortNum, *pcFlags.LogLength)
		state, err := pcClient.GetProjectState(withMemoryUsage)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get project state")
		}
		//pretty print state
		printState(state)
	},
}

func printState(state *types.ProjectState) {
	col := color.New(color.FgGreen)
	col.EnableColor()
	green := col.SprintFunc()

	longestKey := len("Running Processes")
	printStateLine("Hostname", state.HostName, longestKey, green)
	printStateLine("User", state.UserName, longestKey, green)
	printStateLine("Version", state.Version, longestKey, green)
	printStateLine("Up Time", state.UpTime.Round(time.Second).String(), longestKey, green)
	printStateLine("Processes", strconv.Itoa(state.ProcessNum), longestKey, green)
	printStateLine("Running Processes", strconv.Itoa(state.RunningProcessNum), longestKey, green)

	fmt.Printf("%s:\n", green("File Names"))
	for _, file := range state.FileNames {
		fmt.Printf("\t - %s\n", file)
	}

	if state.MemoryState != nil {
		printStateLine("Allocated MB", strconv.FormatUint(state.MemoryState.Allocated, 10), longestKey, green)
		printStateLine("Total Alloc MB", strconv.FormatUint(state.MemoryState.TotalAllocated, 10), longestKey, green)
		printStateLine("System MB", strconv.FormatUint(state.MemoryState.SystemMemory, 10), longestKey, green)
		printStateLine("GC Cycles", strconv.Itoa(int(state.MemoryState.GcCycles)), longestKey, green)
	}
}
func printStateLine(key, value string, longestKey int, green func(a ...interface{}) string) {
	dotPads := longestKey - len(key)
	padding := strings.Repeat(" ", dotPads)

	fmt.Printf("%s:%s %s\n", green(key), padding, value)
}

func init() {
	projectCmd.AddCommand(stateCmd)
	stateCmd.Flags().BoolVar(&withMemoryUsage, "with-memory", false, "check memory usage")
}
