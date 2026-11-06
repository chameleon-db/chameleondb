package main

import (
	"fmt"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show ChameleonDB version",
	Long:  "Display the current version of ChameleonDB",
	Run: func(cmd *cobra.Command, args []string) {
		eng := engine.NewEngineForCLI()
		version := eng.Version()

		fmt.Printf("ChameleonDB v%s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
