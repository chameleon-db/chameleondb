package integration

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// setupTypedDB loads the typed fixture schema and creates its table by hand,
// so the column types (including timestamptz) are exactly the ones listed in
// the DDL below.
func setupTypedDB(t *testing.T) (*engine.Engine, context.Context, func()) {
	t.Helper()

	ctx := context.Background()
	eng := engine.NewEngineForCLI()
	content, err := os.ReadFile("../fixtures/typed_schema.cham")
	require.NoError(t, err)
	_, err = eng.LoadSchemaFromString(string(content))
	require.NoError(t, err)
	require.NoError(t, eng.Connect(ctx, testConfig()))

	conn, err := pgx.Connect(ctx, testConfig().ConnectionString())
	require.NoError(t, err)
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
		DROP TABLE IF EXISTS samples;
		CREATE TABLE samples (
			id uuid PRIMARY KEY,
			label text NOT NULL,
			qty integer,
			amount numeric NOT NULL,
			active boolean NOT NULL,
			ratio double precision,
			seen_at timestamptz
		);
		INSERT INTO samples (id, label, qty, amount, active, ratio, seen_at) VALUES
		('10000000-0000-0000-0000-000000000001', 'alpha', 1, 10.50, true,  0.5, '2024-01-01T00:00:00Z'),
		('10000000-0000-0000-0000-000000000002', 'beta',  2, 20.00, false, 1.5, '2024-06-01T12:00:00Z'),
		('10000000-0000-0000-0000-000000000003', 'gamma', 3, 30.25, true,  2.5, '2025-01-01T00:00:00Z'),
		('10000000-0000-0000-0000-000000000004', 'delta', NULL, 40.00, false, NULL, NULL);
	`)
	require.NoError(t, err)

	cleanup := func() {
		c, err := pgx.Connect(ctx, testConfig().ConnectionString())
		if err == nil {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS samples")
			c.Close(ctx)
		}
		eng.Close()
	}
	return eng, ctx, cleanup
}

func labels(result *engine.QueryResult) []string {
	var out []string
	for _, row := range result.Rows {
		out = append(out, row.String("label"))
	}
	return out
}

func TestQueryTypedParametersAcrossColumnTypes(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTypedDB(t)
	defer cleanup()

	cases := []struct {
		name   string
		field  string
		op     string
		value  interface{}
		expect []string
	}{
		// uuid
		{"uuid eq", "id", "eq", "10000000-0000-0000-0000-000000000002", []string{"beta"}},
		{"uuid neq", "id", "neq", "10000000-0000-0000-0000-000000000002", []string{"alpha", "gamma", "delta"}},
		// text
		{"text eq", "label", "eq", "alpha", []string{"alpha"}},
		{"text like", "label", "like", "a", []string{"alpha", "beta", "gamma", "delta"}},
		{"text like escaped", "label", "like", "%", nil},
		// boolean
		{"bool eq true", "active", "eq", true, []string{"alpha", "gamma"}},
		{"bool eq false", "active", "eq", false, []string{"beta", "delta"}},
		// numeric
		{"numeric gt float", "amount", "gt", 20.0, []string{"gamma", "delta"}},
		{"numeric gt int", "amount", "gt", 20, []string{"gamma", "delta"}},
		{"numeric eq int64", "amount", "eq", int64(20), []string{"beta"}},
		{"numeric eq fractional", "amount", "eq", 10.5, []string{"alpha"}},
		// integer
		{"int gte int", "qty", "gte", 2, []string{"beta", "gamma"}},
		// double precision
		{"float8 lt float", "ratio", "lt", 2.0, []string{"alpha", "beta"}},
		{"float8 gt int", "ratio", "gt", 1, []string{"beta", "gamma"}},
		// timestamptz
		{"timestamptz lt string", "seen_at", "lt", "2024-03-01T00:00:00Z", []string{"alpha"}},
		{"timestamptz gte string", "seen_at", "gte", "2024-03-01T00:00:00Z", []string{"beta", "gamma"}},
		{"timestamptz eq string", "seen_at", "eq", "2024-01-01 00:00:00+00", []string{"alpha"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := eng.Query("Sample").Filter(tc.field, tc.op, tc.value).Execute(ctx)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.expect, labels(result))
		})
	}
}

// A float filter on an integer column. The old generator wrote the number
// into the SQL text, so PostgreSQL compared it as numeric.
func TestQueryFloatFilterOnIntColumn(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTypedDB(t)
	defer cleanup()

	cases := []struct {
		op     string
		value  float64
		expect []string
	}{
		{"lt", 2.5, []string{"alpha", "beta"}},
		{"lte", 2.5, []string{"alpha", "beta"}},
		{"gt", 1.5, []string{"beta", "gamma"}},
		{"gte", 2.0, []string{"beta", "gamma"}},
		{"eq", 2.0, []string{"beta"}},
		{"neq", 2.0, []string{"alpha", "gamma"}},
		{"gt", 2.5, []string{"gamma"}},
	}

	for _, tc := range cases {
		result, err := eng.Query("Sample").Filter("qty", tc.op, tc.value).Execute(ctx)
		require.NoError(t, err, "%s %v", tc.op, tc.value)
		assert.ElementsMatch(t, tc.expect, labels(result), "%s %v", tc.op, tc.value)
	}
}

func TestQueryIncludeWithZeroMainRows(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	result, err := eng.Query("User").
		Filter("age", "gt", 1000).
		Include("orders").
		Include("orders.items").
		Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Count())
	assert.Empty(t, result.Relations["orders"])
	assert.Empty(t, result.Relations["orders.items"])

	// Same on an empty table.
	conn, err := pgx.Connect(ctx, testConfig().ConnectionString())
	require.NoError(t, err)
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, "TRUNCATE order_items, orders, users CASCADE")
	require.NoError(t, err)

	result, err = eng.Query("User").Include("orders").Include("orders.items").Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Count())
	assert.Empty(t, result.Relations["orders"])
	assert.Empty(t, result.Relations["orders.items"])
}
