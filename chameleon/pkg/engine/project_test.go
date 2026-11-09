package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
)

// initProject creates a project directory with an initialized vault and a
// schema source, and returns its path.
func initProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := vault.NewVault(dir).Initialize(); err != nil {
		t.Fatalf("vault init failed: %v", err)
	}
	return dir
}

func TestOpenProject_FromAnotherWorkingDirectory(t *testing.T) {
	project := initProject(t)

	elsewhere := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(elsewhere); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	p, err := openProject(project)
	if err != nil {
		t.Fatalf("expected the project to open from another directory, got: %v", err)
	}

	if p.dir != project {
		t.Errorf("expected dir %s, got %s", project, p.dir)
	}
	want := filepath.Join(project, defaultMergedSchemaPath)
	if p.schemaSourcePath != want {
		t.Errorf("expected schema source %s, got %s", want, p.schemaSourcePath)
	}
}

func TestOpenProject_RelativeDirectory(t *testing.T) {
	project := initProject(t)

	prev, _ := os.Getwd()
	if err := os.Chdir(filepath.Dir(project)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	p, err := openProject(filepath.Base(project))
	if err != nil {
		t.Fatalf("relative directory should resolve: %v", err)
	}
	if !filepath.IsAbs(p.dir) {
		t.Errorf("project dir should be absolute, got %s", p.dir)
	}
}

func TestOpenProject_ConfigPathsResolveAgainstProject(t *testing.T) {
	project := initProject(t)
	cfg := "version: \"0.1.4\"\ndatabase:\n  driver: postgresql\n  connection_string: postgresql://localhost/x\nschema:\n  paths:\n    - ./schemas\n  merged_output: .chameleon/state/custom.cham\n"
	if err := os.WriteFile(filepath.Join(project, ".chameleon.yml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(t.TempDir())

	p, err := openProject(project)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(project, ".chameleon/state/custom.cham")
	if p.schemaSourcePath != want {
		t.Errorf("expected %s, got %s", want, p.schemaSourcePath)
	}
}

func TestOpenProject_NoVaultNamesTheDirectory(t *testing.T) {
	dir := t.TempDir()

	_, err := openProject(dir)
	if err == nil {
		t.Fatal("expected an error without a vault")
	}

	msg := err.Error()
	for _, want := range []string{
		dir,
		filepath.Join(dir, vault.VaultDirName),
		"chameleon migrate --apply",
		"NewEngineAt",
		ProjectDirEnv,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error should mention %q, got:\n%s", want, msg)
		}
	}
}

func TestOpenProject_MissingDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")

	_, err := openProject(missing)
	if err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("expected an error naming %s, got: %v", missing, err)
	}
}

func TestOpenProject_NotADirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := openProject(file)
	if err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("expected a not-a-directory error, got: %v", err)
	}
}

func TestOpenProject_IntegrityFailureSaysWhy(t *testing.T) {
	dir := initProject(t)

	manifest := filepath.Join(dir, vault.VaultDirName, vault.ManifestFileName)
	if err := os.WriteFile(manifest, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := openProject(dir)
	if err == nil {
		t.Fatal("expected an integrity error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "integrity check failed") || !strings.Contains(msg, dir) {
		t.Errorf("error should name the directory, got: %s", msg)
	}
	if !strings.Contains(msg, "manifest") {
		t.Errorf("error should carry the cause, got: %s", msg)
	}
}
