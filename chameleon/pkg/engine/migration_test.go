package engine

import (
	"testing"
)

func TestGenerateMigration(t *testing.T) {
	eng := NewEngine()
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

	assertContains(t, sql, "CREATE TABLE users")
	assertContains(t, sql, "CREATE TABLE orders")
	assertContains(t, sql, "FOREIGN KEY (user_id) REFERENCES users(id)")
}

func TestGenerateMigrationNoSchema(t *testing.T) {
	eng := NewEngine()

	_, err := eng.GenerateMigration()
	if err == nil {
		t.Fatal("Expected error when no schema loaded")
	}
}
