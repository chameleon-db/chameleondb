package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/chameleon-db/chameleondb/chameleon/internal/admin"
	"github.com/chameleon-db/chameleondb/chameleon/internal/journal"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
)

var (
	journalLimit  int
	journalFormat string
)

var journalCmd = &cobra.Command{
	Use:   "journal <subcommand>",
	Short: "Query and audit the operation journal",
	Long: `View and search the operation journal (audit log).

The journal is an append-only log of all ChameleonDB operations.
Stored in .chameleon/journal/ with daily rotation.

Subcommands:
  journal last        Show last N operations
  journal errors      Show error operations
  journal migrations  Show migration history
  journal schema      Show schema version history (vault)
  journal search      Search journal entries`,
	Args: cobra.MinimumNArgs(1),
}

var journalLastCmd = &cobra.Command{
	Use:   "last [n]",
	Short: "Show last N journal entries",
	Long: `Display the most recent journal entries.

Examples:
  chameleon journal last        # Last 10 entries
  chameleon journal last 20     # Last 20 entries
  chameleon journal last 5 --format=json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Resolve limit from flag, optionally overridden by positional arg.
		limit := journalLimit
		if len(args) > 0 {
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid number: %s", args[0])
			}
			limit = n
		}
		if limit <= 0 {
			return fmt.Errorf("limit must be greater than 0")
		}

		// Initialize journal logger
		factory := admin.NewManagerFactory(workDir)
		logger, err := factory.CreateJournalLogger()
		if err != nil {
			return fmt.Errorf("failed to initialize journal: %w", err)
		}

		// Get last entries
		entries, err := logger.Last(limit)
		if err != nil {
			return fmt.Errorf("failed to read journal: %w", err)
		}

		if len(entries) == 0 {
			printInfo("No journal entries found")
			return nil
		}

		// Format output
		if journalFormat == "json" {
			printEntriesJSON(entries)
		} else {
			printEntriesTable(entries)
		}

		return nil
	},
}

var journalErrorsCmd = &cobra.Command{
	Use:   "errors",
	Short: "Show error journal entries",
	Long: `Display all error operations from today's journal.

Examples:
  chameleon journal errors
  chameleon journal errors --format=json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Initialize journal logger
		factory := admin.NewManagerFactory(workDir)
		logger, err := factory.CreateJournalLogger()
		if err != nil {
			return fmt.Errorf("failed to initialize journal: %w", err)
		}

		// Get error entries
		entries, err := logger.Errors()
		if err != nil {
			return fmt.Errorf("failed to read journal: %w", err)
		}

		if len(entries) == 0 {
			printSuccess("No errors found")
			return nil
		}

		// Format output
		if journalFormat == "json" {
			printEntriesJSON(entries)
		} else {
			printEntriesTable(entries)
		}

		return nil
	},
}

var journalMigrationsCmd = &cobra.Command{
	Use:   "migrations",
	Short: "Show migration history",
	Long: `Display all migration operations from the journal.

Examples:
  chameleon journal migrations
  chameleon journal migrations --format=json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Initialize journal logger
		factory := admin.NewManagerFactory(workDir)
		logger, err := factory.CreateJournalLogger()
		if err != nil {
			return fmt.Errorf("failed to initialize journal: %w", err)
		}

		// Get migration entries
		entries, err := logger.Migrations()
		if err != nil {
			return fmt.Errorf("failed to read journal: %w", err)
		}

		if len(entries) == 0 {
			printInfo("No migration entries found")
			return nil
		}

		// Format output
		if journalFormat == "json" {
			printEntriesJSON(entries)
		} else {
			printMigrationsTable(entries)
		}

		return nil
	},
}

// ========================================
// Schema Vault Journal
// ========================================

var journalSchemaCmd = &cobra.Command{
	Use:   "schema [version]",
	Short: "Show schema version history (vault)",
	Long: `View the complete version history of schemas from the Schema Vault.
	
Examples:
  chameleon journal schema          # View all versions
  chameleon journal schema v002     # View details of v002`,
	Args: cobra.MaximumNArgs(1),
	RunE: runJournalSchema,
}

func runJournalSchema(cmd *cobra.Command, args []string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	v := vault.NewVault(workDir)

	if !v.Exists() {
		fmt.Println("❌ No vault found")
		fmt.Println("   Run 'chameleon migrate' to initialize")
		return nil
	}

	if len(args) == 1 {
		// Show specific version
		showVersionDetail(v, args[0])
	} else {
		// Show all versions
		showVersionHistory(v)
	}

	return nil
}

func showVersionHistory(v *vault.Vault) {
	history, err := v.GetVersionHistory()
	if err != nil {
		fmt.Printf("❌ Failed to read history: %v\n", err)
		return
	}

	if len(history) == 0 {
		fmt.Println("📖 No schema versions yet")
		return
	}

	fmt.Println()
	fmt.Println("📖 Schema Version History")
	fmt.Println()

	// Show in reverse order (newest first)
	for i := len(history) - 1; i >= 0; i-- {
		entry := history[i]

		// Mark current version
		marker := ""
		if err := v.Load(); err == nil && v.Manifest.CurrentVersion == entry.Version {
			marker = " (current) ✓"
		}

		fmt.Println(vault.FormatVersion(&entry) + marker)
		fmt.Println()
	}
}

func showVersionDetail(v *vault.Vault, version string) {
	entry, err := v.GetVersion(version)
	if err != nil {
		fmt.Printf("❌ Version %s not found\n", version)
		return
	}

	fmt.Println()
	fmt.Printf("📋 Schema Version: %s\n", version)
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("Hash:      %s\n", entry.Hash)
	fmt.Printf("Timestamp: %s\n", entry.Timestamp.Format("2006-01-02T15:04:05Z"))
	fmt.Printf("Author:    %s\n", entry.Author)

	if entry.Parent != nil {
		fmt.Printf("Parent:    %s\n", *entry.Parent)
	} else {
		fmt.Println("Parent:    none (initial version)")
	}

	fmt.Printf("Status:    %s\n", statusString(entry.Locked))
	fmt.Println()

	if entry.ChangesSummary != "" {
		fmt.Println("📝 Changes Summary:")
		fmt.Printf("  %s\n", entry.ChangesSummary)
		fmt.Println()
	}

	if len(entry.Files) > 0 {
		fmt.Println("📂 Files:")
		for _, file := range entry.Files {
			fmt.Printf("  • %s\n", file)
		}
		fmt.Println()
	}
}

func statusString(locked bool) string {
	if locked {
		return "locked ✓"
	}
	return "unlocked"
}

func init() {
	// Add journal subcommands
	journalCmd.AddCommand(journalLastCmd)
	journalCmd.AddCommand(journalErrorsCmd)
	journalCmd.AddCommand(journalMigrationsCmd)
	journalCmd.AddCommand(journalSchemaCmd)

	// Add flags
	journalLastCmd.Flags().IntVar(&journalLimit, "limit", 10, "number of entries to show")
	journalCmd.PersistentFlags().StringVar(&journalFormat, "format", "table", "output format (table|json)")

	rootCmd.AddCommand(journalCmd)
}

// printEntriesTable prints entries in table format
func printEntriesTable(entries []*journal.Entry) {
	fmt.Println()
	fmt.Println("Timestamp                Action      Status      Details")
	fmt.Println("─────────────────────────────────────────────────────────────────")

	for _, entry := range entries {
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
		status := entry.Status
		if entry.Error != "" {
			status = "error"
		}

		details := ""
		if entry.Duration > 0 {
			details = fmt.Sprintf("duration=%dms", entry.Duration)
		}
		if entry.Error != "" {
			if details != "" {
				details += " "
			}
			details += fmt.Sprintf("error=%s", truncate(entry.Error, 50))
		}

		fmt.Printf("%-25s %-11s %-11s %s\n", timestamp, entry.Action, status, details)
	}

	fmt.Println()
}

// printMigrationsTable prints migration entries in table format
func printMigrationsTable(entries []*journal.Entry) {
	fmt.Println()
	fmt.Println("Timestamp                Version              Status    Duration")
	fmt.Println("─────────────────────────────────────────────────────────────────")

	for _, entry := range entries {
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")

		version := ""
		if v, ok := entry.Details["version"].(string); ok {
			version = v
		}

		status := entry.Status
		duration := ""
		if entry.Duration > 0 {
			duration = fmt.Sprintf("%dms", entry.Duration)
		}

		fmt.Printf("%-25s %-20s %-9s %s\n", timestamp, version, status, duration)
	}

	fmt.Println()
}

// printEntriesJSON prints entries in JSON format
func printEntriesJSON(entries []*journal.Entry) {
	type entryJSON struct {
		Timestamp  string `json:"timestamp"`
		Action     string `json:"action"`
		Status     string `json:"status"`
		DurationMS int64  `json:"duration_ms,omitempty"`
		Error      string `json:"error,omitempty"`
	}

	out := make([]entryJSON, 0, len(entries))
	for _, entry := range entries {
		item := entryJSON{
			Timestamp: entry.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			Action:    entry.Action,
			Status:    entry.Status,
		}
		if entry.Duration > 0 {
			item.DurationMS = entry.Duration
		}
		if entry.Error != "" {
			item.Error = entry.Error
		}
		out = append(out, item)
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		printError("Failed to encode journal output as JSON: %v", err)
		return
	}

	fmt.Println(string(data))
}

// truncate truncates a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
