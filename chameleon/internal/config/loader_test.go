package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLoader(t *testing.T) {
	workDir := "/test/work/dir"
	loader := NewLoader(workDir)

	if loader == nil {
		t.Fatal("Expected non-nil loader")
	}

	expectedPath := filepath.Join(workDir, ".chameleon.yml")
	if loader.filePath != expectedPath {
		t.Errorf("Expected filePath %s, got %s", expectedPath, loader.filePath)
	}

	if loader.workDir != workDir {
		t.Errorf("Expected workDir %s, got %s", workDir, loader.workDir)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	_, err := loader.Load()
	if err == nil {
		t.Fatal("Expected error when config file doesn't exist")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestLoad_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".chameleon.yml")

	configContent := `version: "0.1.4"
database:
  driver: "postgresql"
  connection_string: "postgresql://localhost:5432/test"
  max_connections: 10
  connection_timeout: 30
  migration_timeout: 300

schema:
  paths:
    - "./schemas"
  merged_output: ".chameleon/state/schema.merged.cham"
  validation_strict: false

features:
  auto_migration: true
  rollback_enabled: true
  audit_logging: false
  backup_on_migrate: false
  dry_run_default: false

safety:
  require_confirmation: false
  backup_before_apply: true
  validate_schema: true
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewLoader(tmpDir)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Version != "0.1.4" {
		t.Errorf("Expected version 0.1.4, got %s", cfg.Version)
	}

	if cfg.Database.Driver != "postgresql" {
		t.Errorf("Expected driver postgresql, got %s", cfg.Database.Driver)
	}

	if cfg.Database.MaxConnections != 10 {
		t.Errorf("Expected max_connections 10, got %d", cfg.Database.MaxConnections)
	}

	if !cfg.Features.AutoMigration {
		t.Error("Expected auto_migration to be true")
	}

	if !cfg.Safety.BackupBeforeApply {
		t.Error("Expected backup_before_apply to be true")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".chameleon.yml")

	invalidYAML := `version: "0.1.4"
database:
  driver: postgresql
  connection_string: [this is invalid yaml syntax
`

	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewLoader(tmpDir)
	_, err = loader.Load()
	if err == nil {
		t.Fatal("Expected error when parsing invalid YAML")
	}

	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected 'failed to parse' error, got: %v", err)
	}
}

func TestLoad_ExpandsEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".chameleon.yml")

	// Set test environment variable
	testDBURL := "postgresql://testhost:5432/testdb"
	os.Setenv("TEST_DATABASE_URL", testDBURL)
	defer os.Unsetenv("TEST_DATABASE_URL")

	configContent := `version: "0.1.4"
database:
  driver: "postgresql"
  connection_string: "${TEST_DATABASE_URL}"
  max_connections: 10
  connection_timeout: 30
  migration_timeout: 300

schema:
  paths:
    - "./schemas"
  merged_output: ".chameleon/state/schema.merged.cham"

features:
  auto_migration: true

safety:
  validate_schema: true
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewLoader(tmpDir)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Database.ConnectionString != testDBURL {
		t.Errorf("Expected connection string %s, got %s", testDBURL, cfg.Database.ConnectionString)
	}
}

func TestResolvePaths(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".chameleon.yml")

	configContent := `version: "0.1.4"
database:
  driver: "postgresql"
  connection_string: "postgresql://localhost:5432/test"
  max_connections: 10
  connection_timeout: 30
  migration_timeout: 300

schema:
  paths:
    - "./schemas"
    - "another/path"
  merged_output: ".chameleon/state/schema.merged.cham"

features:
  auto_migration: true

safety:
  validate_schema: true
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewLoader(tmpDir)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that paths are absolute
	for _, path := range cfg.Schema.Paths {
		if !filepath.IsAbs(path) {
			t.Errorf("Expected absolute path, got relative: %s", path)
		}
	}

	if !filepath.IsAbs(cfg.Schema.MergedOutput) {
		t.Errorf("Expected absolute merged_output path, got relative: %s", cfg.Schema.MergedOutput)
	}
}

func TestLoadOrDefault_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	cfg, err := loader.LoadOrDefault()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should return defaults
	if cfg == nil {
		t.Fatal("Expected default config, got nil")
	}

	// Verify it's the default config
	defaults := Defaults()
	if cfg.Version != defaults.Version {
		t.Errorf("Expected default version %s, got %s", defaults.Version, cfg.Version)
	}
}

func TestLoadOrDefault_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".chameleon.yml")

	configContent := `version: "0.1.4"
database:
  driver: "postgresql"
  connection_string: "postgresql://localhost:5432/test"
  max_connections: 10
  connection_timeout: 30
  migration_timeout: 300

schema:
  paths:
    - "./schemas"

features:
  auto_migration: true

safety:
  validate_schema: true
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewLoader(tmpDir)
	cfg, err := loader.LoadOrDefault()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Version != "0.1.4" {
		t.Errorf("Expected version 0.1.4, got %s", cfg.Version)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	cfg := Defaults()
	cfg.Database.ConnectionString = "postgresql://localhost:5432/savetest"

	err := loader.Save(cfg)
	if err != nil {
		t.Fatalf("Expected no error saving config, got: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tmpDir, ".chameleon.yml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Config file was not created")
	}

	// Load it back and verify
	loadedCfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Expected no error loading config, got: %v", err)
	}

	if loadedCfg.Database.ConnectionString != "postgresql://localhost:5432/savetest" {
		t.Errorf("Expected connection string to be saved correctly")
	}
}

func TestTemplate(t *testing.T) {
	template := Template()

	if template == "" {
		t.Fatal("Expected non-empty template")
	}

	if !strings.Contains(template, "ChameleonDB Configuration") {
		t.Error("Template should contain title")
	}

	if !strings.Contains(template, "database:") {
		t.Error("Template should contain database section")
	}

	if !strings.Contains(template, "schema:") {
		t.Error("Template should contain schema section")
	}

	if !strings.Contains(template, "features:") {
		t.Error("Template should contain features section")
	}

	if !strings.Contains(template, "safety:") {
		t.Error("Template should contain safety section")
	}
}
