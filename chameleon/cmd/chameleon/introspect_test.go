package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAndGetOutputPath(t *testing.T) {
	oldForce := introspectForce
	defer func() { introspectForce = oldForce }()

	t.Run("returns non-existing file", func(t *testing.T) {
		introspectForce = false
		tmpDir := t.TempDir()
		out := filepath.Join(tmpDir, "schema.cham")

		got, err := validateAndGetOutputPath(out)
		if err != nil {
			t.Fatalf("validateAndGetOutputPath() error = %v", err)
		}
		if got != out {
			t.Fatalf("validateAndGetOutputPath() = %q, want %q", got, out)
		}
	})

	t.Run("rejects directory output", func(t *testing.T) {
		introspectForce = false
		tmpDir := t.TempDir()

		_, err := validateAndGetOutputPath(tmpDir)
		if err == nil {
			t.Fatal("expected error for directory output path")
		}
	})
}

func TestSchemaContentHelpers(t *testing.T) {
	t.Run("isModifiedSchema", func(t *testing.T) {
		if isModifiedSchema("entity User {\n  id: uuid primary,\n}\n") != true {
			t.Fatal("expected modified schema to be detected")
		}
		if isModifiedSchema("") != false {
			t.Fatal("expected empty schema to not be modified")
		}
	})

	t.Run("isEmpty", func(t *testing.T) {
		onlyComments := "// header\n\n// notes\n"
		if !isEmpty(onlyComments) {
			t.Fatal("expected comments-only content to be empty")
		}

		notEmpty := "// header\nentity User {\n}\n"
		if isEmpty(notEmpty) {
			t.Fatal("expected entity content to be non-empty")
		}
	})
}

func TestCopyFileAndSafeWriteSchema(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.cham")
	dst := filepath.Join(tmpDir, "backup.cham")
	out := filepath.Join(tmpDir, "output.cham")

	const sourceContent = "entity User {\n  id: uuid primary,\n}\n"
	if err := os.WriteFile(src, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("failed creating source file: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	copied, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed reading backup file: %v", err)
	}
	if string(copied) != sourceContent {
		t.Fatalf("backup content mismatch: got %q", string(copied))
	}

	if err := safeWriteSchema(out, "entity Order {\n}\n"); err != nil {
		t.Fatalf("safeWriteSchema() error = %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
}

func TestResolveIntrospectConnectionString(t *testing.T) {
	t.Run("returns literal connection string", func(t *testing.T) {
		got, err := resolveIntrospectConnectionString("postgresql://user:pass@localhost:5432/db")
		if err != nil {
			t.Fatalf("resolveIntrospectConnectionString() error = %v", err)
		}
		if got != "postgresql://user:pass@localhost:5432/db" {
			t.Fatalf("unexpected connection string: %q", got)
		}
	})

	t.Run("resolves dollar env reference", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgresql://railway:pass@host:5432/db")

		got, err := resolveIntrospectConnectionString("$DATABASE_URL")
		if err != nil {
			t.Fatalf("resolveIntrospectConnectionString() error = %v", err)
		}
		if got != "postgresql://railway:pass@host:5432/db" {
			t.Fatalf("unexpected connection string: %q", got)
		}
	})

	t.Run("resolves bracket env reference", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgresql://railway:pass@host:5432/db")

		got, err := resolveIntrospectConnectionString("${DATABASE_URL}")
		if err != nil {
			t.Fatalf("resolveIntrospectConnectionString() error = %v", err)
		}
		if got != "postgresql://railway:pass@host:5432/db" {
			t.Fatalf("unexpected connection string: %q", got)
		}
	})

	t.Run("resolves env prefix reference", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgresql://railway:pass@host:5432/db")

		got, err := resolveIntrospectConnectionString("env:DATABASE_URL")
		if err != nil {
			t.Fatalf("resolveIntrospectConnectionString() error = %v", err)
		}
		if got != "postgresql://railway:pass@host:5432/db" {
			t.Fatalf("unexpected connection string: %q", got)
		}
	})

	t.Run("fails when env var is missing", func(t *testing.T) {
		_, err := resolveIntrospectConnectionString("$DOES_NOT_EXIST")
		if err == nil {
			t.Fatal("expected error for missing environment variable")
		}
	})

	t.Run("fails when env var name is invalid", func(t *testing.T) {
		_, err := resolveIntrospectConnectionString("$1INVALID")
		if err == nil {
			t.Fatal("expected error for invalid environment variable name")
		}
	})
}
