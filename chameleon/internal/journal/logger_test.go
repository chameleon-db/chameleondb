package journal

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoggerCreatesDirectory(t *testing.T) {
	journalDir := filepath.Join(t.TempDir(), "journal")

	logger, err := NewLogger(journalDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	if st, err := os.Stat(journalDir); err != nil || !st.IsDir() {
		t.Fatalf("expected journal directory to exist: %v", err)
	}
}

func TestLogAndLast(t *testing.T) {
	journalDir := t.TempDir()
	logger, err := NewLogger(journalDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	if err := logger.Log("validate", "ok", map[string]interface{}{"entity": "User"}, nil); err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	if err := logger.Log("migrate", "ok", map[string]interface{}{"version": "v001"}, nil); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	entries, err := logger.Last(10)
	if err != nil {
		t.Fatalf("Last() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Action != "validate" || entries[1].Action != "migrate" {
		t.Fatalf("unexpected actions: %#v", []string{entries[0].Action, entries[1].Action})
	}
}

func TestErrorsFiltersEntries(t *testing.T) {
	journalDir := t.TempDir()
	logger, err := NewLogger(journalDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	if err := logger.Log("validate", "ok", nil, nil); err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	if err := logger.LogError("migrate", errors.New("boom"), map[string]interface{}{"step": "apply"}); err != nil {
		t.Fatalf("LogError() error = %v", err)
	}

	errEntries, err := logger.Errors()
	if err != nil {
		t.Fatalf("Errors() error = %v", err)
	}
	if len(errEntries) != 1 {
		t.Fatalf("expected 1 error entry, got %d", len(errEntries))
	}
	if errEntries[0].Action != "migrate" || errEntries[0].Status != "error" {
		t.Fatalf("unexpected error entry: %#v", errEntries[0])
	}
}

func TestLogMigrationAndMigrations(t *testing.T) {
	journalDir := t.TempDir()
	logger, err := NewLogger(journalDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	if err := logger.LogMigration("v002", "ok", 17, "backup.sql", map[string]interface{}{"from": "v001"}); err != nil {
		t.Fatalf("LogMigration() error = %v", err)
	}
	if err := logger.Log("validate", "ok", nil, nil); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	entries, err := logger.Migrations()
	if err != nil {
		t.Fatalf("Migrations() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 migration entry, got %d", len(entries))
	}
	if entries[0].Action != "migrate" {
		t.Fatalf("expected migrate action, got %s", entries[0].Action)
	}
}

func TestLastAndErrorsWhenNoFile(t *testing.T) {
	journalDir := t.TempDir()
	logger, err := NewLogger(journalDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entries, err := logger.Last(5)
	if err != nil {
		t.Fatalf("Last() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty entries, got %d", len(entries))
	}

	errEntries, err := logger.Errors()
	if err != nil {
		t.Fatalf("Errors() error = %v", err)
	}
	if len(errEntries) != 0 {
		t.Fatalf("expected empty errors, got %d", len(errEntries))
	}
}
