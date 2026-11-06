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

func readGitignore(t *testing.T, workDir string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(workDir, ".chameleon", ".gitignore"))
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}
	return string(data)
}

func TestInitializeIgnoresModePasswordVerifier(t *testing.T) {
	workDir := t.TempDir()
	if err := NewDirectory(workDir).Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	content := readGitignore(t, workDir)
	for _, rule := range []string{"state/", "journal/", "backups/", "vault/mode_auth.json"} {
		if !strings.Contains(content, rule+"\n") {
			t.Fatalf(".gitignore should contain %q, got:\n%s", rule, content)
		}
	}
	if strings.Contains(content, "\nvault/\n") {
		t.Fatalf("the whole vault must stay versioned, got:\n%s", content)
	}
}

func TestEnsureGitignoreUpgradesOldFileWithoutDuplicates(t *testing.T) {
	workDir := t.TempDir()
	root := filepath.Join(workDir, ".chameleon")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}

	// What older versions wrote, plus a rule the user added, with no final newline.
	old := "# ChameleonDB Administrative Files\nstate/\njournal/\nbackups/\n\n!config.yml\nmy-local-notes/"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(old), 0644); err != nil {
		t.Fatal(err)
	}

	dir := NewDirectory(workDir)
	if err := dir.EnsureGitignore(); err != nil {
		t.Fatalf("EnsureGitignore() error = %v", err)
	}
	first := readGitignore(t, workDir)

	if !strings.Contains(first, "my-local-notes/\nvault/mode_auth.json\n") {
		t.Fatalf("missing rule should be appended on its own line, got:\n%s", first)
	}
	if !strings.Contains(first, "!config.yml") {
		t.Fatalf("existing user content must be kept, got:\n%s", first)
	}
	if strings.Count(first, "state/") != 1 {
		t.Fatalf("existing rules must not be duplicated, got:\n%s", first)
	}

	if err := dir.EnsureGitignore(); err != nil {
		t.Fatalf("second EnsureGitignore() error = %v", err)
	}
	if second := readGitignore(t, workDir); second != first {
		t.Fatalf("EnsureGitignore should be idempotent.\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}
