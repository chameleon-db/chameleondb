package mutation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/jackc/pgx/v5/pgconn"
)

// mapDatabaseError converts PostgreSQL errors to ChameleonDB error types
// Returns the original error if it's not a PostgreSQL error or unknown type
func mapDatabaseError(err error, entity string, operation string, values map[string]interface{}) error {
	if err == nil {
		return nil
	}

	// Try to extract PostgreSQL error
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		// Not a PostgreSQL error, return as-is
		return fmt.Errorf("%s failed: %w", operation, err)
	}

	// Map based on PostgreSQL error code
	// See: https://www.postgresql.org/docs/current/errcodes-appendix.html
	switch pgErr.Code {
	case "23505": // unique_violation
		return mapUniqueViolation(pgErr, entity, values)

	case "23503": // foreign_key_violation
		return mapForeignKeyViolation(pgErr, entity, values)

	case "23502": // not_null_violation
		return mapNotNullViolation(pgErr, entity, values)

	case "23514": // check_violation
		return mapCheckViolation(pgErr, entity, values)

	case "42P01": // undefined_table
		return &engine.ValidationError{
			Field:    entity,
			Type:     "undefined_table",
			Value:    entity,
			Expected: "valid table name",
			Message:  fmt.Sprintf("Table for entity '%s' does not exist. Run migrations first.", entity),
		}

	case "42703": // undefined_column
		return mapUndefinedColumn(pgErr, entity, values)

	default:
		// Unknown PostgreSQL error, return with context
		return fmt.Errorf("%s failed: %s (code: %s)", operation, pgErr.Message, pgErr.Code)
	}
}

// mapUniqueViolation handles unique constraint violations
func mapUniqueViolation(pgErr *pgconn.PgError, entity string, values map[string]interface{}) error {
	// Extract constraint name and field from error details
	// Detail format: "Key (email)=(test@mail.com) already exists."
	field := extractFieldFromDetail(pgErr.Detail)
	value := extractValueForField(field, values)

	return &engine.UniqueConstraintError{
		Field:      field,
		Value:      value,
		Table:      entity,
		Suggestion: fmt.Sprintf("Use a different value for %s, or update the existing record", field),
	}
}

// mapForeignKeyViolation handles foreign key constraint violations
func mapForeignKeyViolation(pgErr *pgconn.PgError, entity string, values map[string]interface{}) error {
	// Detail format: "Key (author_id)=(uuid-999) is not present in table "users"."
	field := extractFieldFromDetail(pgErr.Detail)
	value := extractValueForField(field, values)

	// Try to extract referenced table from constraint name
	// Constraint name format: fk_posts_author_id_users
	referencedTable := extractReferencedTable(pgErr.ConstraintName)

	return &engine.ForeignKeyError{
		Field:            field,
		Value:            value,
		ReferencedTable:  referencedTable,
		ReferencedField:  "id", // Usually the PK
		ReferencedEntity: referencedTable,
		Suggestion:       fmt.Sprintf("Ensure the referenced %s exists before creating this %s", referencedTable, entity),
	}
}

// mapNotNullViolation handles NOT NULL constraint violations
func mapNotNullViolation(pgErr *pgconn.PgError, entity string, values map[string]interface{}) error {
	// Column name is in pgErr.ColumnName
	field := pgErr.ColumnName
	if field == "" {
		// Fallback to parsing message
		field = extractFieldFromMessage(pgErr.Message)
	}

	return &engine.NotNullError{
		Field:      field,
		Suggestion: fmt.Sprintf("Provide a value for %s (this field is required)", field),
	}
}

// mapCheckViolation handles CHECK constraint violations
func mapCheckViolation(pgErr *pgconn.PgError, entity string, values map[string]interface{}) error {
	return &engine.ConstraintError{
		Type:       "check",
		Field:      extractFieldFromMessage(pgErr.Message),
		Value:      nil,
		Suggestion: fmt.Sprintf("Value violates check constraint: %s", pgErr.ConstraintName),
	}
}

// mapUndefinedColumn handles undefined column errors
func mapUndefinedColumn(pgErr *pgconn.PgError, entity string, values map[string]interface{}) error {
	// Message format: 'column "unknown_field" of relation "users" does not exist'
	field := extractFieldFromMessage(pgErr.Message)

	return &engine.UnknownFieldError{
		Entity:    entity,
		Field:     field,
		Available: nil, // Could be populated from schema if needed
	}
}

// ============================================================
// HELPER FUNCTIONS - Extract info from PostgreSQL errors
// ============================================================

// extractFieldFromDetail extracts field name from error detail
// Input: "Key (email)=(test@mail.com) already exists."
// Output: "email"
func extractFieldFromDetail(detail string) string {
	if detail == "" {
		return ""
	}

	// Look for pattern: (field_name)
	start := strings.Index(detail, "(")
	end := strings.Index(detail, ")")
	if start >= 0 && end > start {
		return detail[start+1 : end]
	}

	return ""
}

// extractValueForField gets the value from values map for a field
func extractValueForField(field string, values map[string]interface{}) interface{} {
	if field == "" {
		return nil
	}
	return values[field]
}

// extractReferencedTable tries to extract referenced table from constraint name
// Input: "fk_posts_author_id_users"
// Output: "users"
func extractReferencedTable(constraintName string) string {
	if constraintName == "" {
		return "referenced_table"
	}

	// Common pattern: fk_{table}_{field}_{referenced_table}
	parts := strings.Split(constraintName, "_")
	if len(parts) >= 4 && parts[0] == "fk" {
		return parts[len(parts)-1]
	}

	return "referenced_table"
}

// extractFieldFromMessage extracts field name from error message
// Input: 'column "unknown_field" of relation "users" does not exist'
// Output: "unknown_field"
func extractFieldFromMessage(message string) string {
	if message == "" {
		return ""
	}

	// Look for quoted field name
	start := strings.Index(message, `"`)
	if start >= 0 {
		end := strings.Index(message[start+1:], `"`)
		if end >= 0 {
			return message[start+1 : start+1+end]
		}
	}

	return ""
}
