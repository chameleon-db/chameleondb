package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chameleon-db/chameleondb/chameleon/internal/cliout"
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
  • No tampering detection

Exits 0 when every check passes, 5 when the integrity check fails and 7 when
there is no vault. With --json it prints one document with the details.`,
	RunE: runVerify,
}

func init() {
	supportJSON(verifyCmd)
	rootCmd.AddCommand(verifyCmd)
}

// verifyVersion is the result for one registered version.
type verifyVersion struct {
	Version string `json:"version"`
	OK      bool   `json:"ok"`
	Issue   string `json:"issue,omitempty"`
}

// verifySchemaFile is the result for the merged schema file.
type verifySchemaFile struct {
	Exists bool `json:"exists"`
	// MatchesCurrent is nil when there is no current version to compare with.
	MatchesCurrent *bool `json:"matches_current,omitempty"`
}

type verifyReport struct {
	Valid      bool             `json:"valid"`
	Versions   []verifyVersion  `json:"versions"`
	Issues     []string         `json:"issues"`
	SchemaFile verifySchemaFile `json:"schema_file"`
}

func runVerify(cmd *cobra.Command, args []string) error {
	v := vault.NewVault(".")

	if !v.Exists() {
		fmt.Println("❌ No vault found")
		fmt.Println("   Run 'chameleon migrate' to initialize")
		return cliout.New(cliout.CodeNotInitialized, "no vault found").
			WithHint("run 'chameleon migrate' to initialize").AsSilent()
	}

	fmt.Println("🔍 Running Integrity Verification...")
	fmt.Println()

	// Load manifest
	fmt.Print("Vault:")
	if err := v.Load(); err != nil {
		fmt.Printf(" ❌\n")
		fmt.Printf("   Failed to load manifest: %v\n", err)
		return cliout.Errorf(cliout.CodeIntegrityFailed, "failed to load manifest: %w", err).AsSilent()
	}
	fmt.Println()

	// Verify manifest is valid JSON
	fmt.Print("  ✓ manifest.json is valid\n")

	// Verify each version
	result, err := v.VerifyIntegrity()
	if err != nil {
		fmt.Printf("❌ Verification failed: %v\n", err)
		return cliout.Errorf(cliout.CodeIntegrityFailed, "verification failed: %w", err).AsSilent()
	}

	report := verifyReport{Valid: result.Valid, Versions: []verifyVersion{}, Issues: append([]string{}, result.Issues...)}

	for _, version := range result.VersionsOK {
		fmt.Printf("  ✓ %s integrity OK\n", version)
		report.Versions = append(report.Versions, verifyVersion{Version: version, OK: true})
	}

	for i, version := range result.VersionsFail {
		fmt.Printf("  ❌ %s integrity FAILED\n", version)
		entry := verifyVersion{Version: version}
		if i < len(result.Issues) {
			fmt.Printf("     %s\n", result.Issues[i])
			entry.Issue = result.Issues[i]
		}
		report.Versions = append(report.Versions, entry)
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
		return cliout.Errorf(cliout.CodeInternal, "failed to get working directory: %w", err).AsSilent()
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
		report.SchemaFile.Exists = true

		// Check if matches current version
		if v.Manifest.CurrentVersion != "" {
			current, _ := v.GetCurrentVersion()
			currentHash, _ := v.ComputeSchemaHash(schemaPath)

			matches := current != nil && currentHash == current.Hash
			report.SchemaFile.MatchesCurrent = &matches
			if matches {
				fmt.Printf("  ✓ Matches %s hash\n", v.Manifest.CurrentVersion)
			} else {
				fmt.Printf("  ⚠️  Modified (not matching %s)\n", v.Manifest.CurrentVersion)
			}
		}
	}

	fmt.Println()
	setResult(report)

	// Summary
	if result.Valid {
		fmt.Println("✅ All checks passed")
		return nil
	}

	fmt.Printf("❌ %d integrity issues found\n", len(result.Issues))
	fmt.Println()
	fmt.Println("🔧 Recovery options:")
	fmt.Println("   • Check integrity.log for audit trail")
	fmt.Println("   • Review recent changes to vault files")
	fmt.Println("   • Contact your DBA if tampering is suspected")

	details := make([]cliout.Detail, 0, len(result.Issues))
	for _, issue := range result.Issues {
		details = append(details, cliout.Detail{Code: cliout.CodeIntegrityFailed, Message: issue})
	}
	return cliout.Errorf(cliout.CodeIntegrityFailed, "%d integrity issues found", len(result.Issues)).
		WithDetails(details...).
		WithHint("check .chameleon/vault/integrity.log for the audit trail").
		AsSilent()
}
