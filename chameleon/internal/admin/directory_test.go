package admin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectoryInitializeCreatesStructure(t *testing.T) {
	workDir := t.TempDir()
	dir := NewDirectory(workDir)

	if err := dir.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	expected := []string{
		filepath.Join(workDir, ".chameleon"),
		filepath.Join(workDir, ".chameleon", "state"),
		filepath.Join(workDir, ".chameleon", "state", "migrations"),
		filepath.Join(workDir, ".chameleon", "state", "debug"),
		filepath.Join(workDir, ".chameleon", "journal"),
		filepath.Join(workDir, ".chameleon", "backups"),
		filepath.Join(workDir, ".chameleon", ".gitignore"),
	}

	for _, path := range expected {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected path to exist: %s (%v)", path, err)
		}
	}
}

func TestDirectoryGetPaths(t *testing.T) {
	workDir := t.TempDir()
	dir := NewDirectory(workDir)
	paths := dir.GetPaths()

	if paths.Root != filepath.Join(workDir, ".chameleon") {
		t.Fatalf("unexpected root path: %s", paths.Root)
	}
	if paths.Migrations != filepath.Join(workDir, ".chameleon", "state", "migrations") {
		t.Fatalf("unexpected migrations path: %s", paths.Migrations)
	}
	if paths.Journal != filepath.Join(workDir, ".chameleon", "journal") {
		t.Fatalf("unexpected journal path: %s", paths.Journal)
	}
}

func TestManagerFactoryStatusNotInitialized(t *testing.T) {
	workDir := t.TempDir()
	factory := NewManagerFactory(workDir)

	status, err := factory.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != "not_initialized" {
		t.Fatalf("expected not_initialized, got %q", status)
	}
}

func TestManagerFactoryInitializeAndStatus(t *testing.T) {
	workDir := t.TempDir()
	factory := NewManagerFactory(workDir)

	if err := factory.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	status, err := factory.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	checks := []string{"initialized", "Config loaded: no", "Journal:", "Backups:"}
	for _, check := range checks {
		if !strings.Contains(status, check) {
			t.Fatalf("status output missing %q. got:\n%s", check, status)
		}
	}
}

func TestManagerFactoryCreateManagers(t *testing.T) {
	workDir := t.TempDir()
	factory := NewManagerFactory(workDir)

	if err := factory.Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	logger, err := factory.CreateJournalLogger()
	if err != nil {
		t.Fatalf("CreateJournalLogger() error = %v", err)
	}
	if logger == nil {
		t.Fatal("expected non-nil journal logger")
	}

	tracker, err := factory.CreateStateTracker()
	if err != nil {
		t.Fatalf("CreateStateTracker() error = %v", err)
	}
	if tracker == nil {
		t.Fatal("expected non-nil state tracker")
	}

	loader := factory.CreateConfigLoader()
	if loader == nil {
		t.Fatal("expected non-nil config loader")
	}
}
