package cmd

import (
	"encoding/json"
	"os"
	"github.com/spf13/cobra/doc"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/invopop/jsonschema"
	"github.com/stoewer/go-strcase"
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

// schemaCmd represents the schema command
var schemaCmd = &cobra.Command{
    Use:   "schema [OUTPUT_PATH]",
    Short: "Generate Process Compose JSON schema",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        outPath := args[0]
        reflector := jsonschema.Reflector{
            FieldNameTag: "yaml",
            KeyNamer: strcase.SnakeCase,
            AllowAdditionalProperties: true,
        }
        schema := reflector.Reflect(&types.Project{})
        data, err := json.MarshalIndent(schema, "", "  ")
        if err != nil {
			log.Fatal().Err(err).Msg("Failed to marshal schema")
        }
        err = os.WriteFile(outPath, data, 0644)
        if err != nil {
			log.Fatal().Err(err).Msg("Failed to write schema")
        }
    },
    Hidden: true,
}


func init() {
	rootCmd.AddCommand(docsCmd)
    rootCmd.AddCommand(schemaCmd)
}
