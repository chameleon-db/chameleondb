package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Initialize a new ChameleonDB project",
	Long: `Create a new ChameleonDB project with an example schema.

This will create:
  <name>/
    schema.cham       Example schema
    .chameleon        Configuration file
    README.md         Getting started guide`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]

		printInfo("Creating new project: %s", projectName)

		// Create project directory
		if err := os.MkdirAll(projectName, 0755); err != nil {
			return fmt.Errorf("failed to create project directory: %w", err)
		}

		// Create schema.cham
		schemaPath := filepath.Join(projectName, "schema.cham")
		schemaContent := exampleSchema()
		if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
			return fmt.Errorf("failed to create schema.cham: %w", err)
		}
		printSuccess("Created schema.cham")

		// Create .chameleon config
		configPath := filepath.Join(projectName, ".chameleon")
		configContent := exampleConfig()
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to create .chameleon: %w", err)
		}
		printSuccess("Created .chameleon")

		// Create README
		readmePath := filepath.Join(projectName, "README.md")
		readmeContent := exampleReadme(projectName)
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			return fmt.Errorf("failed to create README.md: %w", err)
		}
		printSuccess("Created README.md")

		fmt.Println()
		printSuccess("Project initialized successfully!")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Printf("  cd %s\n", projectName)
		fmt.Println("  chameleon validate")
		fmt.Println("  chameleon migrate --dry-run")
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func exampleSchema() string {
	return `// ChameleonDB Example Schema
// This is a simple blog schema to get you started

entity User {
    id: uuid primary,
    email: string unique,
    name: string,
    created_at: timestamp default now(),
    posts: [Post] via author_id,
}

entity Post {
    id: uuid primary,
    title: string,
    content: string,
    published: bool,
    created_at: timestamp default now(),
    author_id: uuid,
    author: User,
    comments: [Comment] via post_id,
}

entity Comment {
    id: uuid primary,
    content: string,
    created_at: timestamp default now(),
    post_id: uuid,
    post: Post,
}
`
}

func exampleConfig() string {
	return `# ChameleonDB Configuration

[database]
host = "localhost"
port = 5432
database = "myapp"
user = "postgres"
password = ""
max_conns = 5
min_conns = 1

[migration]
# Directory for migration files
dir = "migrations"
`
}

func exampleReadme(projectName string) string {
	return "# " + projectName + "\n\n" +
		"ChameleonDB project created with `chameleon init`.\n\n" +
		"## Quick Start\n\n" +
		"### 1. Validate your schema\n" +
		"```bash\n" +
		"chameleon validate\n" +
		"```\n\n" +
		"### 2. Generate migration\n" +
		"```bash\n" +
		"chameleon migrate --dry-run\n" +
		"```\n\n" +
		"### 3. Apply migration to database\n" +
		"```bash\n" +
		"# Make sure PostgreSQL is running\n" +
		"chameleon migrate --apply\n" +
		"```\n\n" +
		"## Schema\n\n" +
		"The schema is defined in `schema.cham`. Edit it to model your domain.\n" +
		"Psst, you can validate your schema with the [Visualizer Tool](https://chameleondb.dev/visualizer/schema-visualizer.html).\n\n" +
		"## Learn More\n\n" +
		"- [ChameleonDB Documentation](https://chameleondb.dev/docs)\n" +
		"- [Query API Reference]https://chameleondb.dev/docs/pages/query-reference.html"
}
