package main

import (
	"os"

	"github.com/chameleon-db/chameleondb/chameleon/internal/cliout"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose bool

	// Colors
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	warningColor = color.New(color.FgYellow)
	infoColor    = color.New(color.FgCyan)
)

var rootCmd = &cobra.Command{
	Use:   "chameleon",
	Short: "ChameleonDB - Type-safe database access language",
	Long: `ChameleonDB is a graph-oriented, strongly-typed database access language
that compiles schemas to migrations and validates queries at compile time.

Get started:
  chameleon init myproject
  cd myproject
  chameleon validate
  chameleon migrate --apply`,
	SilenceUsage:      true,
	SilenceErrors:     true,
	PersistentPreRunE: prepareOutput,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "print one JSON document on stdout; human output goes to stderr (validate, status, verify, migrate, introspect)")
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return cliout.Wrap(cliout.CodeUsage, err)
	})
}

// Helper functions for consistent output
func printSuccess(format string, args ...interface{}) {
	successColor.Printf("✓ "+format+"\n", args...)
}

func printError(format string, args ...interface{}) {
	errorColor.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
}

func printWarning(format string, args ...interface{}) {
	warningColor.Printf("⚠ "+format+"\n", args...)
}

func printInfo(format string, args ...interface{}) {
	infoColor.Printf("ℹ "+format+"\n", args...)
}

func getConfigFromEnv() engine.ConnectorConfig {
	config, err := LoadConnectorConfig()
	if err != nil {
		printWarning("Could not read config: %v", err)
		return engine.DefaultConfig()
	}
	return config
}
