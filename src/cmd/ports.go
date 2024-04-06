package cmd

import (
	"fmt"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

// portsCmd represents the ports command
var portsCmd = &cobra.Command{
	Use:   "ports [PROCESS]",
	Short: "Get the ports that a process is listening on",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		ports, err := getClient().GetProcessPorts(name)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to get process %s ports", name)
			return
		}
		log.Info().Msgf("Process %s TCP ports: %v", name, ports.TcpPorts)
		fmt.Printf("Process %s TCP ports: %v\n", name, ports.TcpPorts)
	},
}

func init() {
	processCmd.AddCommand(portsCmd)
}
