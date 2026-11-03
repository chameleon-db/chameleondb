package engine

import (
	"strings"
	"testing"
)

func TestGenerateMigration(t *testing.T) {
	eng := NewEngineForCLI()
	_, err := eng.LoadSchemaFromString(`
		entity User {
			id: uuid primary,
			email: string unique,
			name: string,
			orders: [Order] via user_id,
		}

		entity Order {
			id: uuid primary,
			total: decimal,
			user_id: uuid,
			user: User,
		}
	`)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	sql, err := eng.GenerateMigration()
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	assertContains(t, sql, "CREATE TABLE IF NOT EXISTS users")
	assertContains(t, sql, "CREATE TABLE IF NOT EXISTS orders")
	assertContains(t, sql, "FOREIGN KEY (user_id) REFERENCES users(id)")
}

// The generated migration must be create-only: it must never contain
// statements that destroy or alter existing data.
func TestGenerateMigrationIsNonDestructive(t *testing.T) {
	eng := NewEngineForCLI()
	_, err := eng.LoadSchemaFromString(`
		entity User {
			id: uuid primary,
			email: string unique,
			orders: [Order] via user_id,
		}

		entity Order {
			id: uuid primary,
			user_id: uuid,
			user: User,
		}
	`)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	sql, err := eng.GenerateMigration()
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	upper := strings.ToUpper(sql)
	for _, forbidden := range []string{"DROP ", "TRUNCATE", "DELETE ", "ALTER ", "CASCADE"} {
		if strings.Contains(upper, forbidden) {
			t.Errorf("migration must be create-only, found %q in:\n%s", forbidden, sql)
		}
	}
}

func TestGenerateMigrationNoSchema(t *testing.T) {
	eng := NewEngineWithoutSchema()

	_, err := eng.GenerateMigration()
	if err == nil {
		t.Fatal("Expected error when no schema loaded")
	}
}
