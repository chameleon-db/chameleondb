package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chameleon-db/chameleondb/chameleon/internal/config"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
)

// ProjectDirEnv names the environment variable that NewEngine reads to find
// the project directory when it is not the working directory.
const ProjectDirEnv = "CHAMELEON_PROJECT_DIR"

const defaultMergedSchemaPath = ".chameleon/state/schema.merged.cham"

// project is an initialized ChameleonDB project found on disk.
type project struct {
	dir              string
	vault            *vault.Vault
	schemaSourcePath string
}

// openProject locates the project in dir, checks that its vault exists and is
// intact, and resolves where the schema is read from. Every error names the
// directory that was searched.
func openProject(dir string) (*project, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project directory %q: %w", dir, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("project directory %s: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project directory %s is not a directory", abs)
	}

	schemaSourcePath, err := resolveSchemaSourcePath(abs)
	if err != nil {
		return nil, err
	}

	v := vault.NewVault(abs)
	if !v.Exists() {
		return nil, fmt.Errorf(
			"vault not initialized in %s (looked for %s)\n"+
				"Run 'chameleon migrate --apply' in the project directory, "+
				"or point to the project with NewEngineAt or %s",
			abs, filepath.Join(abs, vault.VaultDirName), ProjectDirEnv)
	}

	result, err := v.VerifyIntegrity()
	if err != nil {
		return nil, fmt.Errorf("integrity check failed in %s: %w", abs, err)
	}
	if !result.Valid {
		return nil, fmt.Errorf("integrity check failed in %s: %s", abs, strings.Join(result.Issues, "; "))
	}

	return &project{dir: abs, vault: v, schemaSourcePath: schemaSourcePath}, nil
}

func resolveSchemaSourcePath(workDir string) (string, error) {
	defaultPath, err := filepath.Abs(filepath.Join(workDir, defaultMergedSchemaPath))
	if err != nil {
		return "", fmt.Errorf("failed to resolve default schema path: %w", err)
	}

	configPath := filepath.Join(workDir, ".chameleon.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return defaultPath, nil
	} else if err != nil {
		return "", fmt.Errorf("failed to read config path: %w", err)
	}

	loader := config.NewLoader(workDir)
	cfg, err := loader.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load config for schema source: %w", err)
	}

	if cfg.Schema.MergedOutput != "" {
		return cfg.Schema.MergedOutput, nil
	}

	return defaultPath, nil
}
