package main

import (
	"encoding/json"
	"fmt"

	"github.com/chameleon-db/chameleondb/chameleon/internal/cliout"
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
					Line       *int    `json:"line"`
					Column     *int    `json:"column"`
					Snippet    *string `json:"snippet"`
					Suggestion *string `json:"suggestion"`
				} `json:"errors"`
			}

			var details []cliout.Detail
			if jsonErr := json.Unmarshal([]byte(rawErr), &validationResult); jsonErr == nil {
				for _, errItem := range validationResult.Errors {
					fmt.Println(errItem.Message)
					d := cliout.Detail{Code: cliout.CodeSchemaInvalid, Message: errItem.Message, File: schemaLabel}
					if errItem.Line != nil {
						d.Line = *errItem.Line
					}
					if errItem.Column != nil {
						d.Column = *errItem.Column
					}
					if errItem.Snippet != nil {
						fmt.Println(*errItem.Snippet)
						d.Snippet = *errItem.Snippet
					}
					if errItem.Suggestion != nil {
						fmt.Println("Help: " + *errItem.Suggestion)
						d.Hint = *errItem.Suggestion
					}
					details = append(details, d)
				}
			} else {
				// Fallback: try to format as old error format
				formatted := engine.FormatError(rawErr)
				fmt.Println(formatted)
				details = append(details, cliout.Detail{Code: cliout.CodeSchemaInvalid, Message: formatted, File: schemaLabel})
			}

			setResult(map[string]any{"valid": false, "source": schemaLabel})
			return cliout.New(cliout.CodeSchemaInvalid, "schema validation failed").WithDetails(details...)
		}

		setResult(map[string]any{"valid": true, "source": schemaLabel})
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
	supportJSON(validateCmd)
	rootCmd.AddCommand(validateCmd)
}
