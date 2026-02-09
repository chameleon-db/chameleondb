package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine/introspect"
	"github.com/spf13/cobra"
)

var (
	introspectOutput string
	introspectForce  bool
)

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
		connStr := args[0]

		// STEP 1: Check if schema.cam exists and validate it
		outputFile := introspectOutput
		if outputFile == "" {
			outputFile = "schema.cham"
		}

		// Validar y posiblemente cambiar el outputFile
		outputFile, err := validateAndGetOutputPath(outputFile)
		if err != nil {
			return err
		}

		printInfo("Introspecting database...")

		ctx := context.Background()

		// STEP 2: Create introspector (auto-detects DB type)
		inspector, err := introspect.NewIntrospector(ctx, connStr)
		if err != nil {
			return fmt.Errorf("failed to create introspector: %w", err)
		}
		defer inspector.Close()

		// STEP 3: Detect DB type
		detected, err := inspector.Detect(ctx)
		if err != nil {
			return fmt.Errorf("failed to detect database: %w", err)
		}
		if !detected {
			return fmt.Errorf("failed to connect or detect database type")
		}

		printSuccess("Database detected")

		// STEP 4: Introspect all tables
		printInfo("Scanning tables...")
		tables, err := inspector.GetAllTables(ctx)
		if err != nil {
			return fmt.Errorf("introspection failed: %w", err)
		}

		printSuccess(fmt.Sprintf("Found %d table(s)", len(tables)))

		// STEP 5: Generate schema
		printInfo("Generating schema...")
		schema, err := introspect.GenerateChameleonSchema(tables)
		if err != nil {
			return fmt.Errorf("schema generation failed: %w", err)
		}

		// STEP 6: Write output with safety checks
		if err := safeWriteSchema(outputFile, schema); err != nil {
			return err
		}

		printSuccess(fmt.Sprintf("Schema written to %s", outputFile))
		printInfo("\nNext steps:")
		fmt.Println("  1. Review schema and adjust relations manually")
		fmt.Println("  2. Run: chameleon validate")
		fmt.Println("  3. Use with your application")

		return nil
	},
}

// validateAndGetOutputPath valida y retorna el archivo final a usar
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

// checkExistingSchemaAndGetOutput valida schema existente y retorna el archivo final
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

// askOverwriteWithBackupAndGetOutput retorna el archivo final a usar
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

		// If new path is same as old path or empty, ask again
		if newPath == filePath || newPath == "" {
			printWarning("You entered the same file path!")
			fmt.Println("Please choose a different file or select option 1 to backup and overwrite.")
			return askOverwriteWithBackupAndGetOutput(filePath)
		}

		// If new path don't hace .cham extension, add it
		if !strings.HasSuffix(newPath, ".cham") {
			newPath += ".cham"
			printInfo(fmt.Sprintf("Auto-added .cham extension: %s", newPath))
		}

		// Validar recursivamente el nuevo path
		return validateAndGetOutputPath(newPath)

	case "3":
		return "", fmt.Errorf("introspection cancelled")

	default:
		return "", fmt.Errorf("invalid choice")
	}
}

// checkExistingSchema validates an existing schema.cham
func checkExistingSchema(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read existing schema: %w", err)
	}

	schemaContent := string(content)

	// Check 1: Is it the default generated schema from 'chameleon init'?
	if isDefaultSchema(schemaContent) {
		printWarning("Found default schema.cham from 'chameleon init'")
		return askOverwrite(filePath)
	}

	// Check 2: Is it a modified schema?
	if isModifiedSchema(schemaContent) {
		printError("Found modified schema.cham with custom entities!")
		fmt.Println()
		printWarning("⚠️  This appears to be a working schema file")
		fmt.Println()
		return askOverwriteWithBackup(filePath)
	}

	// Check 3: Is it empty or just comments?
	if isEmpty(schemaContent) {
		return askOverwrite(filePath)
	}

	return nil
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

// askOverwriteWithBackup prompts user and offers to create a backup
func askOverwriteWithBackup(filePath string) error {
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
		return err
	}

	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		// Create backup
		backupFile := filePath + ".backup"
		if err := copyFile(filePath, backupFile); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		printSuccess(fmt.Sprintf("Backup created: %s", backupFile))
		return nil

	case "2":
		fmt.Print("Enter new output file path: ")
		newPath, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		newPath = strings.TrimSpace(newPath)
		// IMPORTANTE: Actualizar la variable global
		introspectOutput = newPath
		return nil

	case "3":
		return fmt.Errorf("introspection cancelled")

	default:
		return fmt.Errorf("invalid choice")
	}
}

// copyFile creates a backup of the existing schema
func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, 0644)
}

// safeWriteSchema writes schema to file with proper error handling
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
