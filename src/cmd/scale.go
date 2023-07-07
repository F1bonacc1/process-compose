/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/rs/zerolog/log"
	"strconv"

	"github.com/spf13/cobra"
)

// scaleCmd represents the scale command
var scaleCmd = &cobra.Command{
	Use:   "scale [PROCESS] [COUNT]",
	Short: "Scale a process to a given count",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		count, err := strconv.Atoi(args[1])
		if err != nil {
			log.Error().Msgf("second argument must be an integer: %v", err)
			return
		}
		err = client.ScaleProcess(pcAddress, port, name, count)
		if err != nil {
			log.Error().Msgf("Failed to scale processes %s: %v", name, err)
			fmt.Println(err.Error())
			return
		}
		log.Info().Msgf("Process %s scaled to %s", name, args[1])
	},
}

func init() {
	processCmd.AddCommand(scaleCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// scaleCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// scaleCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
