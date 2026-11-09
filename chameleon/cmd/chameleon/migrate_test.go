package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/internal/schema"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
)

func TestTryMapErrorToSource(t *testing.T) {
	lineMap := map[int]schema.SourceLine{
		5:  {File: "schema.cham", LineNumber: 10},
		10: {File: "entities/user.cham", LineNumber: 5},
		15: {File: "entities/post.cham", LineNumber: 20},
		25: {File: "relations.cham", LineNumber: 3},
	}

	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{
			name:     "error with 'line N' pattern",
			errMsg:   "syntax error at line 10",
			expected: "Error in entities/user.cham:5",
		},
		{
			name:     "error with '--> file:N:col' pattern",
			errMsg:   "--> schema.cham:15:8 unexpected token",
			expected: "Error in entities/post.cham:20",
		},
		{
			name:     "error with ' N │' pattern",
			errMsg:   "  25 │ invalid syntax here",
			expected: "Error in relations.cham:3",
		},
		{
			name:     "error with nearby line (offset +1)",
			errMsg:   "line 11",
			expected: "Error in entities/user.cham:6",
		},
		{
			name:     "error with nearby line (offset -1)",
			errMsg:   "line 9",
			expected: "Error in entities/user.cham:4",
		},
		{
			name:     "error with no matching line",
			errMsg:   "line 100",
			expected: "",
		},
		{
			name:     "error with no pattern match",
			errMsg:   "random error message",
			expected: "",
		},
		{
			name:     "error at line 5",
			errMsg:   "error at line 5",
			expected: "Error in schema.cham:10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tryMapErrorToSource(tt.errMsg, lineMap)
			if got != tt.expected {
				t.Errorf("tryMapErrorToSource(%q, lineMap) = %q, want %q", tt.errMsg, got, tt.expected)
			}
		})
	}
}

func TestTryMapErrorToSourceEmptyMap(t *testing.T) {
	emptyMap := map[int]schema.SourceLine{}
	got := tryMapErrorToSource("line 10", emptyMap)
	if got != "" {
		t.Errorf("tryMapErrorToSource with empty map should return empty string, got %q", got)
	}
}

func TestMigrateConfigLoadFailure(t *testing.T) {
	t.Chdir(t.TempDir())

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected missing config error")
	}
	if want := "failed to load config"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateConfigParseFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := os.WriteFile(".chameleon.yml", []byte("database: ["), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected config parsing error")
	}
	if want := "failed to load config"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateConfigReadFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := os.Remove(".chameleon.yml"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(".chameleon.yml", 0755); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected config read error")
	}
	if want := "failed to load config"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateVaultInitializationFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := os.RemoveAll(filepath.Join(".chameleon", "vault")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".chameleon", "vault"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected vault initialization error")
	}
	if want := "failed to initialize vault"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateJournalInitializationFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := os.RemoveAll(filepath.Join(".chameleon", "journal")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".chameleon", "journal"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected journal initialization error")
	}
	if want := "failed to initialize journal"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateStateInitializationFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := os.RemoveAll(filepath.Join(".chameleon", "state")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".chameleon", "state"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected state initialization error")
	}
	if want := "failed to initialize state tracker"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateManifestLoadFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := vault.NewVault(".").Initialize(); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(".chameleon", "vault", "manifest.json")
	if err := os.WriteFile(manifestPath, []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected manifest integrity error")
	}
	if want := "integrity check failed"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateIntegrityViolation(t *testing.T) {
	initMigrateTestProject(t)

	schemaPath := filepath.Join("schemas", "example.cham")
	v := vault.NewVault(".")
	if _, err := v.RegisterVersion(schemaPath, "tester", "initial"); err != nil {
		t.Fatal(err)
	}

	hashPath := filepath.Join(".chameleon", "vault", "hashes", v.Manifest.CurrentVersion+".hash")
	if err := os.WriteFile(hashPath, []byte("tampered"), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected integrity check error")
	}
	if want := "integrity check failed"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateSchemaLoadFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := os.RemoveAll("schemas"); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected schema loading error")
	}
	if want := "failed to load schemas"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateSchemaValidationFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := os.WriteFile(filepath.Join("schemas", "example.cham"), []byte("// no entities"), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if want := "schema validation failed"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateDuplicateEntityValidationFailure(t *testing.T) {
	initMigrateTestProject(t)
	schema := "entity User { id: uuid primary }\nentity User { id: uuid primary }\n"
	if err := os.WriteFile(filepath.Join("schemas", "example.cham"), []byte(schema), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected duplicate entity validation error")
	}
	if want := "schema validation failed"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateConfigValidationFailure(t *testing.T) {
	initMigrateTestProject(t)
	config := "version: \"0.1.4\"\ndatabase:\n  driver: postgresql\n  connection_string: postgresql://localhost/test\nschema:\n  paths: []\n"
	if err := os.WriteFile(".chameleon.yml", []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected config validation error")
	}
	if want := "failed to load config"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateModeLoadFailure(t *testing.T) {
	initMigrateTestProject(t)
	v := vault.NewVault(".")
	if _, err := v.RegisterVersion(filepath.Join("schemas", "example.cham"), "tester", "initial"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".chameleon", "vault", "mode.json"), []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected paranoid mode error")
	}
	if want := "failed to read paranoid mode"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateVersionRegistrationFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := os.RemoveAll(filepath.Join(".chameleon", "vault")); err != nil {
		t.Fatal(err)
	}
	if err := vault.NewVault(".").Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(".chameleon", "vault", "versions")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".chameleon", "vault", "versions"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	oldDryRun, oldApply := dryRun, applyMigration
	dryRun, applyMigration = false, true
	t.Cleanup(func() { dryRun, applyMigration = oldDryRun, oldApply })

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected version registration error")
	}
	if want := "failed to register version"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigratePendingRegisteredVersionDryRun(t *testing.T) {
	initMigrateTestProject(t)
	oldDryRun, oldApply := dryRun, applyMigration
	dryRun, applyMigration = true, false
	t.Cleanup(func() { dryRun, applyMigration = oldDryRun, oldApply })
	if err := migrateCmd.RunE(nil, nil); err != nil {
		t.Fatalf("initial migrate dry run: %v", err)
	}

	v := vault.NewVault(".")
	if _, err := v.RegisterVersion(filepath.Join(".chameleon", "state", "schema.merged.cham"), "tester", "initial"); err != nil {
		t.Fatal(err)
	}
	if err := v.SetParanoidMode("standard"); err != nil {
		t.Fatal(err)
	}

	if err := migrateCmd.RunE(nil, nil); err != nil {
		t.Fatalf("migrate pending version dry run: %v", err)
	}
}

func TestMigrateParseDebugDirectoryFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := os.WriteFile(filepath.Join("schemas", "example.cham"), []byte("entity User {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	debugPath := filepath.Join(".chameleon", "state", "debug")
	if err := os.RemoveAll(debugPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(debugPath, []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected schema parse error")
	}
	if want := "failed to parse merged schemas"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateMergedOutputWriteFailure(t *testing.T) {
	initMigrateTestProject(t)
	rewriteMigrateConfig(t, func(data string) string {
		return strings.ReplaceAll(data, ".chameleon/state/schema.merged.cham", ".chameleon/state")
	})

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected merged schema write error")
	}
	if want := "failed to save merged schema"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateCurrentStateLoadFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := os.WriteFile(filepath.Join(".chameleon", "state", "current.state.json"), []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected current state error")
	}
	if want := "failed to load current state"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateCurrentStateReadFailure(t *testing.T) {
	initMigrateTestProject(t)
	statePath := filepath.Join(".chameleon", "state", "current.state.json")
	if err := os.Mkdir(statePath, 0755); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected current state read error")
	}
	if want := "failed to load current state"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateVaultCurrentVersionLoadFailure(t *testing.T) {
	initMigrateTestProject(t)
	v := vault.NewVault(".")
	if err := v.Initialize(); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(".chameleon", "vault", "manifest.json")
	rewriteFile(t, manifestPath, func(data string) string {
		return strings.Replace(data, `"current_version": ""`, `"current_version": "missing"`, 1)
	})

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected current vault version error")
	}
	if want := "failed to detect changes"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func TestMigrateMigrationManifestLoadFailure(t *testing.T) {
	initMigrateTestProject(t)
	if err := os.MkdirAll(filepath.Join(".chameleon", "state", "migrations"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".chameleon", "state", "migrations", "manifest.json"), []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}

	err := migrateCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("expected migration manifest error")
	}
	if want := "failed to get last migration"; !strings.Contains(err.Error(), want) {
		t.Fatalf("migrate error = %q, want it to contain %q", err, want)
	}
}

func initMigrateTestProject(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := initCmd.RunE(nil, nil); err != nil {
		t.Fatal(err)
	}
	configData, err := os.ReadFile(".chameleon.yml")
	if err != nil {
		t.Fatal(err)
	}
	configData = []byte(strings.ReplaceAll(string(configData), "${DATABASE_URL}", "postgresql://localhost/test"))
	if err := os.WriteFile(".chameleon.yml", configData, 0644); err != nil {
		t.Fatal(err)
	}
}

func rewriteMigrateConfig(t *testing.T, rewrite func(string) string) {
	t.Helper()
	rewriteFile(t, ".chameleon.yml", rewrite)
}

func rewriteFile(t *testing.T, path string, rewrite func(string) string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(rewrite(string(data))), 0644); err != nil {
		t.Fatal(err)
	}
}
