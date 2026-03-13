package schema
package schema

import (
	"strings"
	"testing"
)

func TestSimpleMergerMergeAndLineMap(t *testing.T) {
	merger := NewSimpleMerger()
	result, err := merger.Merge(
		[]string{"a.cham", "b.cham"},
		[]string{"entity A {\n  id: uuid primary,\n}", "entity B {\n  id: uuid primary,\n}"},
	)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil merge result")
	}
	if !strings.Contains(result.Content, "// From: a.cham") || !strings.Contains(result.Content, "// From: b.cham") {
		t.Fatalf("expected source headers in merged content, got:\n%s", result.Content)
	}
	if len(result.LineMap) == 0 {
		t.Fatal("expected non-empty line map")
	}
}

func TestSimpleMergerMergeErrors(t *testing.T) {
	merger := NewSimpleMerger()
	if _, err := merger.Merge([]string{}, []string{}); err == nil {
		t.Fatal("expected error when no files are provided")
	}
	if _, err := merger.Merge([]string{"a.cham"}, []string{}); err == nil {
		t.Fatal("expected mismatch error for filenames/contents")
	}
}

func TestSimpleMergerValidate(t *testing.T) {
	merger := NewSimpleMerger()

	if err := merger.Validate("// empty"); err == nil {
		t.Fatal("expected error when no entities are found")
	}

	duplicate := `
entity User {
  id: uuid primary,
}

entity User {
  id: uuid primary,
}
`
	if err := merger.Validate(duplicate); err == nil {
		t.Fatal("expected duplicate entity validation error")
	}

	valid := `
entity User {
  id: uuid primary,
}
entity Order {
  id: uuid primary,
}
`
	if err := merger.Validate(valid); err != nil {
		t.Fatalf("expected valid schema, got error: %v", err)
	}
}
