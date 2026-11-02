package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// hostileValues are value classes that must always be treated as data.
var hostileValues = []string{
	"o'brien",
	"''",
	"a'b'c",
	"x; y",
	";",
	"a -- b",
	"/* c */",
	`back\slash`,
	`\'`,
	"café 中文 \U0001F600",
	"100%",
	"a_b",
	"%_%",
	"line\nbreak\ttab",
	"$1",
	`"quoted"`,
}

func countRows(t *testing.T, ctx context.Context, table string) int {
	t.Helper()

	conn, err := pgx.Connect(ctx, testConfig().ConnectionString())
	require.NoError(t, err)
	defer conn.Close(ctx)

	var n int
	// table is a fixed name chosen by the test, never user input
	require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s", table)).Scan(&n))
	return n
}

func insertUserNamed(t *testing.T, ctx context.Context, id, email, name string) {
	t.Helper()

	conn, err := pgx.Connect(ctx, testConfig().ConnectionString())
	require.NoError(t, err)
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx,
		"INSERT INTO users (id, email, name, age, created_at) VALUES ($1, $2, $3, NULL, NOW())",
		id, email, name)
	require.NoError(t, err)
}

func assertTablesIntact(t *testing.T, ctx context.Context, users int) {
	t.Helper()

	assert.Equal(t, users, countRows(t, ctx, "users"))
	assert.Equal(t, 3, countRows(t, ctx, "orders"))
	assert.Equal(t, 3, countRows(t, ctx, "order_items"))
}

func TestQueryHostileValuesAreData(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// One user per hostile value, so a correct match returns exactly that row.
	for i, value := range hostileValues {
		insertUserNamed(t, ctx,
			fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
			fmt.Sprintf("hostile%d@mail.com", i),
			value)
	}
	total := 3 + len(hostileValues)

	for i, value := range hostileValues {
		t.Run(fmt.Sprintf("value_%d", i), func(t *testing.T) {
			// Equality returns exactly the row holding the value.
			result, err := eng.Query("User").Filter("name", "eq", value).Execute(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, result.Count())
			assert.Equal(t, value, result.Rows[0].String("name"))

			// Inequality returns every other row.
			result, err = eng.Query("User").Filter("name", "neq", value).Execute(ctx)
			require.NoError(t, err)
			assert.Equal(t, total-1, result.Count())

			// The same value used on a relation filter matches nothing.
			result, err = eng.Query("User").Filter("orders.status", "eq", value).Execute(ctx)
			require.NoError(t, err)
			assert.Equal(t, 0, result.Count())

			// Combined with eager loading.
			result, err = eng.Query("User").
				Filter("name", "eq", value).
				Include("orders").
				Execute(ctx)
			require.NoError(t, err)
			assert.Equal(t, 1, result.Count())

			assertTablesIntact(t, ctx, total)
		})
	}
}

func TestQueryHostileValuesOnOtherColumns(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	for i, value := range hostileValues {
		t.Run(fmt.Sprintf("value_%d", i), func(t *testing.T) {
			// Text columns: no row matches.
			for _, field := range []string{"email", "name"} {
				result, err := eng.Query("User").Filter(field, "eq", value).Execute(ctx)
				require.NoError(t, err)
				assert.Equal(t, 0, result.Count())
			}

			assertTablesIntact(t, ctx, 3)
		})
	}
}

func TestQueryUnknownIdentifiersExecuteNothing(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	bad := []string{
		"missing",
		"",
		"name x",
		`name"`,
		"na'me",
		"name;",
		"name--",
		"name/**/",
		"name, email",
		"NAME",
	}

	cases := map[string]func(name string) *engine.QueryBuilder{
		"select": func(name string) *engine.QueryBuilder {
			return eng.Query("User").Select("id", name)
		},
		"filter": func(name string) *engine.QueryBuilder {
			return eng.Query("User").Filter(name, "eq", 1)
		},
		"nested relation": func(name string) *engine.QueryBuilder {
			return eng.Query("User").Filter(name+".total", "eq", 1)
		},
		"nested field": func(name string) *engine.QueryBuilder {
			return eng.Query("User").Filter("orders."+name, "eq", 1)
		},
		"order_by": func(name string) *engine.QueryBuilder {
			return eng.Query("User").OrderBy(name, "asc")
		},
		"include": func(name string) *engine.QueryBuilder {
			return eng.Query("User").Include(name)
		},
		"nested include": func(name string) *engine.QueryBuilder {
			return eng.Query("User").Include("orders." + name)
		},
	}

	for kind, build := range cases {
		for _, name := range bad {
			t.Run(fmt.Sprintf("%s/%q", kind, name), func(t *testing.T) {
				// "orders." with an empty segment is dropped by the path
				// splitter and would name a valid path; skip that shape.
				if name == "" && (kind == "nested field" || kind == "nested include" || kind == "nested relation") {
					t.Skip("empty segments are removed before generation")
				}

				_, err := build(name).Execute(ctx)
				require.Error(t, err)

				// Rejected before anything reached the database.
				var pgErr *pgconn.PgError
				assert.False(t, errors.As(err, &pgErr), "database was reached: %v", err)

				assertTablesIntact(t, ctx, 3)
			})
		}
	}
}

func TestQueryUnsupportedPathsAreRejected(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	for _, path := range []string{"orders.items.price", "orders.total.extra"} {
		_, err := eng.Query("User").Filter(path, "eq", 1).Execute(ctx)
		assert.Error(t, err, path)
	}
}

func TestQueryLikeWildcardsMatchLiterally(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	names := []string{
		"100% sure",
		"1000 sure",
		"a_b",
		"axb",
		`back\slash`,
		"backXslash",
		"plain name",
	}
	for i, name := range names {
		insertUserNamed(t, ctx,
			fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
			fmt.Sprintf("like%d@mail.com", i),
			name)
	}

	cases := []struct {
		pattern string
		want    []string
	}{
		{"100%", []string{"100% sure"}},
		{"%", []string{"100% sure"}},
		{"a_b", []string{"a_b"}},
		{"_", []string{"a_b"}},
		{`\`, []string{`back\slash`}},
		{`k\s`, []string{`back\slash`}},
		{`\%`, nil},
		{"sure", []string{"100% sure", "1000 sure"}},
		{"plain", []string{"plain name"}},
		{"%_%", nil},
	}

	for _, tc := range cases {
		t.Run(tc.pattern, func(t *testing.T) {
			result, err := eng.Query("User").
				Filter("name", "like", tc.pattern).
				OrderBy("name", "asc").
				Execute(ctx)
			require.NoError(t, err)

			var got []string
			for _, row := range result.Rows {
				got = append(got, row.String("name"))
			}
			assert.ElementsMatch(t, tc.want, got)
		})
	}
}

func TestQueryIncludeBindsParentIDsAsArray(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// Several parents.
	result, err := eng.Query("User").
		Include("orders").
		Include("orders.items").
		Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Count())
	assert.Len(t, result.Relations["orders"], 3)
	assert.Len(t, result.Relations["orders.items"], 3)

	// One parent.
	result, err = eng.Query("User").
		Filter("email", "eq", "ana@mail.com").
		Include("orders").
		Include("orders.items").
		Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count())
	assert.Len(t, result.Relations["orders"], 2)
	assert.Len(t, result.Relations["orders.items"], 3)

	// No parents: the array is empty and nothing is returned.
	result, err = eng.Query("User").
		Filter("email", "eq", "nobody@mail.com").
		Include("orders").
		Include("orders.items").
		Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Count())
	assert.Empty(t, result.Relations["orders"])
	assert.Empty(t, result.Relations["orders.items"])

	// A relation that is not a parent of anything returned.
	result, err = eng.Query("User").
		Filter("email", "eq", "charlie@mail.com").
		Include("orders").
		Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count())
	assert.Empty(t, result.Relations["orders"])
}

func TestQueryTypedParametersKeepWorking(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	result, err := eng.Query("User").Filter("age", "gte", 30).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count())

	result, err = eng.Query("Order").Filter("total", "gt", 100.0).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Count())

	result, err = eng.Query("User").
		Filter("id", "eq", "11111111-1111-1111-1111-111111111111").
		Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count())

	result, err = eng.Query("User").Filter("orders.total", "gt", 100).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Count())

	// text
	result, err = eng.Query("User").Filter("name", "eq", "Bob Smith").Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count())

	// timestamp, bound as text
	result, err = eng.Query("User").Filter("created_at", "gte", "2000-01-01T00:00:00Z").Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Count())

	// numeric compared with an integer and with a float
	result, err = eng.Query("Order").Filter("total", "gte", 200).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count())
	result, err = eng.Query("Order").Filter("total", "eq", 75.0).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count())

	// float against an integer column keeps its exact value
	result, err = eng.Query("User").Filter("age", "lt", 25.5).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Count())
}
