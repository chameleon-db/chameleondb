package mutation

import (
	"context"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// ============================================================
// TEST HELPERS
// ============================================================

// Helper: create test schema
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

// ============================================================
// INSERT BUILDER TESTS
// ============================================================

func TestInsertBuilder_Set(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")

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
	builder := NewInsertBuilder(schema, "User")

	// Set same field twice (last one should win)
	builder.Set("name", "Ana").Set("name", "Ana María")

	if builder.values["name"] != "Ana María" {
		t.Errorf("Expected last value to win, got '%v'", builder.values["name"])
	}
}

func TestInsertBuilder_Debug(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")

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

func TestInsertBuilder_Execute_Success(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")
	builder.Set("email", "ana@mail.com").Set("name", "Ana")

	result, err := builder.Execute(context.Background())

	if err != nil {
		t.Fatalf("Execute() should succeed with valid input: %v", err)
	}

	if result == nil {
		t.Error("Execute() should return non-nil result")
	}

	// Note: Affected count is mock data since we don't execute real SQL yet
	if result.Affected != 1 {
		t.Errorf("Expected Affected=1, got %d", result.Affected)
	}
}

func TestInsertBuilder_Execute_ValidationError_UnknownField(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")
	builder.Set("unknown_field", "value")

	_, err := builder.Execute(context.Background())

	if err == nil {
		t.Error("Execute() should fail with unknown field")
	}
}

func TestInsertBuilder_Execute_ValidationError_UnknownEntity(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "NonExistentEntity")
	builder.Set("field", "value")

	_, err := builder.Execute(context.Background())

	if err == nil {
		t.Error("Execute() should fail with non-existent entity")
	}
}

func TestInsertBuilder_Execute_ValidationError_MissingRequiredField(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")
	// Missing required fields: email, name

	_, err := builder.Execute(context.Background())

	if err == nil {
		t.Error("Execute() should fail when required fields are missing")
	}
}

func TestInsertBuilder_Chaining(t *testing.T) {
	schema := testSchema()

	// Test full chain: Set -> Set -> Debug -> Execute
	result, err := NewInsertBuilder(schema, "User").
		Set("email", "ana@mail.com").
		Set("name", "Ana").
		Set("age", 28).
		Debug().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Chaining should work: %v", err)
	}

	if result == nil {
		t.Error("Chained execution should return result")
	}
}

// ============================================================
// UPDATE BUILDER TESTS
// ============================================================

func TestUpdateBuilder_Set(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")

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
	builder := NewUpdateBuilder(schema, "User")

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
	builder := NewUpdateBuilder(schema, "User")

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
	builder := NewUpdateBuilder(schema, "User")

	builder.Filter("id", "eq", "uuid-123").
		Filter("email", "eq", "ana@mail.com").
		Set("name", "Ana")

	if len(builder.filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(builder.filters))
	}
}

func TestUpdateBuilder_Debug(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")

	result := builder.Debug()

	if result == nil {
		t.Error("Debug() should return UpdateMutation for chaining")
	}

	if builder.debugLevel == nil {
		t.Error("Debug() should set debugLevel")
	}
}

func TestUpdateBuilder_Execute_Success(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")
	builder.Filter("id", "eq", "uuid-123").Set("name", "Ana")

	result, err := builder.Execute(context.Background())

	if err != nil {
		t.Fatalf("Execute() should succeed: %v", err)
	}

	if result == nil {
		t.Error("Execute() should return result")
	}
}

func TestUpdateBuilder_Execute_ValidationError_NoFilter(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")
	builder.Set("name", "Ana")

	_, err := builder.Execute(context.Background())

	if err == nil {
		t.Error("Execute() should fail without filter (safety guard)")
	}
}

func TestUpdateBuilder_Execute_ValidationError_UpdatePrimaryKey(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")
	builder.Filter("email", "eq", "old@mail.com").Set("id", "new-uuid")

	_, err := builder.Execute(context.Background())

	if err == nil {
		t.Error("Execute() should fail when trying to update primary key")
	}
}

func TestUpdateBuilder_Chaining(t *testing.T) {
	schema := testSchema()

	result, err := NewUpdateBuilder(schema, "User").
		Filter("id", "eq", "uuid-123").
		Set("name", "Ana").
		Set("age", 30).
		Debug().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Chaining should work: %v", err)
	}

	if result == nil {
		t.Error("Chained execution should return result")
	}
}

// ============================================================
// DELETE BUILDER TESTS
// ============================================================

func TestDeleteBuilder_Filter(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")

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
	builder := NewDeleteBuilder(schema, "User")

	builder.Filter("id", "eq", "uuid-123").
		Filter("email", "eq", "ana@mail.com")

	if len(builder.filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(builder.filters))
	}
}

func TestDeleteBuilder_Debug(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")

	result := builder.Debug()

	if result == nil {
		t.Error("Debug() should return DeleteMutation for chaining")
	}

	if builder.debugLevel == nil {
		t.Error("Debug() should set debugLevel")
	}
}

func TestDeleteBuilder_Execute_Success(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")
	builder.Filter("id", "eq", "uuid-123")

	result, err := builder.Execute(context.Background())

	if err != nil {
		t.Fatalf("Execute() should succeed: %v", err)
	}

	if result == nil {
		t.Error("Execute() should return result")
	}
}

func TestDeleteBuilder_Execute_ValidationError_NoFilter(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")

	_, err := builder.Execute(context.Background())

	if err == nil {
		t.Error("Execute() should fail without filter (safety guard)")
	}
}

func TestDeleteBuilder_Chaining(t *testing.T) {
	schema := testSchema()

	result, err := NewDeleteBuilder(schema, "User").
		Filter("id", "eq", "uuid-123").
		Debug().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Chaining should work: %v", err)
	}

	if result == nil {
		t.Error("Chained execution should return result")
	}
}

// ============================================================
// INTERFACE COMPLIANCE TESTS
// ============================================================

func TestInsertBuilder_ImplementsInterface(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")

	// Verify it implements engine.InsertMutation
	var _ engine.InsertMutation = builder
}

func TestUpdateBuilder_ImplementsInterface(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")

	// Verify it implements engine.UpdateMutation
	var _ engine.UpdateMutation = builder
}

func TestDeleteBuilder_ImplementsInterface(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")

	// Verify it implements engine.DeleteMutation
	var _ engine.DeleteMutation = builder
}

// ============================================================
// CROSS-BUILDER INTEGRATION TESTS
// ============================================================

func TestFactory_Integration(t *testing.T) {
	schema := testSchema()
	factory := NewFactory(schema)

	// Test that factory creates working builders
	insert := factory.NewInsert("User")
	if insert == nil {
		t.Error("Factory should create InsertMutation")
	}

	update := factory.NewUpdate("User")
	if update == nil {
		t.Error("Factory should create UpdateMutation")
	}

	delete := factory.NewDelete("User")
	if delete == nil {
		t.Error("Factory should create DeleteMutation")
	}

	// Test chaining through factory-created builders
	_, err := insert.Set("email", "test@mail.com").
		Set("name", "Test").
		Execute(context.Background())

	if err != nil {
		t.Errorf("Factory-created builder should work: %v", err)
	}
}

// ============================================================
// VALIDATION EDGE CASES
// ============================================================

func TestInsertBuilder_Execute_InvalidEmail(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")
	builder.Set("email", "not-an-email").Set("name", "Ana")

	_, err := builder.Execute(context.Background())

	if err == nil {
		t.Error("Execute() should fail with invalid email format")
	}
}

func TestInsertBuilder_Execute_NullableField(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")
	builder.Set("email", "ana@mail.com").
		Set("name", "Ana").
		Set("age", nil) // age is nullable

	result, err := builder.Execute(context.Background())

	if err != nil {
		t.Errorf("Execute() should succeed with null nullable field: %v", err)
	}

	if result == nil {
		t.Error("Should return result")
	}
}

func TestUpdateBuilder_Execute_EmptyUpdate(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")
	builder.Filter("id", "eq", "uuid-123")
	// No Set() called

	_, err := builder.Execute(context.Background())

	if err == nil {
		t.Error("Execute() should fail with empty update")
	}
}

// ============================================================
// DEBUG MODE TESTS
// ============================================================

func TestInsertBuilder_Debug_DoesNotBreakExecution(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")

	result, err := builder.
		Set("email", "ana@mail.com").
		Set("name", "Ana").
		Debug(). // Enable debug
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Debug mode should not break execution: %v", err)
	}

	if result == nil {
		t.Error("Should return result even with debug enabled")
	}
}

func TestUpdateBuilder_Debug_DoesNotBreakExecution(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")

	result, err := builder.
		Filter("id", "eq", "uuid-123").
		Set("name", "Ana").
		Debug().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Debug mode should not break execution: %v", err)
	}

	if result == nil {
		t.Error("Should return result even with debug enabled")
	}
}

func TestDeleteBuilder_Debug_DoesNotBreakExecution(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")

	result, err := builder.
		Filter("id", "eq", "uuid-123").
		Debug().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Debug mode should not break execution: %v", err)
	}

	if result == nil {
		t.Error("Should return result even with debug enabled")
	}
}
