package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/chameleon-db/chameleondb/chameleon/internal/admin"
	"github.com/chameleon-db/chameleondb/chameleon/internal/config"
	"github.com/chameleon-db/chameleondb/chameleon/internal/initwizard"
)

var (
	initDBURL     string
	initCompose   bool
	initDetectDB  bool
	initNoExample bool
	initNoInput   bool
)

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new ChameleonDB project",
	Long: `Create a new ChameleonDB project with schema and configuration.

This will create:
  .chameleon.yml        Main configuration file
  .chameleon/           Admin directory (config, state, journal, backups)
  schemas/              Directory for schema files
  README.md             Getting started guide

If no name is provided, initializes in current directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var workDir string

		// Determine working directory
		if len(args) > 0 {
			projectName := args[0]
			workDir = projectName

			// Check if already initialized
			configPath := filepath.Join(workDir, ".chameleon.yml")
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("project already initialized at %s\nRun 'chameleon migrate' to manage migrations or delete .chameleon.yml to reinitialize", workDir)
			}

			// Create project directory
			if err := os.MkdirAll(workDir, 0755); err != nil {
				return fmt.Errorf("failed to create project directory: %w", err)
			}
			printInfo("Creating new project in: %s", workDir)
		} else {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			// Check if already initialized
			configPath := filepath.Join(workDir, ".chameleon.yml")
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("ChameleonDB already initialized in current directory\nRun 'chameleon migrate' to manage migrations or delete .chameleon.yml to reinitialize")
			}

			printInfo("Initializing ChameleonDB in current directory: %s", workDir)
		}

		choices, err := initwizard.Resolve(
			initwizard.NewPrompter(os.Stdin, os.Stdout),
			initwizard.Flags{DBURL: initDBURL, Compose: initCompose, DetectDB: initDetectDB, NoExample: initNoExample},
			initwizard.Env{Dir: workDir, Getenv: os.Getenv, Interactive: interactiveTerminal(initNoInput), Ping: pingPostgres},
		)
		if err != nil {
			return err
		}

		// Initialize admin structure (.chameleon/)
		printInfo("Creating .chameleon/ structure...")
		factory := admin.NewManagerFactory(workDir)
		if err := factory.Initialize(); err != nil {
			return fmt.Errorf("failed to create admin structure: %w", err)
		}
		printSuccess("Created .chameleon/ directory")

		// Create .chameleon.yml
		printInfo("Creating .chameleon.yml...")
		cfg := config.Defaults()
		cfg.CreatedAt = time.Now()

		// Set paths relative to workDir if it's a new project
		if len(args) > 0 {
			cfg.Schema.Paths = []string{"./schemas"}
			cfg.Schema.MergedOutput = ".chameleon/state/schema.merged.cham"
		}

		loader := factory.CreateConfigLoader()
		if err := loader.Save(cfg); err != nil {
			return fmt.Errorf("failed to create .chameleon.yml: %w", err)
		}
		printSuccess("Created .chameleon.yml")

		// Create schemas directory
		printInfo("Creating schemas/ directory...")
		schemasDir := filepath.Join(workDir, "schemas")
		if err := os.MkdirAll(schemasDir, 0755); err != nil {
			return fmt.Errorf("failed to create schemas directory: %w", err)
		}

		// Create example schema
		if choices.CreateExample {
			schemaPath := filepath.Join(schemasDir, "example.cham")
			schemaContent := exampleSchema()
			if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
				return fmt.Errorf("failed to create example schema: %w", err)
			}
			printSuccess("Created schemas/example.cham")
		}

		if choices.WriteCompose {
			created, err := initwizard.WriteComposeFile(workDir)
			if err != nil {
				return err
			}
			if created {
				printSuccess("Created %s (PostgreSQL 16)", initwizard.ComposeFileName)
			} else {
				printWarning("A compose file already exists; it was not changed")
			}

			if choices.RunCompose {
				printInfo("Starting PostgreSQL (docker compose up -d)...")
				if err := runDockerComposeUp(workDir); err != nil {
					printWarning("docker compose failed: %v", err)
				} else if err := waitForPostgres(choices.URL, 20); err != nil {
					printWarning("%v", err)
				} else {
					printSuccess("PostgreSQL is accepting connections")
				}
			}
		}
		if choices.URL != "" {
			if created, err := initwizard.WriteEnvExample(workDir, choices.URL); err != nil {
				return err
			} else if created {
				printSuccess("Created .env.example (placeholder password)")
			}
		}

		// Create README
		printInfo("Creating README.md...")
		readmePath := filepath.Join(workDir, "README.md")
		projectName := filepath.Base(workDir)
		readmeContent := exampleReadme(projectName)
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			return fmt.Errorf("failed to create README.md: %w", err)
		}
		printSuccess("Created README.md")

		// Print status
		fmt.Println()
		printSuccess("Project initialized successfully!")
		fmt.Println()
		fmt.Println("Structure created:")
		fmt.Println("  .chameleon.yml        Configuration (edit this)")
		fmt.Println("  .chameleon/           Admin directory (auto-managed)")
		fmt.Println("    ├── config.yml      Configuration source")
		fmt.Println("    ├── state/          Local state (not versioned)")
		fmt.Println("    ├── journal/        Audit logs (not versioned)")
		fmt.Println("    └── backups/        Backup files (not versioned)")
		fmt.Println("  schemas/              Schema files")
		if choices.CreateExample {
			fmt.Println("    └── example.cham    Example schema")
		}
		fmt.Println("  README.md             Getting started guide")
		fmt.Println()

		fmt.Println("Next steps:")
		if len(args) > 0 {
			fmt.Printf("  cd %s\n", projectName)
		}
		if choices.URL != "" {
			fmt.Printf("  export DATABASE_URL=\"%s\"\n", choices.URL)
		} else {
			fmt.Println("  export DATABASE_URL=\"postgresql://user:password@localhost/dbname\"")
		}
		fmt.Println("  chameleon migrate --dry-run")
		fmt.Println("  chameleon migrate --apply")
		fmt.Println()
		for _, note := range choices.Notes {
			printWarning("%s", note)
		}
		if len(choices.Notes) > 0 {
			fmt.Println()
		}
		fmt.Println("About the mode:")
		fmt.Println("  The first 'chameleon migrate --apply' leaves the project in readonly mode,")
		fmt.Println("  which guards against accidental schema changes. You will be asked there")
		fmt.Println("  whether to set a password and switch to standard mode.")
		fmt.Println()
		fmt.Println("Edit schemas/*.cham to define your database schema.")
		fmt.Println("Configuration: .chameleon.yml (version controlled)")
		fmt.Println("State/Journal: .chameleon/ (local, not versioned)")
		fmt.Println()

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initDBURL, "db-url", "", "PostgreSQL connection string to use (printed as an export line; the config keeps ${DATABASE_URL})")
	initCmd.Flags().BoolVar(&initCompose, "compose", false, "write a docker-compose.yml with PostgreSQL 16")
	initCmd.Flags().BoolVar(&initDetectDB, "detect-db", false, "use the first PostgreSQL connection found in the environment, .env files or compose files")
	initCmd.Flags().BoolVar(&initNoExample, "no-example", false, "do not create schemas/example.cham")
	initCmd.Flags().BoolVar(&initNoInput, "no-input", false, "never ask questions (also the case without a terminal or with CI set)")
	rootCmd.AddCommand(initCmd)
}

func exampleSchema() string {
	return `// ChameleonDB Example Schema
// This is a simple blog schema to get you started

entity User {
    id: uuid primary default uuid(),
    email: string unique,
    name: string,
    created_at: timestamp default now(),
}

entity Post {
    id: uuid primary default uuid(),
    title: string,
    content: string,
    published: bool,
    author_id: uuid,
    created_at: timestamp default now(),
}

entity Comment {
    id: uuid primary default uuid(),
    content: string,
    post_id: uuid,
    user_id: uuid,
    created_at: timestamp default now(),
}
`
}

func exampleReadme(projectName string) string {
	return `# ` + projectName + `

ChameleonDB project initialized with ` + "`chameleon init`" + `.

## Quick Start

### 1. Set up database connection

` + "```bash" + `
export DATABASE_URL="postgresql://user:password@localhost/dbname"
` + "```" + `

### 2. Validate your schema

` + "```bash" + `
chameleon migrate --check
` + "```" + `

### 3. Preview migration (dry-run)

` + "```bash" + `
chameleon migrate --dry-run
` + "```" + `

### 4. Apply migration to database

` + "```bash" + `
chameleon migrate --apply
` + "```" + `

## Project Structure

` + "```" + `
.
├── .chameleon.yml          Configuration (version controlled)
├── .chameleon/             Admin directory (local, not versioned)
│   ├── config.yml          Source config
│   ├── state/              Current DB state
│   ├── journal/            Audit logs
│   └── backups/            Migration backups
├── schemas/                Schema files
│   └── example.cham        Example schema
└── README.md               This file
` + "```" + `

## Configuration

Edit ` + "`.chameleon.yml`" + ` to:
- Review the database connection (PostgreSQL)
- Set connection string or use ` + "`${DATABASE_URL}`" + ` env var
- Configure features (auto_migration, dry_run_default)
- Set safety options (validation, confirmation)

## Schema

Define your database schema in ` + "`schemas/*.cham`" + `:

` + "```" + `
entity User {
    id: uuid primary default uuid(),
    email: string unique,
    name: string,
    created_at: timestamp default now(),
}

entity Post {
    id: uuid primary default uuid(),
    title: string,
    author_id: uuid,
    created_at: timestamp default now(),
}
` + "```" + `

Run ` + "`chameleon migrate --check`" + ` to validate.

## Migrations

Migrations are tracked in ` + "`.chameleon/state/migrations/manifest.json`" + `.

Each migration:
- Has a unique version (timestamp-based)
- Includes schema hash for integrity
- Supports rollback (planned v0.2)
- Is backed up before applying

View history:

` + "```bash" + `
chameleon journal migrations
` + "```" + `

## Development

### Validate schema

` + "```bash" + `
chameleon migrate --check
` + "```" + `

### View audit log

` + "```bash" + `
chameleon journal last 10
` + "```" + `

### See migration history

` + "```bash" + `
chameleon journal migrations
` + "```" + `

## Learn More

- [ChameleonDB Documentation](https://docs.chameleondb.dev)
- [Schema Reference](https://docs.chameleondb.dev/schema/entities)
- [Query API](https://docs.chameleondb.dev/queries/overview)
`
}
