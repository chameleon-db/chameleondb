package integration

import (
	"strings"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine/introspect"
)

func TestIntrospectGetAllTablesAndGenerateSchema(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	inspector, err := introspect.NewIntrospector(ctx, testConfig().ConnectionString())
	if err != nil {
		t.Fatalf("failed to create introspector: %v", err)
	}
	defer inspector.Close()

	detected, err := inspector.Detect(ctx)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !detected {
		t.Fatal("expected database to be detected")
	}

	tables, err := inspector.GetAllTables(ctx)
	if err != nil {
		t.Fatalf("GetAllTables failed: %v", err)
	}
	if len(tables) != 4 {
		t.Fatalf("expected 4 tables, got %d", len(tables))
	}

	for _, table := range tables {
		if len(table.Columns) == 0 {
			t.Fatalf("table %s has no columns", table.Name)
		}

		seen := map[string]bool{}
		for _, col := range table.Columns {
			if seen[col.Name] {
				t.Fatalf("table %s has duplicated column %s", table.Name, col.Name)
			}
			seen[col.Name] = true
		}
	}

	schema, err := introspect.GenerateChameleonSchema(tables)
	if err != nil {
		t.Fatalf("GenerateChameleonSchema failed: %v", err)
	}

	if !strings.Contains(schema, "entity Users {") {
		t.Fatal("expected Users entity in generated schema")
	}
	if !strings.Contains(schema, "entity Orders {") {
		t.Fatal("expected Orders entity in generated schema")
	}
	if !strings.Contains(schema, "entity OrderItems {") {
		t.Fatal("expected OrderItems entity in generated schema")
	}
	if !strings.Contains(schema, "entity Posts {") {
		t.Fatal("expected Posts entity in generated schema")
	}
}
