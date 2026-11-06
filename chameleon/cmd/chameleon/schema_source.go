package main

import (
	"fmt"
	"os"

	"github.com/chameleon-db/chameleondb/chameleon/internal/admin"
	"github.com/chameleon-db/chameleondb/chameleon/internal/schema"
)

// legacySchemaFile is the single-file layout used before schema.paths existed.
const legacySchemaFile = "schema.cham"

// loadProjectSchema loads every schema file configured in .chameleon.yml
// (schema.paths) and merges them exactly as 'chameleon migrate' does, so
// the result is the schema the vault registers.
func loadProjectSchema(workDir string) (string, error) {
	cfg, err := admin.NewManagerFactory(workDir).CreateConfigLoader().Load()
	if err != nil {
		return "", err
	}

	filenames, contents, err := schema.NewFileLoader(cfg.Schema.Paths).LoadAll()
	if err != nil {
		return "", fmt.Errorf("failed to load schemas: %w", err)
	}

	merged, err := schema.NewSimpleMerger().Merge(filenames, contents)
	if err != nil {
		return "", fmt.Errorf("failed to merge schemas: %w", err)
	}

	return merged.Content, nil
}

// readValidateInput picks the schema 'chameleon validate' checks: the file
// given as argument, else schema.cham in the current directory, else the
// schemas configured for the project.
func readValidateInput(args []string) (label string, content string, err error) {
	if len(args) > 0 {
		label = args[0]
		if _, statErr := os.Stat(label); os.IsNotExist(statErr) {
			return "", "", fmt.Errorf("schema file not found: %s", label)
		}
	} else if _, statErr := os.Stat(legacySchemaFile); statErr == nil {
		label = legacySchemaFile
	}

	if label != "" {
		data, readErr := os.ReadFile(label)
		if readErr != nil {
			return "", "", fmt.Errorf("failed to read file: %w", readErr)
		}
		return label, string(data), nil
	}

	workDir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("failed to get working directory: %w", err)
	}

	merged, err := loadProjectSchema(workDir)
	if err != nil {
		return "", "", fmt.Errorf("no schema file given, no %s here and no project schemas found: %w", legacySchemaFile, err)
	}
	return "project schemas", merged, nil
}
