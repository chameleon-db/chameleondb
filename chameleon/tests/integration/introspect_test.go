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

	// toEntityName() singularizes table names: users -> User, orders -> Order
	if !strings.Contains(schema, "entity User {") {
		t.Fatalf("expected User entity in generated schema, got:\n%s", schema)
	}
	if !strings.Contains(schema, "entity Order {") {
		t.Fatalf("expected Order entity in generated schema, got:\n%s", schema)
	}
	if !strings.Contains(schema, "entity OrderItem {") {
		t.Fatalf("expected OrderItem entity in generated schema, got:\n%s", schema)
	}
	if !strings.Contains(schema, "entity Post {") {
		t.Fatalf("expected Post entity in generated schema, got:\n%s", schema)
	}
}
