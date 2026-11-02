package engine

import (
	"encoding/json"
	"strings"
	"testing"
)

func setupTestEngine(t *testing.T) *Engine {
	t.Helper()

	e := NewEngineForCLI()
	_, err := e.LoadSchemaFromString(`
		entity User {
			id: uuid primary,
			email: string unique,
			name: string,
			age: int nullable,
			orders: [Order] via user_id,
		}

		entity Order {
			id: uuid primary,
			total: decimal,
			status: string,
			user_id: uuid,
			user: User,
			items: [OrderItem] via order_id,
		}

		entity OrderItem {
			id: uuid primary,
			quantity: int,
			price: decimal,
			order_id: uuid,
			order: Order,
		}
	`)
	if err != nil {
		t.Fatalf("Failed to load test schema: %v", err)
	}
	return e
}

func TestQueryBuilder_SimpleQuery(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	assertContains(t, result.MainQuery, "SELECT")
	assertContains(t, result.MainQuery, `FROM "users"`)
}

func TestQueryBuilder_FilterEquality(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Filter("email", "eq", "ana@mail.com").
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	assertContains(t, result.MainQuery, "WHERE")
	assertContains(t, result.MainQuery, `"email" = $1`)
	if strings.Contains(result.MainQuery, "ana@mail.com") {
		t.Errorf("value must not appear in SQL text: %s", result.MainQuery)
	}
	if len(result.MainParams) != 1 || result.MainParams[0] != (SQLParam{Kind: ParamString, Value: "ana@mail.com"}) {
		t.Errorf("unexpected params: %#v", result.MainParams)
	}
}

func TestQueryBuilder_MultipleFilters(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Filter("age", "gte", 18).
		Filter("age", "lte", 65).
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	assertContains(t, result.MainQuery, "$1")
	assertContains(t, result.MainQuery, "$2")
	if len(result.MainParams) != 2 {
		t.Fatalf("expected 2 params, got %d", len(result.MainParams))
	}
	assertContains(t, result.MainQuery, "AND")
}

func TestQueryBuilder_Include(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Include("orders").
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if len(result.EagerQueries) == 0 {
		t.Fatal("Expected eager queries for include")
	}
}

func TestQueryBuilder_NestedInclude(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Include("orders").
		Include("orders.items").
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if len(result.EagerQueries) < 2 {
		t.Fatalf("Expected 2 eager queries, got %d", len(result.EagerQueries))
	}
}

func TestQueryBuilder_FilterOnRelation(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Filter("orders.total", "gt", 100).
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	assertContains(t, result.MainQuery, "JOIN")
	assertContains(t, result.MainQuery, "DISTINCT")
}

func TestQueryBuilder_OrderByLimitOffset(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		OrderBy("name", "asc").
		Limit(10).
		Offset(20).
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	assertContains(t, result.MainQuery, "ORDER BY")
	assertContains(t, result.MainQuery, "LIMIT 10")
	assertContains(t, result.MainQuery, "OFFSET 20")
}

func TestQueryBuilder_FullQuery(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").
		Filter("age", "gte", 18).
		Filter("orders.total", "gt", 50).
		Include("orders").
		Include("orders.items").
		OrderBy("name", "desc").
		Limit(10).
		ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	// Main query checks
	assertContains(t, result.MainQuery, "DISTINCT")
	assertContains(t, result.MainQuery, "JOIN")
	assertContains(t, result.MainQuery, "ORDER BY")
	assertContains(t, result.MainQuery, "LIMIT 10")

	// Eager queries
	if len(result.EagerQueries) < 2 {
		t.Fatalf("Expected 2 eager queries, got %d", len(result.EagerQueries))
	}
}

func TestQueryBuilder_NoSchema(t *testing.T) {
	e := NewEngineWithoutSchema() // No schema loaded

	_, err := e.Query("User").ToSQL()
	if err == nil {
		t.Fatal("Expected error when no schema is loaded")
	}
}

// Helper
func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !contains(haystack, needle) {
		t.Errorf("Expected output to contain %q\n\nGot:\n%s", needle, haystack)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestQueryBuilder_ValuesAreNotInSQLText(t *testing.T) {
	e := setupTestEngine(t)

	values := []string{
		"o'brien", "''", "x; y", "a -- b", "/* c */", `back\slash`,
		"café 中文", "100%", "a_b", "line\nbreak",
	}

	for _, value := range values {
		result, err := e.Query("User").
			Filter("name", "eq", value).
			Filter("orders.status", "neq", value).
			ToSQL()
		if err != nil {
			t.Fatalf("ToSQL failed for %q: %v", value, err)
		}

		if strings.Contains(result.MainQuery, value) {
			t.Errorf("value %q appears in SQL text %q", value, result.MainQuery)
		}
		if len(result.MainParams) != 2 || result.MainParams[0] != NewSQLParam(value) || result.MainParams[1] != NewSQLParam(value) {
			t.Errorf("unexpected params for %q: %#v", value, result.MainParams)
		}
	}
}

func TestQueryBuilder_UnknownIdentifiersAreRejected(t *testing.T) {
	e := setupTestEngine(t)

	builders := map[string]*QueryBuilder{
		"select":   e.Query("User").Select("id", "missing"),
		"filter":   e.Query("User").Filter("missing", "eq", 1),
		"relation": e.Query("User").Filter("missing.total", "eq", 1),
		"nested":   e.Query("User").Filter("orders.missing", "eq", 1),
		"order_by": e.Query("User").OrderBy("missing", "asc"),
		"include":  e.Query("User").Include("orders.missing"),
	}

	for name, qb := range builders {
		if result, err := qb.ToSQL(); err == nil {
			t.Errorf("%s: expected error, got SQL %q", name, result.MainQuery)
		}
	}
}

func TestQueryBuilder_EagerQueriesCarryParentIDParameter(t *testing.T) {
	e := setupTestEngine(t)

	result, err := e.Query("User").Include("orders.items").ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if len(result.EagerQueries) != 2 || len(result.EagerParams) != 2 {
		t.Fatalf("expected 2 eager queries with params, got %d and %d", len(result.EagerQueries), len(result.EagerParams))
	}
	for i, params := range result.EagerParams {
		assertContains(t, result.EagerQueries[i][1], "= ANY($1)")
		if len(params) != 1 || params[0].Kind != ParamParentIDs {
			t.Errorf("eager query %d: unexpected params %#v", i, params)
		}
	}
}

func TestSQLParamJSON(t *testing.T) {
	input := `[{"String":"x"},{"Int":9007199254740993},{"Float":1.5},{"Bool":true},"Null","ParentIds"]`

	var params []SQLParam
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	want := []SQLParam{
		{Kind: ParamString, Value: "x"},
		{Kind: ParamInt, Value: int64(9007199254740993)},
		{Kind: ParamFloat, Value: 1.5},
		{Kind: ParamBool, Value: true},
		{Kind: ParamNull},
		{Kind: ParamParentIDs},
	}
	if len(params) != len(want) {
		t.Fatalf("expected %d params, got %d", len(want), len(params))
	}
	for i := range want {
		if params[i] != want[i] {
			t.Errorf("param %d: got %#v, want %#v", i, params[i], want[i])
		}
	}

	out, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var again []SQLParam
	if err := json.Unmarshal(out, &again); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}
	for i := range want {
		if again[i] != want[i] {
			t.Errorf("round trip %d: got %#v, want %#v", i, again[i], want[i])
		}
	}
}

func TestSQLParamJSONRejectsInvalid(t *testing.T) {
	for _, input := range []string{`"Other"`, `{"Other":1}`, `{"Int":"x"}`, `{"String":"a","Int":1}`, `5`, `{}`} {
		var p SQLParam
		if err := json.Unmarshal([]byte(input), &p); err == nil {
			t.Errorf("expected error for %s", input)
		}
	}
}

func TestGeneratedSQLWithoutParamsStillParses(t *testing.T) {
	var g GeneratedSQL
	if err := json.Unmarshal([]byte(`{"main_query":"SELECT 1","eager_queries":[]}`), &g); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if g.MainQuery != "SELECT 1" || len(g.MainParams) != 0 || len(g.EagerParams) != 0 {
		t.Errorf("unexpected result: %#v", g)
	}
}
