package engine

/*
import "fmt"

// ============================================================
// DATABASE CONSTRAINT ERRORS
// ============================================================

// PostgresUniqueConstraintError represents a unique constraint violation from PostgreSQL
type PostgresUniqueConstraintError struct {
	Field      string
	Value      interface{}
	Table      string
	Suggestion string
}

func (e *PostgresUniqueConstraintError) Error() string {
	return fmt.Sprintf(
		"unique constraint violation on field '%s' in table '%s'\n"+
			"Value: %v already exists\n"+
			"Suggestion: %s",
		e.Field, e.Table, e.Value, e.Suggestion,
	)
}

// ForeignKeyError represents a foreign key constraint violation
type PostgresForeignKeyError struct {
	Field            string
	Value            interface{}
	ReferencedTable  string
	ReferencedField  string
	ReferencedEntity string
	Suggestion       string
}

func (e *PostgresForeignKeyError) Error() string {
	return fmt.Sprintf(
		"foreign key constraint violation on field '%s'\n"+
			"Value: %v does not exist in %s.%s\n"+
			"Suggestion: %s",
		e.Field, e.Value, e.ReferencedTable, e.ReferencedField, e.Suggestion,
	)
}

// NotNullError represents a NOT NULL constraint violation
type PostgresNotNullError struct {
	Field      string
	Suggestion string
}

func (e *PostgresNotNullError) Error() string {
	return fmt.Sprintf(
		"NOT NULL constraint violation on field '%s'\n"+
			"Suggestion: %s",
		e.Field, e.Suggestion,
	)
}

// ConstraintError represents a generic constraint violation (e.g., CHECK)
type PostgresConstraintError struct {
	Type       string // "check", "exclusion", etc.
	Field      string
	Value      interface{}
	Suggestion string
}

func (e *PostgresConstraintError) Error() string {
	return fmt.Sprintf(
		"%s constraint violation on field '%s'\n"+
			"Value: %v\n"+
			"Suggestion: %s",
		e.Type, e.Field, e.Value, e.Suggestion,
	)
}

// UnknownFieldError represents an attempt to use a field that doesn't exist in schema
type PostgresUnknownFieldError struct {
	Entity    string
	Field     string
	Available []string // Optional: list of valid fields
}

func (e *PostgresUnknownFieldError) Error() string {
	msg := fmt.Sprintf(
		"unknown field '%s' in entity '%s'",
		e.Field, e.Entity,
	)

	if len(e.Available) > 0 {
		msg += fmt.Sprintf("\nAvailable fields: %v", e.Available)
	}

	return msg
}
*/
