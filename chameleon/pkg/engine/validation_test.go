package engine

import (
	"testing"

	"github.com/google/uuid"
)

func getTestSchema() *Schema {
	return &Schema{
		Entities: []*Entity{
			{
				Name: "User",
				Fields: map[string]*Field{
					"id": {
						Name:       "id",
						Type:       FieldType{Kind: "UUID"},
						PrimaryKey: true,
					},
					"email": {
						Name:     "email",
						Type:     FieldType{Kind: "String"},
						Nullable: false,
					},
					"name": {
						Name:     "name",
						Type:     FieldType{Kind: "String"},
						Nullable: false,
					},
					"age": {
						Name:     "age",
						Type:     FieldType{Kind: "Int"},
						Nullable: true,
					},
				},
			},
			{
				Name: "Post",
				Fields: map[string]*Field{
					"id": {
						Name:       "id",
						Type:       FieldType{Kind: "UUID"},
						PrimaryKey: true,
					},
					"title": {
						Name:     "title",
						Type:     FieldType{Kind: "String"},
						Nullable: false,
					},
				},
			},
		},
	}
}

func TestNewValidator(t *testing.T) {
	schema := getTestSchema()
	config := DefaultValidatorConfig()
	validator := NewValidator(schema, config)

	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}
	if validator.schema != schema {
		t.Error("Expected validator to have the schema")
	}
}

func TestDefaultValidatorConfig(t *testing.T) {
	config := DefaultValidatorConfig()
	if !config.StrictTypes {
		t.Error("Expected StrictTypes to be true")
	}
	if !config.ValidateFK {
		t.Error("Expected ValidateFK to be true")
	}
}

func TestValidateInsertInput_Success(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	input := map[string]interface{}{
		"id":    uuid.New().String(),
		"email": "test@mail.com",
		"name":  "John Doe",
		"age":   25,
	}

	err := validator.ValidateInsertInput("User", input)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestValidateInsertInput_UnknownEntity(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	input := map[string]interface{}{
		"name": "Product",
	}

	err := validator.ValidateInsertInput("Product", input)
	if err == nil {
		t.Fatal("Expected error for unknown entity")
	}

	unknownErr, ok := err.(*UnknownEntityError)
	if !ok {
		t.Fatalf("Expected UnknownEntityError, got %T", err)
	}
	if unknownErr.Entity != "Product" {
		t.Errorf("Expected entity 'Product', got %s", unknownErr.Entity)
	}
}

func TestValidateInsertInput_UnknownField(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	input := map[string]interface{}{
		"id":            uuid.New().String(),
		"email":         "test@mail.com",
		"name":          "John",
		"unknown_field": "value",
	}

	err := validator.ValidateInsertInput("User", input)
	if err == nil {
		t.Fatal("Expected error for unknown field")
	}

	unknownErr, ok := err.(*UnknownFieldError)
	if !ok {
		t.Fatalf("Expected UnknownFieldError, got %T", err)
	}
	if unknownErr.Field != "unknown_field" {
		t.Errorf("Expected field 'unknown_field', got %s", unknownErr.Field)
	}
}

func TestValidateInsertInput_MissingRequiredField(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	input := map[string]interface{}{
		"id":    uuid.New().String(),
		"email": "test@mail.com",
		// missing "name" which is required
	}

	err := validator.ValidateInsertInput("User", input)
	if err == nil {
		t.Fatal("Expected error for missing required field")
	}

	notNullErr, ok := err.(*NotNullError)
	if !ok {
		t.Fatalf("Expected NotNullError, got %T", err)
	}
	if notNullErr.Field != "name" {
		t.Errorf("Expected field 'name', got %s", notNullErr.Field)
	}
}

func TestValidateInsertInput_InvalidUUID(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	input := map[string]interface{}{
		"id":    "not-a-valid-uuid",
		"email": "test@mail.com",
		"name":  "John",
	}

	err := validator.ValidateInsertInput("User", input)
	if err == nil {
		t.Fatal("Expected error for invalid UUID")
	}

	formatErr, ok := err.(*FieldFormatError)
	if !ok {
		t.Fatalf("Expected FieldFormatError, got %T", err)
	}
	if formatErr.Field != "id" {
		t.Errorf("Expected field 'id', got %s", formatErr.Field)
	}
	if formatErr.Format != "UUID" {
		t.Errorf("Expected format 'UUID', got %s", formatErr.Format)
	}
}

func TestValidateInsertInput_InvalidEmail(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	input := map[string]interface{}{
		"id":    uuid.New().String(),
		"email": "not-an-email",
		"name":  "John",
	}

	err := validator.ValidateInsertInput("User", input)
	if err == nil {
		t.Fatal("Expected error for invalid email")
	}

	formatErr, ok := err.(*FieldFormatError)
	if !ok {
		t.Fatalf("Expected FieldFormatError, got %T", err)
	}
	if formatErr.Field != "email" {
		t.Errorf("Expected field 'email', got %s", formatErr.Field)
	}
	if formatErr.Format != "email" {
		t.Errorf("Expected format 'email', got %s", formatErr.Format)
	}
}

func TestValidateInsertInput_NullableField(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	input := map[string]interface{}{
		"id":    uuid.New().String(),
		"email": "test@mail.com",
		"name":  "John",
		"age":   nil, // age is nullable
	}

	err := validator.ValidateInsertInput("User", input)
	if err != nil {
		t.Errorf("Expected no error for nullable field, got: %v", err)
	}
}

func TestValidateInsertInput_TypeMismatch(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	input := map[string]interface{}{
		"id":    uuid.New().String(),
		"email": 12345, // should be string
		"name":  "John",
	}

	err := validator.ValidateInsertInput("User", input)
	if err == nil {
		t.Fatal("Expected error for type mismatch")
	}

	typeErr, ok := err.(*TypeMismatchError)
	if !ok {
		t.Fatalf("Expected TypeMismatchError, got %T", err)
	}
	if typeErr.Field != "email" {
		t.Errorf("Expected field 'email', got %s", typeErr.Field)
	}
}

func TestValidateUpdateInput_Success(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	filters := map[string]interface{}{
		"id": uuid.New().String(),
	}
	updates := map[string]interface{}{
		"name": "Updated Name",
		"age":  30,
	}

	err := validator.ValidateUpdateInput("User", filters, updates)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestValidateUpdateInput_NoFilters(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	filters := map[string]interface{}{}
	updates := map[string]interface{}{
		"name": "Updated",
	}

	err := validator.ValidateUpdateInput("User", filters, updates)
	if err == nil {
		t.Fatal("Expected error for missing WHERE clause")
	}

	safetyErr, ok := err.(*SafetyError)
	if !ok {
		t.Fatalf("Expected SafetyError, got %T", err)
	}
	if safetyErr.Operation != "update_without_filter" {
		t.Errorf("Expected operation 'update_without_filter', got %s", safetyErr.Operation)
	}
}

func TestValidateUpdateInput_NoUpdates(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	filters := map[string]interface{}{
		"id": uuid.New().String(),
	}
	updates := map[string]interface{}{}

	err := validator.ValidateUpdateInput("User", filters, updates)
	if err == nil {
		t.Fatal("Expected error for no updates")
	}

	safetyErr, ok := err.(*SafetyError)
	if !ok {
		t.Fatalf("Expected SafetyError, got %T", err)
	}
	if safetyErr.Operation != "update_with_no_fields" {
		t.Errorf("Expected operation 'update_with_no_fields', got %s", safetyErr.Operation)
	}
}

func TestValidateUpdateInput_UpdatePrimaryKey(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	filters := map[string]interface{}{
		"email": "test@mail.com",
	}
	updates := map[string]interface{}{
		"id": uuid.New().String(), // trying to update primary key
	}

	err := validator.ValidateUpdateInput("User", filters, updates)
	if err == nil {
		t.Fatal("Expected error for updating primary key")
	}

	constraintErr, ok := err.(*ConstraintError)
	if !ok {
		t.Fatalf("Expected ConstraintError, got %T", err)
	}
	if constraintErr.Type != "primary_key" {
		t.Errorf("Expected constraint type 'primary_key', got %s", constraintErr.Type)
	}
}

func TestValidateDeleteInput_Success(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	filters := map[string]interface{}{
		"id": uuid.New().String(),
	}

	err := validator.ValidateDeleteInput("User", filters, false)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestValidateDeleteInput_NoFiltersNoForce(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	filters := map[string]interface{}{}

	err := validator.ValidateDeleteInput("User", filters, false)
	if err == nil {
		t.Fatal("Expected error for DELETE without WHERE")
	}

	safetyErr, ok := err.(*SafetyError)
	if !ok {
		t.Fatalf("Expected SafetyError, got %T", err)
	}
	if safetyErr.Operation != "delete_without_filter" {
		t.Errorf("Expected operation 'delete_without_filter', got %s", safetyErr.Operation)
	}
}

func TestValidateDeleteInput_ForceDeleteAll(t *testing.T) {
	schema := getTestSchema()
	validator := NewValidator(schema, DefaultValidatorConfig())

	filters := map[string]interface{}{}

	err := validator.ValidateDeleteInput("User", filters, true)
	if err != nil {
		t.Errorf("Expected no error with forceDeleteAll, got: %v", err)
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid UUID", uuid.New().String(), true},
		{"valid UUID with dashes", "550e8400-e29b-41d4-a716-446655440000", true},
		{"invalid UUID", "not-a-uuid", false},
		{"empty string", "", false},
		{"random string", "abcd1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidUUID(tt.input)
			if got != tt.want {
				t.Errorf("isValidUUID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid email", "test@example.com", true},
		{"valid email with subdomain", "user@mail.example.com", true},
		{"invalid - no @", "noatsign.com", false},
		{"invalid - no domain", "user@", false},
		{"invalid - no TLD", "user@example", false},
		{"invalid - spaces", "user @example.com", false},
		{"invalid - empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidEmail(tt.input)
			if got != tt.want {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
