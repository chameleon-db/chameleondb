package integration

import (
	"context"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestInsert_UniqueConstraintViolation verifies unique constraint errors are mapped correctly.
// This test validates the error mapping pipeline: PostgreSQL 23505 error code →
// mapUniqueViolation() → extract field from error detail → UniqueConstraintError struct.
func TestInsert_UniqueConstraintViolation(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Verify database is ready for operations
	_, err := eng.Query("User").Execute(ctx)
	require.NoError(t, err, "Query should succeed (database ready check)")

	// Insert first user with unique email
	_, err = eng.Insert("User").
		Set("id", uuid.New().String()).
		Set("email", "duplicate@mail.com").
		Set("name", "First User").
		Debug().
		Execute(ctx)
	require.NoError(t, err, "First insert should succeed")

	// Verify first user exists
	result, err := eng.Query("User").Filter("email", "eq", "duplicate@mail.com").Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Rows), "First user should be queryable")

	// Insert second user with duplicate email - should fail with UniqueConstraintError
	_, err = eng.Insert("User").
		Set("id", uuid.New().String()).
		Set("email", "duplicate@mail.com").
		Set("name", "Second User").
		Debug().
		Execute(ctx)
	require.Error(t, err, "Second insert should fail with UNIQUE violation")

	// Extract and validate UniqueConstraintError details
	var uniqueErr *engine.UniqueConstraintError
	require.ErrorAs(t, err, &uniqueErr, "Error should be UniqueConstraintError")

	// Verify error contains correct field information
	require.Equal(t, "email", uniqueErr.Field, "Error should identify the 'email' field")
	require.Equal(t, "duplicate@mail.com", uniqueErr.Value, "Error should contain the duplicate value")
	require.Equal(t, "User", uniqueErr.Table, "Error should reference the 'User' table")

	// Verify error includes helpful suggestion
	require.NotEmpty(t, uniqueErr.Suggestion, "Error should include a suggestion for resolution")
	require.Contains(t, uniqueErr.Suggestion, "email", "Suggestion should reference the conflicting field")
}

// TestInsert_NotNullViolation verifies NOT NULL constraint errors are mapped correctly.
// Validates error mapping for PostgreSQL 23502 error (not_null_violation).
func TestInsert_NotNullViolation(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Try to insert without required field (email)
	_, err := eng.Insert("User").
		Set("name", "Test User").
		// Missing: email (required field)
		Execute(ctx)

	// Should fail with NotNullError
	require.Error(t, err, "Insert without required 'email' field should fail")

	var notNullErr *engine.NotNullError
	require.ErrorAs(t, err, &notNullErr, "Error should be NotNullError")

	// Verify error identifies the missing field
	require.Equal(t, "email", notNullErr.Field, "Error should identify 'email' as the missing field")

	// Verify error message is helpful
	require.NotEmpty(t, notNullErr.Suggestion, "Error should include a helpful suggestion")
	require.Contains(t, notNullErr.Suggestion, "required", "Suggestion should indicate field is required")
}

// TestInsert_ForeignKeyViolation verifies foreign key constraint errors are mapped correctly.
// Validates error mapping for PostgreSQL 23503 error (foreign_key_violation).
func TestInsert_ForeignKeyViolation(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Try to insert post with non-existent author
	_, err := eng.Insert("Post").
		Set("id", uuid.New().String()).
		Set("title", "Test Post").
		Set("content", "Content").
		Set("author_id", "00000000-0000-0000-0000-000000000999").
		Execute(ctx)

	// Should fail with ForeignKeyError
	require.Error(t, err, "Insert with non-existent foreign key should fail")

	var fkErr *engine.ForeignKeyError
	require.ErrorAs(t, err, &fkErr, "Error should be ForeignKeyError")

	// Verify error identifies the foreign key field
	require.Equal(t, "author_id", fkErr.Field, "Error should identify 'author_id' as the FK field")

	// Verify error includes referenced table information
	require.NotEmpty(t, fkErr.ReferencedTable, "Error should identify the referenced table")
	// ReferencedTable may be a fallback if constraint naming doesn't follow expected pattern

	// Verify helpful suggestion
	require.NotEmpty(t, fkErr.Suggestion, "Error should include a helpful suggestion")
	require.Contains(t, fkErr.Suggestion, "Ensure", "Suggestion should provide guidance on resolution")
}

// TestUpdate_UniqueConstraintViolation verifies UPDATE unique constraint errors are mapped correctly.
// Tests that unique violations are caught during UPDATE operations, not just INSERT.
func TestUpdate_UniqueConstraintViolation(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Insert first user with unique email
	user1ID := uuid.New().String()
	_, err := eng.Insert("User").
		Set("id", user1ID).
		Set("email", "user1@mail.com").
		Set("name", "User 1").
		Execute(ctx)
	require.NoError(t, err, "First user insert should succeed")

	// Insert second user with different email
	user2ID := uuid.New().String()
	_, err = eng.Insert("User").
		Set("id", user2ID).
		Set("email", "user2@mail.com").
		Set("name", "User 2").
		Execute(ctx)
	require.NoError(t, err, "Second user insert should succeed")

	// Try to update user2's email to user1's email (unique constraint violation)
	_, err = eng.Update("User").
		Filter("id", "eq", user2ID).
		Set("email", "user1@mail.com"). // ← Duplicate email
		Execute(ctx)

	// Should fail with UniqueConstraintError
	require.Error(t, err, "Update to duplicate email should fail with UNIQUE violation")

	var uniqueErr *engine.UniqueConstraintError
	require.ErrorAs(t, err, &uniqueErr, "Error should be UniqueConstraintError")

	// Verify error details are correct
	require.Equal(t, "email", uniqueErr.Field, "Error should identify 'email' field")
	require.Equal(t, "user1@mail.com", uniqueErr.Value, "Error should contain the duplicate value")
}

// TestInsert_UndefinedColumn verifies schema validation errors for unknown fields.
// Validates error mapping for PostgreSQL 42703 error (undefined_column).
func TestInsert_UndefinedColumn(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Try to insert with unknown field
	_, err := eng.Insert("User").
		Set("id", uuid.New().String()).
		Set("email", "test@mail.com").
		Set("unknown_field", "value"). // ← Doesn't exist in schema
		Execute(ctx)

	// Should fail with UnknownFieldError
	require.Error(t, err, "Insert with undefined column should fail")

	var unknownFieldErr *engine.UnknownFieldError
	require.ErrorAs(t, err, &unknownFieldErr, "Error should be UnknownFieldError")

	// Verify error identifies the unknown field
	require.Equal(t, "unknown_field", unknownFieldErr.Field, "Error should identify the unknown field")
	require.Equal(t, "User", unknownFieldErr.Entity, "Error should reference the 'User' entity")
}

// TestErrorMessages_AreHelpful verifies that error messages include context and suggestions.
// This tests the entire error generation pipeline, not just error type extraction.
func TestErrorMessages_AreHelpful(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Insert user
	_, err := eng.Insert("User").
		Set("id", uuid.New().String()).
		Set("email", "test@mail.com").
		Set("name", "Test").
		Execute(ctx)
	require.NoError(t, err, "Initial insert should succeed")

	// Try duplicate insert to trigger unique violation
	_, err = eng.Insert("User").
		Set("id", uuid.New().String()).
		Set("email", "test@mail.com").
		Set("name", "Duplicate").
		Execute(ctx)

	require.Error(t, err, "Duplicate insert should fail")

	// Verify error is properly typed
	var uniqueErr *engine.UniqueConstraintError
	require.ErrorAs(t, err, &uniqueErr, "Should extract typed error")

	// Error message should be helpful even when printed directly
	errMsg := uniqueErr.Error()
	require.NotEmpty(t, errMsg, "Error message should not be empty")
	require.Contains(t, errMsg, "email", "Error message should mention the conflicting field")
	require.Contains(t, errMsg, "test@mail.com", "Error message should show the duplicate value")

	// Verify suggestion is actionable
	require.NotEmpty(t, uniqueErr.Suggestion, "Error should include a suggestion")
	require.Contains(t, uniqueErr.Suggestion, "different value", "Suggestion should guide user on resolution")
}
