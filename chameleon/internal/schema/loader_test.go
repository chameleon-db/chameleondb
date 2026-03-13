package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileLoaderLoad(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "schema.cham")
	content := "entity User { id: uuid primary, }"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	loader := NewFileLoader([]string{dir})
	got, err := loader.Load(file)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != content {
		t.Fatalf("unexpected content: %s", got)
	}
}

func TestFileLoaderLoadAll(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "b.cham"), []byte("entity B { id: uuid primary, }"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.cham"), []byte("entity A { id: uuid primary, }"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("ignore"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	loader := NewFileLoader([]string{dir})
	names, contents, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(names) != 2 || len(contents) != 2 {
		t.Fatalf("expected 2 schemas, got names=%d contents=%d", len(names), len(contents))
	}
	if names[0] != "a.cham" || names[1] != "b.cham" {
		t.Fatalf("expected sorted filenames [a.cham b.cham], got %#v", names)
	}
}

func TestFileLoaderLoadAllNoSchemaFiles(t *testing.T) {
	dir := t.TempDir()
	loader := NewFileLoader([]string{dir})
	_, _, err := loader.LoadAll()
	if err == nil {
		t.Fatal("expected error when no .cham files found")
	}
}
