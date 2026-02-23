package engine

import (
	"testing"
)

func setupTestEngine(t *testing.T) *Engine {
	t.Helper()

	e := NewEngineForCLI()
	_, err := e.LoadSchemaFromString(`
		entity User {
			id: uuid primary,
			email: string unique,
			name: string,
			age: int nullable,
			orders: [Order] via user_id,
		}

		entity Order {
			id: uuid primary,
			total: decimal,
			status: string,
			user_id: uuid,
			user: User,
			items: [OrderItem] via order_id,
		}

		entity OrderItem {
			id: uuid primary,
			quantity: int,
			price: decimal,
			order_id: uuid,
			order: Order,
		}
	`)
	if err != nil {
		t.Fatalf("Failed to load test schema: %v", err)
	}
	return e
}

func TestQueryBuilder_SimpleQuery(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	assertContains(t, result.MainQuery, "SELECT")
	assertContains(t, result.MainQuery, "FROM users")
}

func TestQueryBuilder_FilterEquality(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Filter("email", "eq", "ana@mail.com").
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	assertContains(t, result.MainQuery, "WHERE")
	assertContains(t, result.MainQuery, "email")
	assertContains(t, result.MainQuery, "ana@mail.com")
}

func TestQueryBuilder_MultipleFilters(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Filter("age", "gte", 18).
		Filter("age", "lte", 65).
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	assertContains(t, result.MainQuery, "18")
	assertContains(t, result.MainQuery, "65")
	assertContains(t, result.MainQuery, "AND")
}

func TestQueryBuilder_Include(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Include("orders").
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if len(result.EagerQueries) == 0 {
		t.Fatal("Expected eager queries for include")
	}
}

func TestQueryBuilder_NestedInclude(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Include("orders").
		Include("orders.items").
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if len(result.EagerQueries) < 2 {
		t.Fatalf("Expected 2 eager queries, got %d", len(result.EagerQueries))
	}
}

func TestQueryBuilder_FilterOnRelation(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Filter("orders.total", "gt", 100).
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	assertContains(t, result.MainQuery, "JOIN")
	assertContains(t, result.MainQuery, "DISTINCT")
}

func TestQueryBuilder_OrderByLimitOffset(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		OrderBy("name", "asc").
		Limit(10).
		Offset(20).
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	assertContains(t, result.MainQuery, "ORDER BY")
	assertContains(t, result.MainQuery, "LIMIT 10")
	assertContains(t, result.MainQuery, "OFFSET 20")
}

func TestQueryBuilder_FullQuery(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Filter("age", "gte", 18).
		Filter("orders.total", "gt", 50).
		Include("orders").
		Include("orders.items").
		OrderBy("name", "desc").
		Limit(10).
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	// Main query checks
	assertContains(t, result.MainQuery, "DISTINCT")
	assertContains(t, result.MainQuery, "JOIN")
	assertContains(t, result.MainQuery, "ORDER BY")
	assertContains(t, result.MainQuery, "LIMIT 10")

	// Eager queries
	if len(result.EagerQueries) < 2 {
		t.Fatalf("Expected 2 eager queries, got %d", len(result.EagerQueries))
	}
}

func TestQueryBuilder_NoSchema(t *testing.T) {
	e := NewEngineWithoutSchema() // No schema loaded

	_, err := e.Query("User").ToSQL()
	if err == nil {
		t.Fatal("Expected error when no schema is loaded")
	}
}

// Helper
func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !contains(haystack, needle) {
		t.Errorf("Expected output to contain %q\n\nGot:\n%s", needle, haystack)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
