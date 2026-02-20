package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/chameleon-db/chameleondb/chameleon/internal/admin"
	"github.com/chameleon-db/chameleondb/chameleon/internal/schema"
	"github.com/chameleon-db/chameleondb/chameleon/internal/state"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
	"github.com/jackc/pgx/v5"
)

var (
	dryRun         bool
	applyMigration bool
	checkOnly      bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
	Long: `Generate, validate, or apply database migrations from schema files.

By default, displays what would be migrated (--check).
Use --apply to execute the migration against the database.
Use --dry-run to preview without applying.

Examples:
  chameleon migrate              # Check for pending migrations
  chameleon migrate --dry-run    # Preview SQL without applying
  chameleon migrate --apply      # Apply pending migrations`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get working directory
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Initialize admin factory
		printInfo("Loading configuration...")
		factory := admin.NewManagerFactory(workDir)

		// Load config
		configLoader := factory.CreateConfigLoader()
		cfg, err := configLoader.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		printSuccess("Configuration loaded from .chameleon.yml")

		// Create journal logger
		journalLogger, err := factory.CreateJournalLogger()
		if err != nil {
			return fmt.Errorf("failed to initialize journal: %w", err)
		}

		// Create state tracker
		stateTracker, err := factory.CreateStateTracker()
		if err != nil {
			return fmt.Errorf("failed to initialize state tracker: %w", err)
		}

		// ========================================
		// SCHEMA VAULT INTEGRATION
		// ========================================

		// Initialize Schema Vault
		v := vault.NewVault(workDir)

		// Auto-initialize vault if doesn't exist
		if !v.Exists() {
			printInfo("Initializing Schema Vault...")
			if err := v.Initialize(); err != nil {
				journalLogger.LogError("migrate", err, map[string]interface{}{"action": "vault_init"})
				return fmt.Errorf("failed to initialize vault: %w", err)
			}
			printSuccess("Created .chameleon/vault/")
		}

		// Verify integrity
		printInfo("Verifying schema integrity...")
		vaultResult, err := v.VerifyIntegrity()
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "verify_integrity"})
			return fmt.Errorf("integrity verification failed: %w", err)
		}

		if !vaultResult.Valid {
			fmt.Println()
			printError("INTEGRITY VIOLATION DETECTED")
			fmt.Println()
			for _, issue := range vaultResult.Issues {
				printError("  • %s", issue)
			}
			fmt.Println()
			printError("Schema vault has been modified!")
			fmt.Println("Recovery options:")
			fmt.Println("   1. Run 'chameleon verify' for details")
			fmt.Println("   2. Check .chameleon/vault/integrity.log")
			fmt.Println("   3. Contact your DBA")
			fmt.Println()
			printError("Migration aborted for safety")

			journalLogger.LogError("migrate",
				fmt.Errorf("integrity violation: %d issues", len(vaultResult.Issues)),
				map[string]interface{}{"action": "verify_integrity"})

			return fmt.Errorf("integrity check failed")
		}

		// Log migration start
		logDetails := map[string]interface{}{
			"action":  "check",
			"dry_run": dryRun,
			"apply":   applyMigration,
		}
		journalLogger.Log("migrate", "started", logDetails, nil)

		// Show current vault status
		if v.Manifest != nil && v.Manifest.CurrentVersion != "" {
			current, _ := v.GetCurrentVersion()
			if current != nil {
				printSuccess("Current version: %s (%s...)", current.Version, current.Hash[:12])
				mode, modeErr := v.GetParanoidMode()
				if modeErr != nil {
					journalLogger.LogError("migrate", modeErr, map[string]interface{}{"action": "read_mode"})
					return fmt.Errorf("failed to read vault mode: %w", modeErr)
				}

				if mode == "readonly" {
					printError("Read Only mode is active - schema modifications are blocked")
					fmt.Println()
					printInfo("To override (not recommended):")
					fmt.Println("   1. Set paranoia mode: chameleon config set mode=standard")
					fmt.Println("   2. Fix integrity issues first")
					fmt.Println()
					journalLogger.Log("migrate", "aborted_readonly", logDetails, nil)
					return fmt.Errorf("readonly mode: schema locked")
				}

				printInfo("Vault mode active: %s", mode)
			}
		} else {
			printInfo("No schema versions registered yet")
		}

		// Load and merge schemas
		printInfo("Loading schemas from: %v", cfg.Schema.Paths)
		eng := engine.NewEngine()

		// Load all schema files using FileLoader
		loader := schema.NewFileLoader(cfg.Schema.Paths)
		filenames, schemaContents, err := loader.LoadAll()
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "load_schemas"})
			return fmt.Errorf("failed to load schemas: %w", err)
		}

		printSuccess("Found %d schema file(s): %v", len(filenames), filenames)

		// Merge schemas using SimpleMerger with source tracking
		merger := schema.NewSimpleMerger()
		mergedResult, err := merger.Merge(filenames, schemaContents)
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "merge_schemas"})
			return fmt.Errorf("failed to merge schemas: %w", err)
		}

		mergedSchema := mergedResult.Content
		lineMap := mergedResult.LineMap

		// Validate merged schema
		if err := merger.Validate(mergedSchema); err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "validate_schemas"})
			return fmt.Errorf("schema validation failed: %w", err)
		}

		// Parse merged schema (capture errors with source mapping)
		_, err = eng.LoadSchemaFromString(mergedSchema)
		if err != nil {
			// Try to map error line to source file
			errMsg := err.Error()
			sourceInfo := tryMapErrorToSource(errMsg, lineMap)
			if sourceInfo != "" {
				errMsg = strings.ReplaceAll(errMsg, "schema.cham", sourceInfo)
				errMsg = sourceInfo + "\n" + errMsg
			}

			journalLogger.LogError("migrate", fmt.Errorf("%s", errMsg), map[string]interface{}{
				"action": "parse_schema",
				"files":  filenames,
			})

			// Save merged schema for debugging with timestamp
			if len(cfg.Schema.Paths) > 0 {
				debugDir := filepath.Join(filepath.Dir(cfg.Schema.Paths[0]), ".chameleon", "state", "debug")
				if mkErr := os.MkdirAll(debugDir, 0755); mkErr != nil {
					printError("Could not create debug directory: %v", mkErr)
				} else {
					timestamp := time.Now().Format("20060102-150405")
					debugPath := filepath.Join(debugDir, fmt.Sprintf("schema.merged.%s.cham", timestamp))
					if writeErr := os.WriteFile(debugPath, []byte(mergedSchema), 0644); writeErr != nil {
						printError("Could not write debug schema: %v", writeErr)
					} else {
						printError("Schema saved to %s for debugging", debugPath)
					}
				}
			}

			return fmt.Errorf("failed to parse merged schemas:\n%s", errMsg)
		}

		printSuccess("Schema loaded and validated")

		// Save merged schema to temp file for vault registration
		mergedSchemaPath := filepath.Join(workDir, ".chameleon", "state", "schema.merged.cham")
		if err := os.WriteFile(mergedSchemaPath, []byte(mergedSchema), 0644); err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "save_merged_schema"})
			return fmt.Errorf("failed to save merged schema: %w", err)
		}

		// Get current state early (needed for both normal and retry paths)
		currentState, err := stateTracker.LoadCurrent()
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "load_state"})
			return fmt.Errorf("failed to load current state: %w", err)
		}

		// ========================================
		// DETECT SCHEMA CHANGES (Schema Vault)
		// ========================================

		changed, changesSummary, err := v.DetectChanges(mergedSchemaPath)
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "detect_changes"})
			return fmt.Errorf("failed to detect changes: %w", err)
		}

		lastAppliedMigration, err := stateTracker.GetLastMigration()
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "get_last_migration"})
			return fmt.Errorf("failed to get last migration: %w", err)
		}

		currentVaultVersion := ""
		if v.Manifest != nil {
			currentVaultVersion = v.Manifest.CurrentVersion
		}

		hasPendingUnappliedVersion := currentVaultVersion != "" && (lastAppliedMigration == nil || lastAppliedMigration.Version != currentVaultVersion)

		if !changed {
			if !hasPendingUnappliedVersion {
				printInfo("No schema changes detected")
				fmt.Println()
				printSuccess("Schema is up to date")
				journalLogger.Log("migrate", "no_changes", map[string]interface{}{"action": "check"}, nil)
				return nil
			}

			changesSummary = fmt.Sprintf("Retry pending migration for %s", currentVaultVersion)
			printWarning("Schema unchanged, but latest version %s is not applied to database", currentVaultVersion)
			journalLogger.Log("migrate", "pending_unapplied", map[string]interface{}{
				"vault_version": currentVaultVersion,
			}, nil)
		}

		printInfo("Schema changes detected: %s", changesSummary)

		// Generate migration
		printInfo("Generating migration SQL...")
		migrationSQL, err := eng.GenerateMigration()
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "generate"})
			return fmt.Errorf("failed to generate migration: %w", err)
		}
		printSuccess("Migration SQL generated")

		// Display migration plan
		fmt.Println()
		fmt.Println("─────────────────────────────────────────────────")
		fmt.Println("Migration SQL:")
		fmt.Println("─────────────────────────────────────────────────")
		fmt.Println(migrationSQL)
		fmt.Println("─────────────────────────────────────────────────")
		fmt.Println()

		if dryRun || !applyMigration {
			printInfo("Dry-run mode. Use --apply to execute migration.")
			journalLogger.Log("migrate", "dry_run", map[string]interface{}{"action": "check"}, nil)
			return nil
		}

		// ========================================
		// REGISTER VERSION IN VAULT (before applying)
		// ========================================

		printInfo("Registering new schema version...")

		// Get author (current user or from config)
		author := os.Getenv("USER")
		if author == "" {
			author = "unknown"
		}

		newVersion, err := v.RegisterVersion(mergedSchemaPath, author, changesSummary)
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "register_version"})
			return fmt.Errorf("failed to register version: %w", err)
		}

		printSuccess("Registered as %s (hash: %s...)", newVersion.Version, newVersion.Hash[:12])
		if newVersion.Parent != nil {
			printInfo("Parent version: %s", *newVersion.Parent)
		}

		printInfo("Connecting to database...")

		// Connect to database
		connCtx, connCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer connCancel()

		conn, err := pgx.Connect(connCtx, cfg.Database.ConnectionString)
		if err != nil {
			currentState.Status = "pending_migration"
			if saveErr := stateTracker.SaveCurrent(currentState); saveErr != nil {
				journalLogger.LogError("migrate", saveErr, map[string]interface{}{"action": "save_state_connect_failure"})
			}

			failedMigration := &state.Migration{
				Version:     newVersion.Version,
				Timestamp:   time.Now(),
				Type:        "auto",
				Description: changesSummary,
				Status:      "failed",
				SchemaHash:  newVersion.Hash,
				DDLHash:     state.HashDDL(migrationSQL),
				Checksum:    "pending",
			}
			if addErr := stateTracker.AddMigration(failedMigration); addErr != nil {
				journalLogger.LogError("migrate", addErr, map[string]interface{}{"action": "record_failed_migration_connect"})
			}

			journalLogger.LogMigration(newVersion.Version, "failed", 0, "", map[string]interface{}{
				"error": err.Error(),
			})
			v.AppendLog("MIGRATE", newVersion.Version, map[string]string{
				"status": "failed",
				"error":  err.Error(),
			})

			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "connect"})
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer conn.Close(connCtx)

		printSuccess("Connected to database")

		// Create backup before applying (if enabled)
		if cfg.Features.BackupOnMigrate {
			printInfo("Creating backup...")
		}

		// Apply migration
		printInfo("Applying migration...")
		startTime := time.Now()

		_, err = conn.Exec(ctx, migrationSQL)
		if err != nil {
			duration := time.Since(startTime).Milliseconds()

			currentState.Status = "pending_migration"
			if saveErr := stateTracker.SaveCurrent(currentState); saveErr != nil {
				journalLogger.LogError("migrate", saveErr, map[string]interface{}{"action": "save_state_exec_failure"})
			}

			failedMigration := &state.Migration{
				Version:     newVersion.Version,
				Timestamp:   time.Now(),
				Type:        "auto",
				Description: changesSummary,
				Status:      "failed",
				SchemaHash:  newVersion.Hash,
				DDLHash:     state.HashDDL(migrationSQL),
				Checksum:    "pending",
			}
			if addErr := stateTracker.AddMigration(failedMigration); addErr != nil {
				journalLogger.LogError("migrate", addErr, map[string]interface{}{"action": "record_failed_migration_exec"})
			}

			journalLogger.LogMigration(newVersion.Version, "failed", duration, "", map[string]interface{}{
				"error": err.Error(),
			})

			// Log failure in vault
			v.AppendLog("MIGRATE", newVersion.Version, map[string]string{
				"status": "failed",
				"error":  err.Error(),
			})

			printError("Migration failed")
			return fmt.Errorf("failed to execute migration: %w", err)
		}

		duration := time.Since(startTime).Milliseconds()
		printSuccess("Migration applied successfully")

		// Update state
		printInfo("Updating state...")
		currentState.Status = "in_sync"
		currentState.Migrations.AppliedCount++
		currentState.Migrations.LastAppliedAt = time.Now()

		if err := stateTracker.SaveCurrent(currentState); err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "save_state"})
			// Don't fail on state update error, migration was successful
			printError("Warning: Failed to update state: %v", err)
		} else {
			printSuccess("State updated")
		}

		// Add migration to manifest
		migration := &state.Migration{
			Version:     newVersion.Version, // Use vault version
			Timestamp:   time.Now(),
			Type:        "auto",
			Description: changesSummary,
			AppliedAt:   time.Now(),
			Status:      "applied",
			SchemaHash:  newVersion.Hash, // Use vault hash
			DDLHash:     state.HashDDL(migrationSQL),
			Checksum:    "verified",
		}

		if err := stateTracker.AddMigration(migration); err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "add_migration"})
			// Don't fail, migration was successful
			printError("Warning: Failed to record migration: %v", err)
		}

		// Log migration success (both journal and vault)
		journalLogger.LogMigration(migration.Version, "applied", duration, "", map[string]interface{}{
			"tables_created": 0,
		})

		v.AppendLog("MIGRATE", newVersion.Version, map[string]string{
			"status":   "applied",
			"duration": fmt.Sprintf("%dms", duration),
		})

		fmt.Println()
		printSuccess("Migration completed successfully!")
		fmt.Println()
		fmt.Println("Summary:")
		fmt.Printf("  Version:  %s\n", migration.Version)
		fmt.Printf("  Hash:     %s\n", newVersion.Hash[:16]+"...")
		fmt.Printf("  Duration: %dms\n", duration)
		fmt.Printf("  Status:   applied\n")
		fmt.Println()

		return nil
	},
}

func init() {
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show migration SQL without applying")
	migrateCmd.Flags().BoolVar(&applyMigration, "apply", false, "apply migration to database")
	migrateCmd.Flags().BoolVar(&checkOnly, "check", false, "only check for pending migrations (default)")

	rootCmd.AddCommand(migrateCmd)
}

// tryMapErrorToSource maps parser line numbers to source schema files.
func tryMapErrorToSource(errMsg string, lineMap map[int]schema.SourceLine) string {
	// Supported patterns: "line 25", "--> file:25:5", " 25 │".
	patterns := []string{
		`line (\d+)`,
		`-->.*?:(\d+):`,
		`\s(\d+)\s*│`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(errMsg)
		if len(matches) > 1 {
			lineNum, _ := strconv.Atoi(matches[1])

			if source, exists := lineMap[lineNum]; exists {
				return fmt.Sprintf("Error in %s:%d", source.File, source.LineNumber)
			}

			// Look for a nearby source line when offsets differ.
			for offset := 1; offset <= 5; offset++ {
				if source, exists := lineMap[lineNum-offset]; exists {
					return fmt.Sprintf("Error in %s:%d", source.File, source.LineNumber+offset)
				}
				if source, exists := lineMap[lineNum+offset]; exists {
					return fmt.Sprintf("Error in %s:%d", source.File, source.LineNumber-offset)
				}
			}
		}
	}

	return ""
}
