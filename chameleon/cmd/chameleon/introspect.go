package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/chameleon-db/chameleondb/chameleon/internal/admin"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine/introspect"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
	"github.com/spf13/cobra"
)

var (
	introspectOutput string
	introspectForce  bool
)

var envVarNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var introspectCmd = &cobra.Command{
	Use:   "introspect <database-url>",
	Short: "Generate schema from existing database",
	Long: `Introspect a database and generate a ChameleonDB schema.

Supports: PostgreSQL, MySQL (coming), SQLite (coming)

Examples:
  chameleon introspect postgresql://user:pass@localhost/mydb
  chameleon introspect postgresql://... -o schema.cham
  chameleon introspect postgresql://... --output schema.cham
  chameleon introspect postgresql://... --force  # Overwrite existing schema`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		startedAt := time.Now()
		connStr, err := resolveIntrospectConnectionString(args[0])
		if err != nil {
			return err
		}

		outputFile := "schemas/" + introspectOutput
		if outputFile == "schemas/" {
			outputFile = "schemas/schema.cham"
		}
		if !strings.HasSuffix(outputFile, ".cham") {
			outputFile += ".cham"
		}

		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		factory := admin.NewManagerFactory(workDir)
		journalLogger, err := factory.CreateJournalLogger()
		if err != nil {
			return fmt.Errorf("failed to initialize journal: %w", err)
		}

		baseDetails := map[string]interface{}{
			"output": outputFile,
			"force":  introspectForce,
		}
		_ = journalLogger.Log("introspect", "started", baseDetails, nil)

		v := vault.NewVault(workDir)
		if v.Exists() {
			mode, modeErr := v.GetParanoidMode()
			if modeErr != nil {
				_ = journalLogger.LogError("introspect", modeErr, map[string]interface{}{"action": "read_mode"})
				return fmt.Errorf("failed to read paranoid mode: %w", modeErr)
			}

			if mode == "readonly" {
				printError("Read Only Paranoid Mode is active - introspection is blocked")
				fmt.Println()
				printInfo("Mode upgrade available:")
				fmt.Println("   1. Run: chameleon config auth set-password")
				fmt.Println("   2. Run: chameleon config set mode=standard")
				fmt.Println("   3. Retry: chameleon introspect <database-url>")
				fmt.Println()
				_ = journalLogger.Log("introspect", "aborted_readonly", map[string]interface{}{
					"mode":   mode,
					"output": outputFile,
				}, nil)
				return fmt.Errorf("readonly mode: introspect is blocked")
			}

			printInfo("Paranoid Mode active: %s", mode)
			_ = journalLogger.Log("introspect", "mode_checked", map[string]interface{}{"mode": mode}, nil)
		} else {
			printWarning("Schema Vault not initialized; paranoid mode check skipped")
			_ = journalLogger.Log("introspect", "mode_check_skipped", map[string]interface{}{"reason": "vault_not_initialized"}, nil)
		}

		// Validate output path and resolve final destination.
		outputFile, err = validateAndGetOutputPath(outputFile)
		if err != nil {
			_ = journalLogger.LogError("introspect", err, map[string]interface{}{"action": "validate_output"})
			return err
		}

		printInfo("Introspecting database...")

		ctx := context.Background()

		// Create introspector using the connection scheme.
		inspector, err := introspect.NewIntrospector(ctx, connStr)
		if err != nil {
			_ = journalLogger.LogError("introspect", err, map[string]interface{}{"action": "create_introspector"})
			return fmt.Errorf("failed to create introspector: %w", err)
		}
		defer inspector.Close()

		// Verify database connectivity and engine detection.
		detected, err := inspector.Detect(ctx)
		if err != nil {
			_ = journalLogger.LogError("introspect", err, map[string]interface{}{"action": "detect_database"})
			return fmt.Errorf("failed to detect database: %w", err)
		}
		if !detected {
			detectErr := fmt.Errorf("failed to connect or detect database type")
			_ = journalLogger.LogError("introspect", detectErr, map[string]interface{}{"action": "detect_database"})
			return detectErr
		}

		printSuccess("Database detected")
		_ = journalLogger.Log("introspect", "database_detected", nil, nil)

		// Introspect all user tables.
		printInfo("Scanning tables...")
		tables, err := inspector.GetAllTables(ctx)
		if err != nil {
			_ = journalLogger.LogError("introspect", err, map[string]interface{}{"action": "scan_tables"})
			return fmt.Errorf("introspection failed: %w", err)
		}

		printSuccess(fmt.Sprintf("Found %d table(s)", len(tables)))
		_ = journalLogger.Log("introspect", "tables_scanned", map[string]interface{}{"tables": len(tables)}, nil)

		// Generate schema output.
		printInfo("Generating schema...")
		schema, err := introspect.GenerateChameleonSchema(tables)
		if err != nil {
			_ = journalLogger.LogError("introspect", err, map[string]interface{}{"action": "generate_schema"})
			return fmt.Errorf("schema generation failed: %w", err)
		}

		// Write schema output with overwrite safety checks.
		if err := safeWriteSchema(outputFile, schema); err != nil {
			_ = journalLogger.LogError("introspect", err, map[string]interface{}{"action": "write_schema", "output": outputFile})
			return err
		}

		durationMs := time.Since(startedAt).Milliseconds()
		_ = journalLogger.Log("introspect", "completed", map[string]interface{}{
			"output":      outputFile,
			"tables":      len(tables),
			"duration_ms": durationMs,
		}, nil)

		printSuccess(fmt.Sprintf("Schema written to %s", outputFile))
		printInfo("\nNext steps:")
		fmt.Println("  1. Review schema and adjust relations manually")
		fmt.Println("  2. Run: chameleon validate")
		fmt.Println("  3. Use with your application")

		return nil
	},
}

func resolveIntrospectConnectionString(input string) (string, error) {
	connStr := strings.TrimSpace(input)
	if connStr == "" {
		return "", fmt.Errorf("database-url cannot be empty")
	}

	if strings.HasPrefix(connStr, "env:") {
		envName := strings.TrimSpace(strings.TrimPrefix(connStr, "env:"))
		return resolveConnectionStringFromEnv(envName)
	}

	if strings.HasPrefix(connStr, "${") && strings.HasSuffix(connStr, "}") {
		envName := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(connStr, "${"), "}"))
		return resolveConnectionStringFromEnv(envName)
	}

	if strings.HasPrefix(connStr, "$") {
		envName := strings.TrimSpace(strings.TrimPrefix(connStr, "$"))
		return resolveConnectionStringFromEnv(envName)
	}

	return connStr, nil
}

func resolveConnectionStringFromEnv(envName string) (string, error) {
	if !envVarNamePattern.MatchString(envName) {
		return "", fmt.Errorf("invalid environment variable name %q", envName)
	}

	value := strings.TrimSpace(os.Getenv(envName))
	if value == "" {
		return "", fmt.Errorf("environment variable %s is not set or empty", envName)
	}

	return value, nil
}

// validateAndGetOutputPath validates and returns the final output path.
func validateAndGetOutputPath(outputFile string) (string, error) {
	// If --force is set, skip all checks
	if introspectForce {
		return outputFile, nil
	}

	// Check if file exists
	info, err := os.Stat(outputFile)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to check output file: %w", err)
	}

	// File doesn't exist - safe to create
	if os.IsNotExist(err) {
		return outputFile, nil
	}

	// File is a directory
	if info.IsDir() {
		return "", fmt.Errorf("output path is a directory: %s", outputFile)
	}

	// File exists - need to validate if it's safe to overwrite
	return checkExistingSchemaAndGetOutput(outputFile)
}

// checkExistingSchemaAndGetOutput validates existing schema content and returns the final output path.
func checkExistingSchemaAndGetOutput(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read existing schema: %w", err)
	}

	schemaContent := string(content)

	// Check 1: Is it the default generated schema from 'chameleon init'?
	if isDefaultSchema(schemaContent) {
		printWarning("Found default schema.cham from 'chameleon init'")
		if err := askOverwrite(filePath); err != nil {
			return "", err
		}
		return filePath, nil
	}

	// Check 2: Is it a modified schema?
	if isModifiedSchema(schemaContent) {
		printError("Found modified schema.cham with custom entities!")
		fmt.Println()
		printWarning("⚠️  This appears to be a working schema file")
		fmt.Println()
		return askOverwriteWithBackupAndGetOutput(filePath)
	}

	// Check 3: Is it empty or just comments?
	if isEmpty(schemaContent) {
		if err := askOverwrite(filePath); err != nil {
			return "", err
		}
		return filePath, nil
	}

	return filePath, nil
}

// askOverwriteWithBackupAndGetOutput prompts for overwrite strategy and returns the selected output path.
func askOverwriteWithBackupAndGetOutput(filePath string) (string, error) {
	fmt.Println()
	printError(fmt.Sprintf("Existing schema detected: %s", filePath))
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  1. Create backup and overwrite (recommended)")
	fmt.Println("  2. Use different output file")
	fmt.Println("  3. Cancel")
	fmt.Println()
	fmt.Print("Choose option (1-3): ")

	reader := bufio.NewReader(os.Stdin)
	choice, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		// Create backup
		backupFile := filePath + ".backup"
		if err := copyFile(filePath, backupFile); err != nil {
			return "", fmt.Errorf("failed to create backup: %w", err)
		}
		printSuccess(fmt.Sprintf("Backup created: %s", backupFile))
		return filePath, nil

	case "2":
		fmt.Print("Enter new output file path: ")
		newPath, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		newPath = strings.TrimSpace(newPath)

		// If the new path is unchanged or empty, ask again.
		if newPath == filePath || newPath == "" {
			printWarning("You entered the same file path!")
			fmt.Println("Please choose a different file or select option 1 to backup and overwrite.")
			return askOverwriteWithBackupAndGetOutput(filePath)
		}

		// Add .cham extension when missing.
		if !strings.HasSuffix(newPath, ".cham") {
			newPath += ".cham"
			printInfo(fmt.Sprintf("Auto-added .cham extension: %s", newPath))
		}

		// Put new path inside schemas/ directory if not already there.
		if !strings.HasPrefix(newPath, "schemas/") {
			newPath = "schemas/" + newPath
			printInfo(fmt.Sprintf("Auto-prefixed with schemas/: %s", newPath))
		}

		// Re-validate the new destination.
		return validateAndGetOutputPath(newPath)

	case "3":
		return "", fmt.Errorf("introspection cancelled")

	default:
		return "", fmt.Errorf("invalid choice")
	}
}

// isDefaultSchema checks if schema matches the default template from 'init'
func isDefaultSchema(content string) bool {
	// Verify if match default schema patterns
	hasDefaultUser := strings.Contains(content, "entity User {")
	hasDefaultPost := strings.Contains(content, "entity Post {")
	hasDefaultComment := strings.Contains(content, "entity Comment {")
	hasViaClause := strings.Contains(content, " via ")
	hasDefaultRelations := strings.Contains(content, "posts: [Post]") ||
		strings.Contains(content, "comments: [Comment]")

	return hasDefaultUser && hasDefaultPost && hasDefaultComment &&
		hasViaClause && hasDefaultRelations
}

// isModifiedSchema checks if schema has been modified
func isModifiedSchema(content string) bool {
	if isDefaultSchema(content) {
		return false
	}

	lines := strings.Split(content, "\n")
	entityCount := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "entity ") {
			entityCount++
		}
	}

	return entityCount > 0
}

// isEmpty checks if schema is empty or only comments
func isEmpty(content string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			return false
		}
	}
	return true
}

// askOverwrite prompts user before overwriting
func askOverwrite(filePath string) error {
	fmt.Println()
	printWarning(fmt.Sprintf("File exists: %s", filePath))
	fmt.Print("Overwrite? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" && response != "y" {
		return fmt.Errorf("introspection cancelled")
	}

	return nil
}

// copyFile creates a backup copy of an existing schema file.
func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, 0644)
}

// safeWriteSchema writes schema content to disk.
func safeWriteSchema(filePath string, schema string) error {
	if err := os.WriteFile(filePath, []byte(schema), 0644); err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}
	return nil
}

func init() {
	introspectCmd.Flags().StringVarP(
		&introspectOutput, "output", "o", "schema.cham",
		"Output file for generated schema",
	)
	introspectCmd.Flags().BoolVarP(
		&introspectForce, "force", "f", false,
		"Force overwrite without confirmation (use with caution!)",
	)
	rootCmd.AddCommand(introspectCmd)
}
