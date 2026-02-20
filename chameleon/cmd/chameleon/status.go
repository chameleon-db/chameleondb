package main

import (
	"fmt"
	"os"
	"time"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show ChameleonDB status",
	Long:  `Display current status of schema, vault, and database connection.`,
	Run:   runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	v := vault.NewVault(".")

	fmt.Println("🗂️  ChameleonDB Status")
	fmt.Println("────────────────────────────────────────────")
	fmt.Println()

	// Schema status
	showSchemaStatus(v)
	fmt.Println()

	// Vault status
	showVaultStatus(v)
	fmt.Println()

	// Configuration
	showConfiguration()
}

func showSchemaStatus(v *vault.Vault) {
	fmt.Println("Schema:")

	if !v.Exists() {
		fmt.Println("  Status:          ⚠️  No vault initialized")
		fmt.Println("  Action:          Run 'chameleon migrate' to start")
		return
	}

	status, err := v.GetStatus()
	if err != nil {
		fmt.Printf("  Status:          ❌ Error: %v\n", err)
		return
	}

	if status.CurrentVersion == "" {
		fmt.Println("  Status:          ⚠️  No versions registered")
		return
	}

	fmt.Printf("  Current version:  %s\n", status.CurrentVersion)

	// Get current version details
	entry, err := v.GetCurrentVersion()
	if err == nil {
		fmt.Printf("  Hash:            %s...\n", entry.Hash[:12])
		fmt.Printf("  Last modified:   %s\n", formatTimeSince(entry.Timestamp))
	}

	// Check if schema file matches current version
	schemaPath := "schema.cham"
	if _, err := os.Stat(schemaPath); err == nil {
		changed, _, _ := v.DetectChanges(schemaPath)
		if changed {
			fmt.Println("  Status:          ⚠️  Schema modified (not registered)")
		} else {
			fmt.Println("  Status:          ✓ Up to date")
		}
	}
}

func showVaultStatus(v *vault.Vault) {
	fmt.Println("Vault:")

	if !v.Exists() {
		fmt.Println("  Status:          Not initialized")
		return
	}

	status, err := v.GetStatus()
	if err != nil {
		fmt.Printf("  Status:          Error: %v\n", err)
		return
	}

	fmt.Printf("  Versions:        %d registered\n", status.TotalVersions)

	// Verify integrity
	result, err := v.VerifyIntegrity()
	if err != nil {
		fmt.Printf("  Integrity:       ❌ Error: %v\n", err)
	} else if result.Valid {
		fmt.Println("  Integrity:       ✓ OK")
	} else {
		fmt.Printf("  Integrity:       ❌ %d issues\n", len(result.Issues))
	}

	mode, err := v.GetParanoidMode()
	if err == nil {
		modeIcon := getModeIcon(mode)
		fmt.Printf("  Mode:            %s %s\n", modeIcon, mode)
	}
}

func showConfiguration() {
	fmt.Println("Configuration:")
	fmt.Println("  Debug Level:     off")
	// More config options can be added here
}

func formatTimeSince(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		return fmt.Sprintf("%d minutes ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func getModeIcon(mode string) string {
	switch mode {
	case "readonly":
		return "🛡️"
	case "standard":
		return "⚙️"
	case "privileged":
		return "👑"
	case "emergency":
		return "🚨"
	default:
		return "❓"
	}
}
