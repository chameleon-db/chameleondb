package introspect

import (
	"context"
	"testing"
)

func TestDetectFromConnString(t *testing.T) {
	tests := []struct {
		name    string
		connStr string
		want    DatabaseType
	}{
		{name: "postgresql scheme", connStr: "postgresql://user:pass@localhost/db", want: PostgreSQL},
		{name: "postgres alias upper case", connStr: "  POSTGRES://user:pass@localhost/db  ", want: PostgreSQL},
		{name: "postgres dsn format", connStr: "host=localhost port=5432 dbname=test user=postgres password=postgres sslmode=disable", want: PostgreSQL},
		{name: "mysql scheme", connStr: "mysql://user:pass@localhost/db", want: MySQL},
		{name: "sqlite scheme", connStr: "sqlite:///tmp/test.db", want: SQLite},
		{name: "sqlite file scheme", connStr: "file:/tmp/test.db", want: SQLite},
		{name: "unknown scheme", connStr: "sqlserver://localhost", want: Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectFromConnString(tt.connStr)
			if got != tt.want {
				t.Fatalf("detectFromConnString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewIntrospectorRejectsInvalidConnectionString(t *testing.T) {
	ctx := context.Background()

	if _, err := NewIntrospector(ctx, "   "); err == nil {
		t.Fatal("expected error for empty connection string")
	}

	if _, err := NewIntrospector(ctx, "sqlserver://localhost"); err == nil {
		t.Fatal("expected error for unsupported connection scheme")
	}
}
