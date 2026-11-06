package main

import (
	"encoding/json"
	"fmt"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate a ChameleonDB schema",
	Long: `Validate a schema file for syntax and semantic errors.

If no file is specified, validates 'schema.cham' in the current directory
or, when there is none, the schemas configured in .chameleon.yml (schema.paths).

Examples:
  chameleon validate
  chameleon validate schema.cham
  chameleon validate path/to/schema.cham`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng := engine.NewEngineForCLI()

		schemaLabel, content, err := readValidateInput(args)
		if err != nil {
			return err
		}

		printInfo("Validating %s...", schemaLabel)

		// Validate using LoadSchemaFromStringRaw
		_, rawErr, err := eng.LoadSchemaFromStringRaw(content)

		if err != nil {
			printError("Validation failed")
			fmt.Println()

			// Try to parse as ValidationResultJson
			var validationResult struct {
				Valid  bool `json:"valid"`
				Errors []struct {
					Message    string  `json:"message"`
					Snippet    *string `json:"snippet"`
					Suggestion *string `json:"suggestion"`
				} `json:"errors"`
			}

			if jsonErr := json.Unmarshal([]byte(rawErr), &validationResult); jsonErr == nil {
				for _, errItem := range validationResult.Errors {
					fmt.Println(errItem.Message)
					if errItem.Snippet != nil {
						fmt.Println(*errItem.Snippet)
					}
					if errItem.Suggestion != nil {
						fmt.Println("Help: " + *errItem.Suggestion)
					}
				}
			} else {
				// Fallback: try to format as old error format
				formatted := engine.FormatError(rawErr)
				fmt.Println(formatted)
			}

			return fmt.Errorf("schema validation failed")
		}

		printSuccess("Schema is valid")
		if verbose {
			fmt.Println("\nValidation checks passed:")
			fmt.Println("  ✓ Syntax is correct")
			fmt.Println("  ✓ All entity references exist")
			fmt.Println("  ✓ Foreign keys are consistent")
			fmt.Println("  ✓ Primary keys are defined")
			fmt.Println("  ✓ No circular dependencies")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
