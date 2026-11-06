package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadValidateInputExplicitFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "other.cham")
	if err := os.WriteFile(path, []byte("entity A { id: uuid primary }"), 0644); err != nil {
		t.Fatal(err)
	}

	label, content, err := readValidateInput([]string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if label != path || !strings.Contains(content, "entity A") {
		t.Fatalf("got label=%q content=%q", label, content)
	}
}

func TestReadValidateInputMissingExplicitFile(t *testing.T) {
	_, _, err := readValidateInput([]string{filepath.Join(t.TempDir(), "nope.cham")})
	if err == nil || !strings.Contains(err.Error(), "schema file not found") {
		t.Fatalf("expected a not-found error, got %v", err)
	}
}

func TestReadValidateInputPrefersLegacySchemaFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, legacySchemaFile), []byte("entity B { id: uuid primary }"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	label, content, err := readValidateInput(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if label != legacySchemaFile || !strings.Contains(content, "entity B") {
		t.Fatalf("got label=%q content=%q", label, content)
	}
}

func TestReadValidateInputWithoutAnySchemaExplainsWhat(t *testing.T) {
	t.Chdir(t.TempDir())

	_, _, err := readValidateInput(nil)
	if err == nil || !strings.Contains(err.Error(), "no schema file given") {
		t.Fatalf("expected an explanatory error, got %v", err)
	}
}
