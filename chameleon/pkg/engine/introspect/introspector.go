package introspect

import (
	"context"
	"fmt"
	"strings"
)

// DatabaseType identifies the database engine
type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgresql"
	MySQL      DatabaseType = "mysql"
	SQLite     DatabaseType = "sqlite"
)

// ColumnInfo represents a column
type ColumnInfo struct {
	Name       string
	Type       string // DB-specific type (e.g., "varchar", "integer")
	Nullable   bool
	PrimaryKey bool
	Unique     bool
	DefaultVal *string
	ForeignKey *ForeignKeyInfo
}

// ForeignKeyInfo represents a foreign key constraint
type ForeignKeyInfo struct {
	ReferencedTable  string
	ReferencedColumn string
	ConstraintName   string
}

// TableInfo represents a table structure
type TableInfo struct {
	Name    string
	Columns []ColumnInfo
}

// Introspector is the interface all DB engines must implement
type Introspector interface {
	// Detect confirms this is the right DB type
	Detect(ctx context.Context) (bool, error)

	// ListTables returns all user-defined tables
	ListTables(ctx context.Context) ([]string, error)

	// InspectTable returns detailed structure
	InspectTable(ctx context.Context, tableName string) (*TableInfo, error)

	// GetAllTables returns complete schema
	GetAllTables(ctx context.Context) ([]TableInfo, error)

	// Close closes the connection
	Close() error
}

// NewIntrospector creates the right introspector for a connection string
func NewIntrospector(ctx context.Context, connStr string) (Introspector, error) {
	// Try to detect DB type from connection string
	dbType := detectFromConnString(connStr)

	switch dbType {
	case PostgreSQL:
		return newPostgresIntrospector(ctx, connStr)
	case MySQL:
		return nil, fmt.Errorf("MySQL support coming in v0.2")
	case SQLite:
		return nil, fmt.Errorf("SQLite support coming in v0.2")
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// detectFromConnString identifies DB type from connection string
func detectFromConnString(connStr string) DatabaseType {
	if strings.HasPrefix(connStr, "postgresql://") || strings.HasPrefix(connStr, "postgres://") {
		return PostgreSQL
	}
	if strings.HasPrefix(connStr, "mysql://") {
		return MySQL
	}
	if strings.HasPrefix(connStr, "sqlite://") || strings.HasPrefix(connStr, "file:") {
		return SQLite
	}
	// Default to PostgreSQL (v0.1 only supports PostgreSQL)
	return PostgreSQL
}
