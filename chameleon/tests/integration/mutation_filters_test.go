package integration

import (
	"context"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func insertUser(t *testing.T, ctx context.Context, eng *engine.Engine, email, name string, age interface{}) string {
	t.Helper()

	id := uuid.New().String()
	insert := eng.Insert("User").
		Set("id", id).
		Set("email", email).
		Set("name", name)
	if age != nil {
		insert = insert.Set("age", age)
	}

	_, err := insert.Execute(ctx)
	require.NoError(t, err)
	return id
}

// A nil filter value matches NULL instead of comparing with "= NULL", which never matches.
func TestMutationFilters_NilMatchesNull(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()
	runMigration(t, eng, ctx)

	insertUser(t, ctx, eng, "with-age@mail.com", "With Age", 30)
	withoutAge := insertUser(t, ctx, eng, "no-age@mail.com", "No Age", nil)

	updated, err := eng.Update("User").
		Filter("age", "eq", nil).
		Set("name", "Age Unknown").
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, updated.Affected)
	require.Equal(t, withoutAge, uuidString(t, updated.Records[0]["id"]))

	deleted, err := eng.Delete("User").
		Filter("age", "neq", nil).
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, deleted.Affected, "only the user with an age is deleted")
}

// like and ilike mean "contains" and treat % and _ as plain characters,
// the same as the query builder.
func TestMutationFilters_LikeIsContainsAndEscaped(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()
	runMigration(t, eng, ctx)

	insertUser(t, ctx, eng, "ana@mail.com", "Ana Lopez", nil)
	insertUser(t, ctx, eng, "joana@mail.com", "Joana", nil)
	insertUser(t, ctx, eng, "pct@mail.com", "100% sure", nil)
	insertUser(t, ctx, eng, "bob@mail.com", "Bob", nil)

	// like is case-sensitive: only "Joana" contains lowercase "ana".
	updated, err := eng.Update("User").
		Filter("name", "like", "ana").
		Set("age", 1).
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, updated.Affected)

	// ilike ignores case: "Ana Lopez" and "Joana".
	updated, err = eng.Update("User").
		Filter("name", "ilike", "ANA").
		Set("age", 2).
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, updated.Affected)

	// A bare % must not match every row.
	deleted, err := eng.Delete("User").
		Filter("name", "like", "%").
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, deleted.Affected, "only the name containing a literal %")

	// _ is a literal underscore, not "any character".
	deleted, err = eng.Delete("User").
		Filter("name", "like", "_").
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, deleted.Affected)
}

// Deleting a parent that still has children reports the dependents,
// not a misleading "referenced record does not exist".
func TestDelete_ParentWithChildrenIsForeignKeyConstraintError(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()
	runMigration(t, eng, ctx)

	userID := insertUser(t, ctx, eng, "author@mail.com", "Author", nil)

	_, err := eng.Insert("Post").
		Set("id", uuid.New().String()).
		Set("title", "First").
		Set("content", "Body").
		Set("author_id", userID).
		Execute(ctx)
	require.NoError(t, err)

	_, err = eng.Delete("User").
		Filter("id", "eq", userID).
		Execute(ctx)
	require.Error(t, err)

	var fkErr *engine.ForeignKeyConstraintError
	require.ErrorAs(t, err, &fkErr, "Error should be ForeignKeyConstraintError")
	require.Equal(t, "User", fkErr.Entity)
	require.Equal(t, "posts", fkErr.DependentTable)
	require.Equal(t, userID, fkErr.ID)
	require.NotEmpty(t, fkErr.Suggestion)

	// After removing the child, the parent can be deleted.
	_, err = eng.Delete("Post").Filter("author_id", "eq", userID).Execute(ctx)
	require.NoError(t, err)

	deleted, err := eng.Delete("User").Filter("id", "eq", userID).Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, deleted.Affected)
}
