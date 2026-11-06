package mutation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// ============================================================
// TEST HELPERS
// ============================================================

// testSchema creates a test schema for unit tests
func testSchema() *engine.Schema {
	schema := &engine.Schema{
		Entities: []*engine.Entity{
			{
				Name: "User",
				Fields: map[string]*engine.Field{
					"id": {
						Name:       "id",
						Type:       engine.FieldType{Kind: "UUID"},
						Nullable:   false,
						Unique:     false,
						PrimaryKey: true,
						Default:    nil,
						Backend:    nil,
					},
					"email": {
						Name:       "email",
						Type:       engine.FieldType{Kind: "String"},
						Nullable:   false,
						Unique:     true,
						PrimaryKey: false,
						Default:    nil,
						Backend:    nil,
					},
					"name": {
						Name:       "name",
						Type:       engine.FieldType{Kind: "String"},
						Nullable:   false,
						Unique:     false,
						PrimaryKey: false,
						Default:    nil,
						Backend:    nil,
					},
					"age": {
						Name:       "age",
						Type:       engine.FieldType{Kind: "Int"},
						Nullable:   true,
						Unique:     false,
						PrimaryKey: false,
						Default:    nil,
						Backend:    nil,
					},
				},
				Relations: map[string]*engine.Relation{},
			},
		},
	}
	return schema
}

// mockConnector returns nil - these are unit tests that don't execute against DB
// For tests that need DB, use integration tests in tests/integration/
func mockConnector() *engine.Connector {
	return nil
}

// ============================================================
// INSERT BUILDER TESTS
// ============================================================

func TestInsertBuilder_Set(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, mockConnector(), "User")

	// Test chainable API
	result := builder.Set("email", "ana@mail.com").Set("name", "Ana")

	// Should return InsertMutation interface (for chaining)
	if result == nil {
		t.Error("Set() should return InsertMutation for chaining")
	}

	// Verify values were set (access internal field for testing)
	if builder.values["email"] != "ana@mail.com" {
		t.Errorf("Expected email='ana@mail.com', got '%v'", builder.values["email"])
	}

	if builder.values["name"] != "Ana" {
		t.Errorf("Expected name='Ana', got '%v'", builder.values["name"])
	}
}

func TestInsertBuilder_Set_MultipleValues(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, mockConnector(), "User")

	// Set same field twice (last one should win)
	builder.Set("name", "Ana").Set("name", "Ana María")

	if builder.values["name"] != "Ana María" {
		t.Errorf("Expected last value to win, got '%v'", builder.values["name"])
	}
}

func TestInsertBuilder_Debug(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, mockConnector(), "User")

	result := builder.Debug()

	if result == nil {
		t.Error("Debug() should return InsertMutation for chaining")
	}

	// Verify debug was enabled
	if builder.debugLevel == nil {
		t.Error("Debug() should set debugLevel")
	}

	if *builder.debugLevel != engine.DebugSQL {
		t.Errorf("Expected DebugSQL level, got %v", *builder.debugLevel)
	}
}

func TestInsertBuilder_Chaining(t *testing.T) {
	schema := testSchema()

	// Test full chain: Set -> Set -> Set -> Debug
	ib := NewInsertBuilder(schema, mockConnector(), "User")
	builder := ib.
		Set("email", "ana@mail.com").
		Set("name", "Ana").
		Set("age", 28).
		Debug()

	if builder == nil {
		t.Error("Chaining should work")
	}

	// Verify values
	if ib.values["email"] != "ana@mail.com" {
		t.Error("Email not set via chaining")
	}
	if ib.values["name"] != "Ana" {
		t.Error("Name not set via chaining")
	}
	if ib.values["age"] != 28 {
		t.Error("Age not set via chaining")
	}
}

// ============================================================
// UPDATE BUILDER TESTS
// ============================================================

func TestUpdateBuilder_Set(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, mockConnector(), "User")

	result := builder.Set("name", "Ana")

	if result == nil {
		t.Error("Set() should return UpdateMutation for chaining")
	}

	if builder.updates["name"] != "Ana" {
		t.Errorf("Expected name='Ana', got '%v'", builder.updates["name"])
	}
}

func TestUpdateBuilder_Filter(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, mockConnector(), "User")

	result := builder.Filter("id", "eq", "uuid-123")

	if result == nil {
		t.Error("Filter() should return UpdateMutation for chaining")
	}

	if len(builder.filters) == 0 {
		t.Error("Filter() should add filter")
	}
}

func TestUpdateBuilder_Filter_And_Set(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, mockConnector(), "User")

	builder.Filter("id", "eq", "uuid-123").
		Set("name", "Ana").
		Set("age", 30)

	if len(builder.filters) == 0 {
		t.Error("Should have filters")
	}

	if len(builder.updates) != 2 {
		t.Errorf("Expected 2 updates, got %d", len(builder.updates))
	}
}

func TestUpdateBuilder_MultipleFilters(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, mockConnector(), "User")

	builder.Filter("id", "eq", "uuid-123").
		Filter("email", "eq", "ana@mail.com").
		Set("name", "Ana")

	if len(builder.filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(builder.filters))
	}
}

func TestUpdateBuilder_Debug(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, mockConnector(), "User")

	result := builder.Debug()

	if result == nil {
		t.Error("Debug() should return UpdateMutation for chaining")
	}

	if builder.debugLevel == nil {
		t.Error("Debug() should set debugLevel")
	}
}

func TestUpdateBuilder_Chaining(t *testing.T) {
	schema := testSchema()

	ub := NewUpdateBuilder(schema, mockConnector(), "User")
	builder := ub.
		Filter("id", "eq", "uuid-123").
		Set("name", "Ana").
		Set("age", 30).
		Debug()

	if builder == nil {
		t.Error("Chaining should work")
	}

	if len(ub.filters) == 0 {
		t.Error("Filter not added via chaining")
	}

	if len(ub.updates) != 2 {
		t.Error("Updates not added via chaining")
	}
}

// ============================================================
// DELETE BUILDER TESTS
// ============================================================

func TestDeleteBuilder_Filter(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, mockConnector(), "User")

	result := builder.Filter("id", "eq", "uuid-123")

	if result == nil {
		t.Error("Filter() should return DeleteMutation for chaining")
	}

	if len(builder.filters) == 0 {
		t.Error("Filter() should add filter")
	}
}

func TestDeleteBuilder_MultipleFilters(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, mockConnector(), "User")

	builder.Filter("id", "eq", "uuid-123").
		Filter("email", "eq", "ana@mail.com")

	if len(builder.filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(builder.filters))
	}
}

func TestDeleteBuilder_Debug(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, mockConnector(), "User")

	result := builder.Debug()

	if result == nil {
		t.Error("Debug() should return DeleteMutation for chaining")
	}

	if builder.debugLevel == nil {
		t.Error("Debug() should set debugLevel")
	}
}

func TestDeleteBuilder_Chaining(t *testing.T) {
	schema := testSchema()

	ub := NewDeleteBuilder(schema, mockConnector(), "User")
	builder := ub.
		Filter("id", "eq", "uuid-123").
		Debug()

	if builder == nil {
		t.Error("Chaining should work")
	}

	if len(ub.filters) == 0 {
		t.Error("Filter not added via chaining")
	}
}

// ============================================================
// INTERFACE COMPLIANCE TESTS
// ============================================================

func TestInsertBuilder_ImplementsInterface(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, mockConnector(), "User")

	// Verify it implements engine.InsertMutation
	var _ engine.InsertMutation = builder
}

func TestUpdateBuilder_ImplementsInterface(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, mockConnector(), "User")

	// Verify it implements engine.UpdateMutation
	var _ engine.UpdateMutation = builder
}

func TestDeleteBuilder_ImplementsInterface(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, mockConnector(), "User")

	// Verify it implements engine.DeleteMutation
	var _ engine.DeleteMutation = builder
}

// ============================================================
// FACTORY INTEGRATION TESTS
// ============================================================

func TestFactory_Integration(t *testing.T) {
	schema := testSchema()
	connector := mockConnector()
	factory := NewFactory()

	// Test that factory creates working builders
	insert := factory.NewInsert("User", schema, connector)
	if insert == nil {
		t.Error("Factory should create InsertMutation")
	}

	update := factory.NewUpdate("User", schema, connector)
	if update == nil {
		t.Error("Factory should create UpdateMutation")
	}

	delete := factory.NewDelete("User", schema, connector)
	if delete == nil {
		t.Error("Factory should create DeleteMutation")
	}

	// Test chaining through factory-created builders
	insert.Set("email", "test@mail.com").Set("name", "Test")

	// Can't execute without real DB, but we verified chaining works
}

// ============================================================
// SQL GENERATION TESTS (without execution)
// ============================================================

func TestInsertBuilder_GenerateSQL(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, mockConnector(), "User")
	builder.Set("email", "ana@mail.com").Set("name", "Ana")

	sql, values, err := builder.generateSQL()
	if err != nil {
		t.Fatalf("generateSQL should not fail: %v", err)
	}

	// Verify SQL structure
	if sql == "" {
		t.Error("SQL should be generated")
	}

	if len(values) != 2 {
		t.Errorf("Expected 2 values, got %d", len(values))
	}

	// SQL should contain INSERT INTO users
	if !contains(sql, "INSERT INTO") {
		t.Error("SQL should contain INSERT INTO")
	}

	if !contains(sql, "users") {
		t.Error("SQL should reference users table")
	}

	if !contains(sql, "RETURNING") {
		t.Error("SQL should have RETURNING clause")
	}
}

func TestUpdateBuilder_GenerateSQL(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, mockConnector(), "User")
	builder.Filter("id", "eq", "uuid-123").Set("name", "Ana")

	sql, values, err := builder.generateSQL()
	if err != nil {
		t.Fatalf("generateSQL should not fail: %v", err)
	}

	if sql == "" {
		t.Error("SQL should be generated")
	}

	if len(values) != 2 {
		t.Errorf("Expected 2 values (1 set + 1 filter), got %d", len(values))
	}

	if !contains(sql, "UPDATE") {
		t.Error("SQL should contain UPDATE")
	}

	if !contains(sql, "WHERE") {
		t.Error("SQL should have WHERE clause")
	}

	if !contains(sql, "RETURNING") {
		t.Error("SQL should have RETURNING clause")
	}
}

func TestDeleteBuilder_GenerateSQL(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, mockConnector(), "User")
	builder.Filter("id", "eq", "uuid-123")

	sql, values, err := builder.generateSQL()
	if err != nil {
		t.Fatalf("generateSQL should not fail: %v", err)
	}

	if sql == "" {
		t.Error("SQL should be generated")
	}

	if len(values) != 1 {
		t.Errorf("Expected 1 value, got %d", len(values))
	}

	if !contains(sql, "DELETE FROM") {
		t.Error("SQL should contain DELETE FROM")
	}

	if !contains(sql, "WHERE") {
		t.Error("SQL should have WHERE clause")
	}
}

func TestUpdateBuilder_GenerateSQL_UnsupportedOperator(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, mockConnector(), "User")
	builder.Filter("id", "contains", "uuid-123").Set("name", "Ana")

	_, _, err := builder.generateSQL()
	if err == nil {
		t.Fatal("generateSQL should fail for unsupported operator")
	}
}

func TestDeleteBuilder_GenerateSQL_NoFilters(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, mockConnector(), "User")

	_, _, err := builder.generateSQL()
	if err == nil {
		t.Fatal("generateSQL should fail without filters")
	}
}

func TestEntityToTableName(t *testing.T) {
	tests := []struct {
		entity string
		want   string
	}{
		{"User", "users"},
		{"OrderItem", "order_items"},
		{"TodoList", "todo_lists"},
		{"Post", "posts"},
	}

	for _, tt := range tests {
		got := entityToTableName(tt.entity)
		if got != tt.want {
			t.Errorf("entityToTableName(%q) = %q, want %q", tt.entity, got, tt.want)
		}
	}
}

// ============================================================
// HELPERS
// ============================================================

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Note: Tests that require actual DB execution are in tests/integration/mutations_test.go
// These unit tests only verify:
// - Builder API (Set, Filter, Debug)
// - Chaining behavior
// - SQL generation (without execution)
// - Interface compliance

// ============================================================
// IDENTIFIER VALIDATION AND QUOTING
// ============================================================

func newInsert(entity string, fields map[string]interface{}) *InsertBuilder {
	b := NewInsertBuilder(testSchema(), mockConnector(), entity)
	for k, v := range fields {
		b.Set(k, v)
	}
	return b
}

func newUpdate(entity string, filter string, set string) *UpdateBuilder {
	b := NewUpdateBuilder(testSchema(), mockConnector(), entity)
	b.Filter(filter, "eq", 1)
	b.Set(set, 1)
	return b
}

func newDelete(entity string, filter string) *DeleteBuilder {
	b := NewDeleteBuilder(testSchema(), mockConnector(), entity)
	b.Filter(filter, "eq", 1)
	return b
}

var unknownFieldNames = []string{"missing", "", "id x", `id"`, "id'", "id;", "id--", "id/**/", "ID", "caf\u00e9"}

func TestDeleteBuilder_ValidateRejectsUnknownFilterField(t *testing.T) {
	for _, name := range unknownFieldNames {
		builder := NewDeleteBuilder(testSchema(), mockConnector(), "User")
		builder.Filter("id", "eq", "uuid-123").Filter(name, "eq", "x")

		validator := engine.NewValidator(builder.schema, builder.config)
		err := validator.ValidateDeleteInput(builder.entity, builder.parseFilters(), false)
		if _, ok := err.(*engine.UnknownFieldError); !ok {
			t.Errorf("field %q: expected UnknownFieldError, got %T (%v)", name, err, err)
		}
	}
}

func TestBuilders_ExecuteRejectUnknownFieldsBeforeConnecting(t *testing.T) {
	// The connector is nil: any attempt to reach the database would panic.
	ctx := context.Background()
	for _, name := range unknownFieldNames {
		if _, err := NewDeleteBuilder(testSchema(), mockConnector(), "User").
			Filter(name, "eq", "x").Execute(ctx); err == nil {
			t.Errorf("delete accepted %q", name)
		}
		if _, err := NewUpdateBuilder(testSchema(), mockConnector(), "User").
			Filter(name, "eq", "x").Set("name", "n").Execute(ctx); err == nil {
			t.Errorf("update filter accepted %q", name)
		}
		if _, err := NewUpdateBuilder(testSchema(), mockConnector(), "User").
			Filter("id", "eq", "uuid-123").Set(name, "n").Execute(ctx); err == nil {
			t.Errorf("update set accepted %q", name)
		}
		if _, err := NewInsertBuilder(testSchema(), mockConnector(), "User").
			Set(name, "n").Execute(ctx); err == nil {
			t.Errorf("insert accepted %q", name)
		}
	}
}

func TestBuilders_GenerateQuotedIdentifiers(t *testing.T) {
	insert, _, err := newInsert("User", map[string]interface{}{"email": "a", "name": "b"}).generateSQL()
	if err != nil {
		t.Fatal(err)
	}
	if !contains(insert, `INSERT INTO "users" ("email", "name") VALUES ($1, $2)`) {
		t.Errorf("unexpected insert SQL: %s", insert)
	}

	update, _, err := newUpdate("User", "id", "name").generateSQL()
	if err != nil {
		t.Fatal(err)
	}
	if !contains(update, `UPDATE "users" SET "name" = $1 WHERE "id" = $2`) {
		t.Errorf("unexpected update SQL: %s", update)
	}

	del, _, err := newDelete("User", "id").generateSQL()
	if err != nil {
		t.Fatal(err)
	}
	if !contains(del, `DELETE FROM "users" WHERE "id" = $1`) {
		t.Errorf("unexpected delete SQL: %s", del)
	}
}

func TestBuilders_GenerateSQLRejectsInvalidNames(t *testing.T) {
	for _, name := range unknownFieldNames {
		if name == "missing" || name == "ID" {
			continue // valid identifier shapes; rejected by schema validation instead
		}
		if _, _, err := newInsert("User", map[string]interface{}{name: 1}).generateSQL(); err == nil {
			t.Errorf("insert SQL generated for %q", name)
		}
		if _, _, err := newUpdate("User", "id", name).generateSQL(); err == nil {
			t.Errorf("update SET SQL generated for %q", name)
		}
		if _, _, err := newUpdate("User", name, "name").generateSQL(); err == nil {
			t.Errorf("update WHERE SQL generated for %q", name)
		}
		if _, _, err := newDelete("User", name).generateSQL(); err == nil {
			t.Errorf("delete SQL generated for %q", name)
		}
	}
}

func TestBuilders_GenerateSQLRejectsInvalidTableName(t *testing.T) {
	for _, entity := range []string{`User"; --`, "User x", "1User"} {
		if _, _, err := newInsert(entity, map[string]interface{}{"a": 1}).generateSQL(); err == nil {
			t.Errorf("insert SQL generated for entity %q", entity)
		}
		if _, _, err := newUpdate(entity, "id", "a").generateSQL(); err == nil {
			t.Errorf("update SQL generated for entity %q", entity)
		}
		if _, _, err := newDelete(entity, "id").generateSQL(); err == nil {
			t.Errorf("delete SQL generated for entity %q", entity)
		}
	}
}

func TestDeleteBuilder_GenerateSQLIsDeterministic(t *testing.T) {
	builder := NewDeleteBuilder(testSchema(), mockConnector(), "User")
	builder.Filter("id", "eq", "a").Filter("email", "eq", "b").Filter("name", "like", "c").Filter("age", "gte", 1)

	wantSQL, wantValues, err := builder.generateSQL()
	if err != nil {
		t.Fatalf("generateSQL() error = %v", err)
	}

	for i := 0; i < 50; i++ {
		gotSQL, gotValues, err := builder.generateSQL()
		if err != nil {
			t.Fatalf("generateSQL() error = %v", err)
		}
		if gotSQL != wantSQL || fmt.Sprint(gotValues) != fmt.Sprint(wantValues) {
			t.Fatalf("generateSQL is not deterministic:\n%s %v\n%s %v", wantSQL, wantValues, gotSQL, gotValues)
		}
	}
}

func TestUpdateAndDeleteBuilders_NullAndLikeFilters(t *testing.T) {
	update := NewUpdateBuilder(testSchema(), mockConnector(), "User")
	update.Set("name", "x").Filter("age", "eq", nil).Filter("email", "like", "ana")

	sql, values, err := update.generateSQL()
	if err != nil {
		t.Fatalf("update generateSQL() error = %v", err)
	}
	if !strings.Contains(sql, `WHERE "age" IS NULL AND "email" LIKE $2 ESCAPE '\'`) {
		t.Fatalf("unexpected UPDATE SQL: %s", sql)
	}
	if fmt.Sprint(values) != "[x %ana%]" {
		t.Fatalf("values = %v", values)
	}

	del := NewDeleteBuilder(testSchema(), mockConnector(), "User")
	del.Filter("age", "neq", nil)

	sql, values, err = del.generateSQL()
	if err != nil {
		t.Fatalf("delete generateSQL() error = %v", err)
	}
	if sql != `DELETE FROM "users" WHERE "age" IS NOT NULL` || len(values) != 0 {
		t.Fatalf("unexpected DELETE SQL: %s %v", sql, values)
	}
}
