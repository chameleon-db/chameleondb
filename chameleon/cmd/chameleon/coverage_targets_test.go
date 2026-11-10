package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chameleon-db/chameleondb/chameleon/internal/config"
	"github.com/chameleon-db/chameleondb/chameleon/internal/journal"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
)

func TestLoadConnectorConfigFromProjectFile(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	t.Setenv("DATABASE_URL", "")

	loader := config.NewLoader(dir)
	cfg := config.Defaults()
	cfg.Database.ConnectionString = "postgresql://alice:secret@db.example:5434/app"
	if err := loader.Save(cfg); err != nil {
		t.Fatal(err)
	}

	got, err := LoadConnectorConfig()
	if err != nil {
		t.Fatalf("LoadConnectorConfig() error = %v", err)
	}
	if got.Host != "db.example" || got.Port != 5434 || got.Database != "app" || got.User != "alice" {
		t.Fatalf("unexpected config: %+v", got)
	}

	cfg.Database.ConnectionString = ""
	if err := loader.Save(cfg); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadConnectorConfig(); err != nil || got != engine.DefaultConfig() {
		t.Fatalf("empty project connection string = %+v, %v", got, err)
	}
}

func TestConfigCommandValidationBranches(t *testing.T) {
	if err := configSetCmd.RunE(nil, []string{"invalid"}); err == nil {
		t.Fatal("expected invalid key=value error")
	}
	if err := configSetCmd.RunE(nil, []string{"unknown=value"}); err == nil {
		t.Fatal("expected unsupported key error")
	}
	if err := configGetCmd.RunE(nil, []string{"unknown"}); err == nil {
		t.Fatal("expected unsupported get key error")
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	if err := configSetCmd.RunE(nil, []string{"mode=standard"}); err == nil {
		t.Fatal("expected mode change to require an initialized vault")
	}
	if err := configGetCmd.RunE(nil, []string{"mode"}); err == nil {
		t.Fatal("expected mode read to require an initialized vault")
	}

	v := vault.NewVault(dir)
	if err := v.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := config.NewLoader(dir).Save(config.Defaults()); err != nil {
		t.Fatal(err)
	}
	if err := configSetCmd.RunE(nil, []string{"schema-paths=schemas"}); err == nil {
		t.Fatal("expected schema path change to require elevated mode")
	}
	if err := configSetCmd.RunE(nil, []string{"mode=standard"}); err == nil {
		t.Fatal("expected mode upgrade without password to fail")
	}
	t.Setenv(modePasswordEnvVar, "test-password")
	if err := configAuthSetPasswordCmd.RunE(nil, nil); err != nil {
		t.Fatalf("set password: %v", err)
	}
	if err := configSetCmd.RunE(nil, []string{"mode=standard"}); err != nil {
		t.Fatalf("mode upgrade: %v", err)
	}
	if err := configSetCmd.RunE(nil, []string{"mode=admin"}); err != nil {
		t.Fatalf("admin alias upgrade: %v", err)
	}
	if err := configSetCmd.RunE(nil, []string{"mode=privileged"}); err != nil {
		t.Fatalf("privileged downgrade: %v", err)
	}
	schemaDir := filepath.Join(dir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	_, _ = w.WriteString("y\n")
	_ = w.Close()
	t.Cleanup(func() { os.Stdin = oldStdin })
	if err := configSetCmd.RunE(nil, []string{"schema-paths=" + schemaDir}); err != nil {
		t.Fatalf("schema paths update: %v", err)
	}
	if err := configSetCmd.RunE(nil, []string{"mode=emergency"}); err != nil {
		t.Fatalf("emergency upgrade: %v", err)
	}
	if err := configSetCmd.RunE(nil, []string{"schema-paths=."}); err != nil {
		t.Fatalf("expected emergency confirmation cancellation to be handled: %v", err)
	}
	if err := configGetCmd.RunE(nil, []string{"schema-paths"}); err != nil {
		t.Fatalf("schema paths read: %v", err)
	}
	if err := configGetCmd.RunE(nil, []string{"mode"}); err != nil {
		t.Fatalf("mode read: %v", err)
	}
	if err := configSetCmd.RunE(nil, []string{"mode=not-a-mode"}); err == nil {
		t.Fatal("expected invalid mode error")
	}
	if err := configSetCmd.RunE(nil, []string{"mode=readonly"}); err != nil {
		t.Fatalf("readonly downgrade: %v", err)
	}
}

func TestInitCommandCreatesProject(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	project := filepath.Join(dir, "sample")
	if err := initCmd.RunE(nil, []string{project}); err != nil {
		t.Fatalf("init command: %v", err)
	}
	for _, path := range []string{
		filepath.Join(project, ".chameleon.yml"),
		filepath.Join(project, ".chameleon"),
		filepath.Join(project, "schemas", "example.cham"),
		filepath.Join(project, "CHAMELEON.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected init output %s: %v", path, err)
		}
	}
	if err := initCmd.RunE(nil, []string{project}); err == nil {
		t.Fatal("expected reinitialization to fail")
	}
}

func TestIntrospectOverwriteAndSchemaClassificationBranches(t *testing.T) {
	oldForce := introspectForce
	introspectForce = false
	t.Cleanup(func() { introspectForce = oldForce })

	dir := t.TempDir()
	modified := filepath.Join(dir, "schema.cham")
	if err := os.WriteFile(modified, []byte("entity User {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })
	_, _ = w.WriteString("1\n")
	_ = w.Close()

	got, err := checkExistingSchemaAndGetOutput(modified)
	if err != nil || got != modified {
		t.Fatalf("backup overwrite = %q, %v", got, err)
	}
	if _, err := os.Stat(modified + ".backup"); err != nil {
		t.Fatalf("expected schema backup: %v", err)
	}

	if got, err := checkExistingSchemaAndGetOutput(filepath.Join(dir, "missing.cham")); err == nil || got != "" {
		t.Fatalf("expected missing file error, got %q, %v", got, err)
	}

	defaultPath := filepath.Join(dir, "default.cham")
	if err := os.WriteFile(defaultPath, []byte(`entity User {}
entity Post {}
entity Comment {}
posts: [Post] via user_id`), 0644); err != nil {
		t.Fatal(err)
	}
	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r2
	_, _ = w2.WriteString("no\n")
	_ = w2.Close()
	if _, err := checkExistingSchemaAndGetOutput(defaultPath); err == nil {
		t.Fatal("expected declined default schema overwrite")
	}

	choicePath := filepath.Join(dir, "choice.cham")
	if err := os.WriteFile(choicePath, []byte("entity User {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	r3, w3, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r3
	_, _ = w3.WriteString("2\ncustom-output\n")
	_ = w3.Close()
	got, err = checkExistingSchemaAndGetOutput(choicePath)
	if err != nil || got != "schemas/custom-output.cham" {
		t.Fatalf("alternate output = %q, %v", got, err)
	}

	introspectForce = true
	if got, err := validateAndGetOutputPath(filepath.Join(dir, "does-not-matter")); err != nil || got == "" {
		t.Fatalf("force output validation = %q, %v", got, err)
	}
	introspectForce = false
	if err := safeWriteSchema(filepath.Join(dir, "missing", "schema.cham"), "schema"); err == nil {
		t.Fatal("expected safe write failure for missing directory")
	}
	if _, err := askOverwriteWithBackupAndGetOutput(choicePath); err == nil {
		t.Fatal("expected EOF from overwrite prompt")
	}
	r4, w4, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r4
	_, _ = w4.WriteString("3\n")
	_ = w4.Close()
	if _, err := askOverwriteWithBackupAndGetOutput(choicePath); err == nil {
		t.Fatal("expected cancellation from overwrite prompt")
	}
	r5, w5, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r5
	_, _ = w5.WriteString("invalid\n")
	_ = w5.Close()
	if _, err := askOverwriteWithBackupAndGetOutput(choicePath); err == nil {
		t.Fatal("expected invalid overwrite choice")
	}
}

func TestMigrateDryRunUsesInitializedProject(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	oldDryRun, oldApply := dryRun, applyMigration
	dryRun, applyMigration = true, false
	t.Cleanup(func() { dryRun, applyMigration = oldDryRun, oldApply })

	if err := initCmd.RunE(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := migrateCmd.RunE(nil, nil); err != nil {
		t.Fatalf("migrate dry run: %v", err)
	}
}

func TestMigrateFailurePaths(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	if err := initCmd.RunE(nil, nil); err != nil {
		t.Fatal(err)
	}

	schemaPath := filepath.Join(dir, "schemas", "example.cham")
	if err := os.WriteFile(schemaPath, []byte("entity User {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	oldDryRun, oldApply := dryRun, applyMigration
	dryRun, applyMigration = true, false
	t.Cleanup(func() { dryRun, applyMigration = oldDryRun, oldApply })
	if err := migrateCmd.RunE(nil, nil); err == nil {
		t.Fatal("expected malformed schema error")
	}

	if err := os.WriteFile(schemaPath, []byte("entity User { id: uuid primary, }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	v := vault.NewVault(dir)
	if err := v.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := v.SetParanoidMode("readonly"); err != nil {
		t.Fatal(err)
	}
	if _, err := v.RegisterVersion(schemaPath, "tester", "initial"); err != nil {
		t.Fatal(err)
	}
	if err := migrateCmd.RunE(nil, nil); err == nil {
		t.Fatal("expected readonly migration error")
	}

	if err := v.SetParanoidMode("standard"); err != nil {
		t.Fatal(err)
	}
	oldDryRun, oldApply = false, true
	dryRun, applyMigration = oldDryRun, oldApply
	if err := migrateCmd.RunE(nil, nil); err == nil {
		t.Fatal("expected database connection failure")
	}
	dryRun, applyMigration = true, false
	if err := migrateCmd.RunE(nil, nil); err != nil {
		t.Fatalf("expected unchanged schema check to succeed: %v", err)
	}
}

func TestJournalVersionOutputBranches(t *testing.T) {
	dir := t.TempDir()
	v := vault.NewVault(dir)
	if err := v.Initialize(); err != nil {
		t.Fatal(err)
	}
	schemaPath := filepath.Join(dir, "schema.cham")
	if err := os.WriteFile(schemaPath, []byte("entity User {\n id: uuid primary,\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	entry, err := v.RegisterVersion(schemaPath, "tester", "initial schema")
	if err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		showVersionHistory(v)
		showVersionDetail(v, entry.Version)
		showVersionDetail(v, "missing")
	})
	for _, want := range []string{"Schema Version History", entry.Version, "initial schema", "not found"} {
		if !strings.Contains(out, want) {
			t.Fatalf("journal output missing %q: %s", want, out)
		}
	}

	empty := vault.NewVault(t.TempDir())
	if out := captureStdout(t, func() { runJournalSchema(nil, nil) }); !strings.Contains(out, "No vault found") {
		t.Fatalf("expected no-vault output: %s", out)
	}
	_ = empty
}

func TestJournalCommandRunEBranches(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	logger, err := journal.NewLogger(filepath.Join(dir, ".chameleon", "journal"))
	if err != nil {
		t.Fatal(err)
	}
	if err := logger.Log("test", "success", nil, nil); err != nil {
		t.Fatal(err)
	}
	oldFormat, oldLimit, oldSearchLimit := journalFormat, journalLimit, journalSearchLimit
	journalFormat, journalLimit, journalSearchLimit = "json", 1, 1
	t.Cleanup(func() { journalFormat, journalLimit, journalSearchLimit = oldFormat, oldLimit, oldSearchLimit })
	if err := journalLastCmd.RunE(nil, nil); err != nil {
		t.Fatalf("journal last: %v", err)
	}
	if err := journalErrorsCmd.RunE(nil, nil); err != nil {
		t.Fatalf("journal errors: %v", err)
	}
	if err := journalMigrationsCmd.RunE(nil, nil); err != nil {
		t.Fatalf("journal migrations: %v", err)
	}
	if err := journalSearchCmd.RunE(nil, []string{"test"}); err != nil {
		t.Fatalf("journal search: %v", err)
	}
	journalSearchLimit = -1
	if err := journalSearchCmd.RunE(nil, []string{"test"}); err == nil {
		t.Fatal("expected negative search limit error")
	}
}

func TestCheckCommandRunEBranches(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.cham")
	invalid := filepath.Join(dir, "invalid.cham")
	if err := os.WriteFile(valid, []byte("entity User { id: uuid primary, }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalid, []byte("entity User {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	oldJSON := outputJSON
	t.Cleanup(func() { outputJSON = oldJSON })
	outputJSON = true
	if err := checkCmd.RunE(nil, []string{valid}); err != nil {
		t.Fatalf("valid check: %v", err)
	}
	if err := checkCmd.RunE(nil, []string{invalid}); err != nil {
		t.Fatalf("invalid check: %v", err)
	}
	if err := checkCmd.RunE(nil, []string{filepath.Join(dir, "missing.cham")}); err != nil {
		t.Fatalf("missing check: %v", err)
	}
}

func TestCheckOutputHelpers(t *testing.T) {
	snippet := "entity User {"
	raw := `{"valid":false,"errors":[{"message":"bad","line":3,"column":2,"snippet":"` + snippet + `"}]}`
	if out := captureStdout(t, func() { _ = printCheckErrors("schema.cham", raw) }); !strings.Contains(out, `"line": 3`) {
		t.Fatalf("parsed check output: %s", out)
	}
	if out := captureStdout(t, func() { _ = printCheckErrors("schema.cham", "") }); !strings.Contains(out, "schema validation failed") {
		t.Fatalf("fallback check output: %s", out)
	}
	if out := captureStdout(t, func() { _ = printJSONError("schema.cham", "read failed") }); !strings.Contains(out, "read failed") {
		t.Fatalf("json error output: %s", out)
	}
	if out := captureStdout(t, printJSONSuccess); !strings.Contains(out, `"valid": true`) {
		t.Fatalf("json success output: %s", out)
	}
	raw = `{"valid":false,"errors":[{"message":"bad"}]}`
	if out := captureStdout(t, func() { _ = printCheckErrors("schema.cham", raw) }); !strings.Contains(out, `"line": 1`) {
		t.Fatalf("default check coordinates: %s", out)
	}
}

func TestJournalJSONAndTableOptionalFields(t *testing.T) {
	entries := []*journal.Entry{{
		Timestamp: time.Unix(0, 0).UTC(),
		Action:    "test",
		Status:    "ok",
		Details:   map[string]interface{}{"value": "x"},
	}}
	out := captureStdout(t, func() {
		printEntriesTable(entries)
		printEntriesJSON(entries)
	})
	if !strings.Contains(out, "test") || !strings.Contains(out, `"action": "test"`) {
		t.Fatalf("journal output: %s", out)
	}
}
