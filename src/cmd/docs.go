package cmd

import (
	"github.com/spf13/cobra/doc"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// docsCmd represents the docs command
var docsCmd = &cobra.Command{
	Use:   "docs [OUTPUT_PATH]",
	Short: "Generate Process Compose markdown documentation",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		outPath := args[0]
		err := doc.GenMarkdownTree(rootCmd, outPath)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to generate docs")
		}
	},
	Hidden: true,
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
