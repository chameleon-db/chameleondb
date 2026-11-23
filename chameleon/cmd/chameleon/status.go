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
	Long: `Display current status of schema, vault, and database connection.

With --json it prints one document with the same information. status reports;
it exits 0 even when the vault has problems. Use 'chameleon verify' for a
pass/fail check.`,
	Run: runStatus,
}

func init() {
	supportJSON(statusCmd)
	rootCmd.AddCommand(statusCmd)
}

// Schema states reported by status.
const (
	schemaStateNoVault    = "no_vault"
	schemaStateNoVersions = "no_versions"
	schemaStateUpToDate   = "up_to_date"
	schemaStateModified   = "modified"
	// schemaStateUnknown: the vault or the project schemas could not be read.
	schemaStateUnknown = "unknown"
)

// statusSchema is the schema half of the status report.
type statusSchema struct {
	State          string     `json:"state"`
	CurrentVersion string     `json:"current_version,omitempty"`
	Hash           string     `json:"hash,omitempty"`
	LastModified   *time.Time `json:"last_modified,omitempty"`
	Error          string     `json:"error,omitempty"`
}

// statusVault is the vault half of the status report.
type statusVault struct {
	Initialized bool     `json:"initialized"`
	Versions    int      `json:"versions"`
	Integrity   string   `json:"integrity,omitempty"` // ok, failed or error
	Issues      []string `json:"issues"`
	Mode        string   `json:"mode,omitempty"`
	Error       string   `json:"error,omitempty"`
}

type statusReport struct {
	Schema statusSchema `json:"schema"`
	Vault  statusVault  `json:"vault"`
}

func runStatus(cmd *cobra.Command, args []string) {
	v := vault.NewVault(".")

	workDir, wdErr := os.Getwd()
	report := buildStatusReport(v, workDir, wdErr)
	setResult(report)

	fmt.Println("🗂️  ChameleonDB Status")
	fmt.Println("────────────────────────────────────────────")
	fmt.Println()

	printSchemaStatus(report.Schema)
	fmt.Println()

	printVaultStatus(report.Vault)
	fmt.Println()

	// Configuration
	showConfiguration()
}

// buildStatusReport gathers what status shows. It never fails: a part that
// cannot be read is reported with its error.
func buildStatusReport(v *vault.Vault, workDir string, wdErr error) statusReport {
	return statusReport{
		Schema: buildSchemaStatus(v, workDir, wdErr),
		Vault:  buildVaultStatus(v),
	}
}

func buildSchemaStatus(v *vault.Vault, workDir string, wdErr error) statusSchema {
	if !v.Exists() {
		return statusSchema{State: schemaStateNoVault}
	}

	status, err := v.GetStatus()
	if err != nil {
		return statusSchema{State: schemaStateUnknown, Error: err.Error()}
	}

	if status.CurrentVersion == "" {
		return statusSchema{State: schemaStateNoVersions}
	}

	out := statusSchema{CurrentVersion: status.CurrentVersion}

	// Get current version details
	if entry, err := v.GetCurrentVersion(); err == nil {
		out.Hash = entry.Hash
		ts := entry.Timestamp
		out.LastModified = &ts
	}

	// Compare the project schemas (as 'migrate' merges them) with the current version
	if wdErr != nil {
		out.State = schemaStateUnknown
		out.Error = "Could not get working directory: " + wdErr.Error()
		return out
	}

	merged, err := loadProjectSchema(workDir)
	if err != nil {
		out.State = schemaStateUnknown
		out.Error = "Could not read project schemas: " + err.Error()
		return out
	}

	changed, err := schemaDiffersFromVault(v, merged)
	if err != nil {
		out.State = schemaStateUnknown
		out.Error = "Could not compare schema: " + err.Error()
		return out
	}
	if changed {
		out.State = schemaStateModified
	} else {
		out.State = schemaStateUpToDate
	}
	return out
}

func printSchemaStatus(s statusSchema) {
	fmt.Println("Schema:")

	switch s.State {
	case schemaStateNoVault:
		fmt.Println("  Status:          ⚠️  No vault initialized")
		fmt.Println("  Action:          Run 'chameleon migrate' to start")
		return
	case schemaStateNoVersions:
		fmt.Println("  Status:          ⚠️  No versions registered")
		return
	}

	if s.CurrentVersion == "" {
		// The vault itself could not be read.
		fmt.Printf("  Status:          ❌ Error: %s\n", s.Error)
		return
	}

	fmt.Printf("  Current version:  %s\n", s.CurrentVersion)
	if s.LastModified != nil {
		fmt.Printf("  Hash:            %s...\n", s.Hash[:12])
		fmt.Printf("  Last modified:   %s\n", formatTimeSince(*s.LastModified))
	}

	switch s.State {
	case schemaStateModified:
		fmt.Println("  Status:          ⚠️  Schema modified (not registered)")
	case schemaStateUpToDate:
		fmt.Println("  Status:          ✓ Up to date")
	default:
		fmt.Printf("  Status:          ⚠️  %s\n", s.Error)
	}
}

// schemaDiffersFromVault reports whether the merged schema differs from the
// version the vault has registered. The vault hashes files, so the content is
// written to a temporary one.
func schemaDiffersFromVault(v *vault.Vault, merged string) (bool, error) {
	tmp, err := os.CreateTemp("", "chameleon-status-*.cham")
	if err != nil {
		return false, err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(merged); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}

	changed, _, err := v.DetectChanges(tmp.Name())
	return changed, err
}

func buildVaultStatus(v *vault.Vault) statusVault {
	out := statusVault{Issues: []string{}}
	if !v.Exists() {
		return out
	}
	out.Initialized = true

	status, err := v.GetStatus()
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.Versions = status.TotalVersions

	// Verify integrity
	result, err := v.VerifyIntegrity()
	switch {
	case err != nil:
		out.Integrity = "error"
		out.Error = err.Error()
	case result.Valid:
		out.Integrity = "ok"
	default:
		out.Integrity = "failed"
		out.Issues = append(out.Issues, result.Issues...)
	}

	if mode, err := v.GetParanoidMode(); err == nil {
		out.Mode = mode
	}
	return out
}

func printVaultStatus(s statusVault) {
	fmt.Println("Vault:")

	if !s.Initialized {
		fmt.Println("  Status:          Not initialized")
		return
	}

	if s.Integrity == "" {
		// GetStatus failed: nothing else was read.
		fmt.Printf("  Status:          Error: %s\n", s.Error)
		return
	}

	fmt.Printf("  Versions:        %d registered\n", s.Versions)

	switch s.Integrity {
	case "error":
		fmt.Printf("  Integrity:       ❌ Error: %s\n", s.Error)
	case "ok":
		fmt.Println("  Integrity:       ✓ OK")
	default:
		fmt.Printf("  Integrity:       ❌ %d issues\n", len(s.Issues))
	}

	if s.Mode != "" {
		fmt.Printf("  Mode:            %s %s\n", getModeIcon(s.Mode), s.Mode)
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
