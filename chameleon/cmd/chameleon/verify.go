package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chameleon-db/chameleondb/chameleon/internal/config"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify schema vault integrity",
	Long: `Run comprehensive integrity checks on the Schema Vault.
	
Verifies:
  • Manifest validity
  • Version file integrity (hash verification)
  • Schema file consistency
  • No tampering detection`,
	Run: runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

func runVerify(cmd *cobra.Command, args []string) {
	v := vault.NewVault(".")

	if !v.Exists() {
		fmt.Println("❌ No vault found")
		fmt.Println("   Run 'chameleon migrate' to initialize")
		os.Exit(1)
	}

	fmt.Println("🔍 Running Integrity Verification...")
	fmt.Println()

	// Load manifest
	fmt.Print("Vault:")
	if err := v.Load(); err != nil {
		fmt.Printf(" ❌\n")
		fmt.Printf("   Failed to load manifest: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()

	// Verify manifest is valid JSON
	fmt.Print("  ✓ manifest.json is valid\n")

	// Verify each version
	result, err := v.VerifyIntegrity()
	if err != nil {
		fmt.Printf("❌ Verification failed: %v\n", err)
		os.Exit(1)
	}

	for _, version := range result.VersionsOK {
		fmt.Printf("  ✓ %s integrity OK\n", version)
	}

	for i, version := range result.VersionsFail {
		fmt.Printf("  ❌ %s integrity FAILED\n", version)
		if i < len(result.Issues) {
			fmt.Printf("     %s\n", result.Issues[i])
		}
	}

	if len(result.VersionsFail) == 0 {
		fmt.Println("  ✓ No tampering detected")
	}

	fmt.Println()

	// Verify schema files
	fmt.Println("Schema Files:")
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	schemaPath := filepath.Join(workDir, ".chameleon", "state", "schema.merged.cham")
	loader := config.NewLoader(workDir)
	if cfg, loadErr := loader.Load(); loadErr == nil && cfg.Schema.MergedOutput != "" {
		schemaPath = cfg.Schema.MergedOutput
	}

	if _, err := os.Stat(schemaPath); err != nil {
		fmt.Println("  ⚠️  schema *.cham not found")
	} else {
		fmt.Println("  ✓ schema *.cham exists")

		// Check if matches current version
		if v.Manifest.CurrentVersion != "" {
			current, _ := v.GetCurrentVersion()
			currentHash, _ := v.ComputeSchemaHash(schemaPath)

			if current != nil && currentHash == current.Hash {
				fmt.Printf("  ✓ Matches %s hash\n", v.Manifest.CurrentVersion)
			} else {
				fmt.Printf("  ⚠️  Modified (not matching %s)\n", v.Manifest.CurrentVersion)
			}
		}
	}

	fmt.Println()

	// Summary
	if result.Valid {
		fmt.Println("✅ All checks passed")
		os.Exit(0)
	} else {
		fmt.Printf("❌ %d integrity issues found\n", len(result.Issues))
		fmt.Println()
		fmt.Println("🔧 Recovery options:")
		fmt.Println("   • Check integrity.log for audit trail")
		fmt.Println("   • Review recent changes to vault files")
		fmt.Println("   • Contact your DBA if tampering is suspected")
		os.Exit(1)
	}
}
