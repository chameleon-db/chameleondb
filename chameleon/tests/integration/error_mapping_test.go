package integration

import (
	"context"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInsert_UniqueConstraintViolation verifies unique constraint errors are mapped correctly
func TestInsert_UniqueConstraintViolation(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Insert first user
	_, err := eng.Insert("User").
		Set("email", "duplicate@mail.com").
		Set("name", "First User").
		Execute(ctx)
	require.NoError(t, err)

	// Try to insert duplicate email
	_, err = eng.Insert("User").
		Set("email", "duplicate@mail.com"). // ← Same email
		Set("name", "Second User").
		Execute(ctx)

	// Should fail with UniqueConstraintError
	require.Error(t, err)

	var uniqueErr *engine.UniqueConstraintError
	if assert.ErrorAs(t, err, &uniqueErr) {
		assert.Equal(t, "email", uniqueErr.Field)
		assert.Equal(t, "duplicate@mail.com", uniqueErr.Value)
		assert.Contains(t, uniqueErr.Error(), "unique constraint")
	}
}

// TestInsert_NotNullViolation verifies NOT NULL errors are mapped correctly
func TestInsert_NotNullViolation(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Try to insert without required field (email)
	_, err := eng.Insert("User").
		Set("name", "Test User").
		// Missing: email (required)
		Execute(ctx)

	// Should fail with NotNullError
	require.Error(t, err)

	var notNullErr *engine.NotNullError
	if assert.ErrorAs(t, err, &notNullErr) {
		assert.Equal(t, "email", notNullErr.Field)
		assert.Contains(t, notNullErr.Error(), "NOT NULL")
	}
}

// TestInsert_ForeignKeyViolation verifies FK errors are mapped correctly
func TestInsert_ForeignKeyViolation(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Try to insert post with non-existent author
	_, err := eng.Insert("Post").
		Set("title", "Test Post").
		Set("content", "Content").
		Set("author_id", "00000000-0000-0000-0000-000000000999").
		Execute(ctx)

	// Should fail with ForeignKeyError
	require.Error(t, err)

	var fkErr *engine.ForeignKeyError
	if assert.ErrorAs(t, err, &fkErr) {
		assert.Equal(t, "author_id", fkErr.Field)
		assert.Contains(t, fkErr.Error(), "foreign key")
	}
}

// TestUpdate_UniqueConstraintViolation verifies UPDATE unique constraint errors
func TestUpdate_UniqueConstraintViolation(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Insert two users
	_, err := eng.Insert("User").
		Set("email", "user1@mail.com").
		Set("name", "User 1").
		Execute(ctx)
	require.NoError(t, err)

	result2, err := eng.Insert("User").
		Set("email", "user2@mail.com").
		Set("name", "User 2").
		Execute(ctx)
	require.NoError(t, err)

	// Try to update user2's email to user1's email (duplicate)
	_, err = eng.Update("User").
		Filter("id", "eq", result2.ID).
		Set("email", "user1@mail.com"). // ← Duplicate
		Execute(ctx)

	// Should fail with UniqueConstraintError
	require.Error(t, err)

	var uniqueErr *engine.UniqueConstraintError
	assert.ErrorAs(t, err, &uniqueErr)
}

// TestInsert_UndefinedColumn verifies unknown field errors
func TestInsert_UndefinedColumn(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Try to insert with unknown field
	_, err := eng.Insert("User").
		Set("email", "test@mail.com").
		Set("unknown_field", "value"). // ← Doesn't exist
		Execute(ctx)

	// Should fail with UnknownFieldError
	require.Error(t, err)

	var unknownFieldErr *engine.UnknownFieldError
	if assert.ErrorAs(t, err, &unknownFieldErr) {
		assert.Equal(t, "unknown_field", unknownFieldErr.Field)
	}
}

// TestErrorMessages_AreHelpful verifies error messages include suggestions
func TestErrorMessages_AreHelpful(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	// Insert user
	eng.Insert("User").
		Set("email", "test@mail.com").
		Set("name", "Test").
		Execute(ctx)

	// Try duplicate
	_, err := eng.Insert("User").
		Set("email", "test@mail.com").
		Set("name", "Duplicate").
		Execute(ctx)

	require.Error(t, err)

	// Error message should be helpful
	errMsg := err.Error()
	assert.Contains(t, errMsg, "unique constraint", "Should mention constraint type")
	assert.Contains(t, errMsg, "email", "Should mention field name")
	assert.Contains(t, errMsg, "Suggestion", "Should include suggestion")
}
