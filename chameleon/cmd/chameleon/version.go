package main

import (
	"context"
	"fmt"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show ChameleonDB version",
	Long:  "Display the current version of ChameleonDB CLI and core library",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		config, err := LoadConnectorConfig()
		if err != nil {
			return
		}

		eng := engine.NewEngine()
		connector := engine.NewConnector(config)
		if err := connector.Connect(ctx); err != nil {
			return
		}
		defer connector.Close()
		version := eng.Version()

		fmt.Printf("ChameleonDB v%s\n", version)

		if verbose {
			fmt.Println("\nComponents:")
			fmt.Printf("  CLI:  v%s\n", version)
			fmt.Printf("  Core: v%s (Rust)\n", version)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
