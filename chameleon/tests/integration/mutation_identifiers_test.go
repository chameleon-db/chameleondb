package integration

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// requireUnknownField checks the call was rejected by schema validation,
// before any statement was sent to the database.
func requireUnknownField(t *testing.T, err error) {
	t.Helper()

	var unknown *engine.UnknownFieldError
	require.ErrorAs(t, err, &unknown)
}

var unknownMutationFields = []string{
	"missing",
	"",
	"id x",
	`id"`,
	"id'",
	"id;",
	"id--",
	"id/**/",
	"ID",
	"name, email",
	"café",
}

func TestMutationDeleteUnknownFilterFieldExecutesNothing(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	for i, name := range unknownMutationFields {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			// With and without a valid filter next to the bad one.
			_, err := eng.Delete("User").Filter(name, "eq", "x").Execute(ctx)
			requireUnknownField(t, err)

			_, err = eng.Delete("User").
				Filter("email", "eq", "ana@mail.com").
				Filter(name, "eq", "x").
				Execute(ctx)
			requireUnknownField(t, err)

			assertTablesIntact(t, ctx, 3)
		})
	}
}

func TestMutationUpdateUnknownFieldExecutesNothing(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	for i, name := range unknownMutationFields {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			_, err := eng.Update("User").
				Filter(name, "eq", "x").
				Set("name", "Changed").
				Execute(ctx)
			requireUnknownField(t, err)

			_, err = eng.Update("User").
				Filter("email", "eq", "ana@mail.com").
				Set(name, "Changed").
				Execute(ctx)
			requireUnknownField(t, err)

			assertTablesIntact(t, ctx, 3)
		})
	}

	result, err := eng.Query("User").Filter("name", "eq", "Changed").Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Count())
}

func TestMutationInsertUnknownFieldExecutesNothing(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	for i, name := range unknownMutationFields {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			_, err := eng.Insert("User").
				Set("id", uuid.New().String()).
				Set("email", fmt.Sprintf("new%d@mail.com", i)).
				Set("name", "New").
				Set(name, "x").
				Execute(ctx)
			requireUnknownField(t, err)

			assertTablesIntact(t, ctx, 3)
		})
	}
}

func TestMutationValidDeleteStillWorks(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// Single filter.
	result, err := eng.Delete("User").Filter("email", "eq", "charlie@mail.com").Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Affected)
	assert.Equal(t, 2, countRows(t, ctx, "users"))

	// Several filters, different operators.
	_, err = eng.Insert("User").
		Set("id", uuid.New().String()).
		Set("email", "temp@mail.com").
		Set("name", "Temp").
		Set("age", 50).
		Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, countRows(t, ctx, "users"))

	result, err = eng.Delete("User").
		Filter("email", "eq", "temp@mail.com").
		Filter("age", "gte", 50).
		Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Affected)
	assert.Equal(t, 2, countRows(t, ctx, "users"))

	// No match.
	result, err = eng.Delete("User").Filter("email", "eq", "nobody@mail.com").Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Affected)
	assert.Equal(t, 2, countRows(t, ctx, "users"))
}

func TestMutationHostileValuesAreData(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	for i, value := range hostileValues {
		t.Run(fmt.Sprintf("value_%d", i), func(t *testing.T) {
			id := uuid.New().String()
			inserted, err := eng.Insert("User").
				Set("id", id).
				Set("email", fmt.Sprintf("m%d@mail.com", i)).
				Set("name", value).
				Execute(ctx)
			require.NoError(t, err)
			assert.Equal(t, value, inserted.Record["name"])

			updated, err := eng.Update("User").
				Filter("name", "eq", value).
				Set("age", 41).
				Execute(ctx)
			require.NoError(t, err)
			assert.Equal(t, 1, updated.Affected)

			deleted, err := eng.Delete("User").Filter("name", "eq", value).Execute(ctx)
			require.NoError(t, err)
			assert.Equal(t, 1, deleted.Affected)

			assertTablesIntact(t, ctx, 3)
		})
	}
}
