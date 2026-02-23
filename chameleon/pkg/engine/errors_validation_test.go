package engine

import (
	"strings"
	"testing"
)

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:    "email",
		Type:     "type_mismatch",
		Value:    123,
		Expected: "string",
		Message:  "Email must be a string",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "email") {
		t.Errorf("Error message should contain field name")
	}
	if !strings.Contains(errMsg, "type_mismatch") {
		t.Errorf("Error message should contain type")
	}
	if !strings.Contains(errMsg, "123") {
		t.Errorf("Error message should contain value")
	}

	if err.Code() != "VALIDATION_ERROR" {
		t.Errorf("Expected code VALIDATION_ERROR, got %s", err.Code())
	}

	// Verify it implements MutationError
	var _ MutationError = err
}

func TestTypeMismatchError(t *testing.T) {
	err := &TypeMismatchError{
		Field:        "age",
		ExpectedType: "int",
		ReceivedType: "string",
		Value:        "twenty-five",
		Suggestion:   "Provide a numeric value",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "age") {
		t.Errorf("Error message should contain field name")
	}
	if !strings.Contains(errMsg, "int") {
		t.Errorf("Error message should contain expected type")
	}
	if !strings.Contains(errMsg, "twenty-five") {
		t.Errorf("Error message should contain value")
	}

	if err.Code() != "TYPE_MISMATCH" {
		t.Errorf("Expected code TYPE_MISMATCH, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestLengthExceededError(t *testing.T) {
	longValue := strings.Repeat("a", 300)
	err := &LengthExceededError{
		Field:  "username",
		MaxLen: 255,
		Actual: 300,
		Value:  longValue,
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "username") {
		t.Errorf("Error message should contain field name")
	}
	if !strings.Contains(errMsg, "255") {
		t.Errorf("Error message should contain max length")
	}
	if !strings.Contains(errMsg, "300") {
		t.Errorf("Error message should contain actual length")
	}

	if err.Code() != "LENGTH_EXCEEDED" {
		t.Errorf("Expected code LENGTH_EXCEEDED, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestFieldFormatError(t *testing.T) {
	err := &FieldFormatError{
		Field:      "email",
		Format:     "email",
		Value:      "not-an-email",
		Suggestion: "Use valid email format: user@domain.com",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "email") {
		t.Errorf("Error message should contain field name")
	}
	if !strings.Contains(errMsg, "not-an-email") {
		t.Errorf("Error message should contain value")
	}
	if !strings.Contains(errMsg, "user@domain.com") {
		t.Errorf("Error message should contain suggestion")
	}

	if err.Code() != "FORMAT_ERROR" {
		t.Errorf("Expected code FORMAT_ERROR, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestConstraintError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConstraintError
		wantCode string
	}{
		{
			name: "unique constraint",
			err: &ConstraintError{
				Type:       "unique",
				Field:      "email",
				Value:      "test@mail.com",
				Suggestion: "Use different email",
			},
			wantCode: "unique_CONSTRAINT",
		},
		{
			name: "not null constraint",
			err: &ConstraintError{
				Type:       "not_null",
				Field:      "name",
				Value:      nil,
				Suggestion: "Provide a value",
			},
			wantCode: "not_null_CONSTRAINT",
		},
		{
			name: "foreign key constraint",
			err: &ConstraintError{
				Type:       "foreign_key",
				Field:      "user_id",
				Value:      "unknown-id",
				Suggestion: "Reference existing user",
			},
			wantCode: "foreign_key_CONSTRAINT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			if !strings.Contains(errMsg, tt.err.Type) {
				t.Errorf("Error message should contain constraint type")
			}
			if !strings.Contains(errMsg, tt.err.Field) {
				t.Errorf("Error message should contain field name")
			}

			if tt.err.Code() != tt.wantCode {
				t.Errorf("Expected code %s, got %s", tt.wantCode, tt.err.Code())
			}

			var _ MutationError = tt.err
		})
	}
}

func TestUniqueConstraintError(t *testing.T) {
	err := &UniqueConstraintError{
		Field: "email",
		Value: "duplicate@mail.com",
		ConflictingRow: map[string]interface{}{
			"id":    "existing-id-123",
			"email": "duplicate@mail.com",
			"name":  "John Doe",
		},
		Table:      "users",
		Suggestion: "Use a different email address",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "email") {
		t.Errorf("Error message should contain field name")
	}
	if !strings.Contains(errMsg, "duplicate@mail.com") {
		t.Errorf("Error message should contain value")
	}
	if !strings.Contains(errMsg, "users") {
		t.Errorf("Error message should contain table name")
	}
	if !strings.Contains(errMsg, "existing-id-123") {
		t.Errorf("Error message should contain conflicting row ID")
	}

	if err.Code() != "UNIQUE_CONSTRAINT_VIOLATION" {
		t.Errorf("Expected code UNIQUE_CONSTRAINT_VIOLATION, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestNotNullError(t *testing.T) {
	err := &NotNullError{
		Field:      "name",
		Suggestion: "This field is required",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "name") {
		t.Errorf("Error message should contain field name")
	}
	if !strings.Contains(errMsg, "required") {
		t.Errorf("Error message should contain suggestion")
	}

	if err.Code() != "NOT_NULL_VIOLATION" {
		t.Errorf("Expected code NOT_NULL_VIOLATION, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestForeignKeyError(t *testing.T) {
	err := &ForeignKeyError{
		Field:           "user_id",
		Value:           "nonexistent-user-123",
		ReferencedTable: "users",
		Suggestion:      "Ensure the user exists before creating the post",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "user_id") {
		t.Errorf("Error message should contain field name")
	}
	if !strings.Contains(errMsg, "users") {
		t.Errorf("Error message should contain referenced table")
	}
	if !strings.Contains(errMsg, "nonexistent-user-123") {
		t.Errorf("Error message should contain value")
	}

	if err.Code() != "FOREIGN_KEY_VIOLATION" {
		t.Errorf("Expected code FOREIGN_KEY_VIOLATION, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestForeignKeyConstraintError(t *testing.T) {
	err := &ForeignKeyConstraintError{
		Entity:         "User",
		ID:             "123",
		DependentTable: "Post",
		DependentCount: 5,
		Suggestion:     "Delete dependent posts first",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "User") {
		t.Errorf("Error message should contain entity name")
	}
	if !strings.Contains(errMsg, "Post") {
		t.Errorf("Error message should contain dependent table")
	}

	if err.Code() != "FOREIGN_KEY_CONSTRAINT_VIOLATION" {
		t.Errorf("Expected code FOREIGN_KEY_CONSTRAINT_VIOLATION, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestSafetyError(t *testing.T) {
	err := &SafetyError{
		Operation:  "delete_without_where",
		Message:    "DELETE without WHERE clause is not allowed",
		Suggestion: "Add Filter() or use ForceDeleteAll()",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "delete_without_where") {
		t.Errorf("Error message should contain operation")
	}
	if !strings.Contains(errMsg, "DELETE") {
		t.Errorf("Error message should contain message")
	}
	if !strings.Contains(errMsg, "Filter") {
		t.Errorf("Error message should contain suggestion")
	}

	if err.Code() != "SAFETY_VIOLATION" {
		t.Errorf("Expected code SAFETY_VIOLATION, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestUnknownEntityError(t *testing.T) {
	err := &UnknownEntityError{
		Entity:    "Product",
		Available: []string{"User", "Post", "Comment"},
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Product") {
		t.Errorf("Error message should contain entity name")
	}
	if !strings.Contains(errMsg, "User") || !strings.Contains(errMsg, "Post") {
		t.Errorf("Error message should contain available entities")
	}

	if err.Code() != "UNKNOWN_ENTITY" {
		t.Errorf("Expected code UNKNOWN_ENTITY, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestUnknownFieldError(t *testing.T) {
	err := &UnknownFieldError{
		Entity:    "User",
		Field:     "phone_number",
		Available: []string{"id", "name", "email", "age"},
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "phone_number") {
		t.Errorf("Error message should contain field name")
	}
	if !strings.Contains(errMsg, "User") {
		t.Errorf("Error message should contain entity name")
	}
	if !strings.Contains(errMsg, "email") {
		t.Errorf("Error message should contain available fields")
	}

	if err.Code() != "UNKNOWN_FIELD" {
		t.Errorf("Expected code UNKNOWN_FIELD, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{
		Entity: "User",
		ID:     "123",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "User") {
		t.Errorf("Error message should contain entity name")
	}
	if !strings.Contains(errMsg, "123") {
		t.Errorf("Error message should contain ID")
	}

	if err.Code() != "NOT_FOUND" {
		t.Errorf("Expected code NOT_FOUND, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestConflictError(t *testing.T) {
	err := &ConflictError{
		Entity:          "Post",
		ID:              "456",
		ExpectedVersion: 1,
		ActualVersion:   2,
		Suggestion:      "Retry with the latest version",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Post") {
		t.Errorf("Error message should contain entity name")
	}
	if !strings.Contains(errMsg, "456") {
		t.Errorf("Error message should contain ID")
	}
	if !strings.Contains(errMsg, "Concurrent modification") {
		t.Errorf("Error message should mention concurrent modification")
	}

	if err.Code() != "CONFLICT" {
		t.Errorf("Expected code CONFLICT, got %s", err.Code())
	}

	var _ MutationError = err
}

func TestAuthorizationError(t *testing.T) {
	err := &AuthorizationError{
		Operation: "DELETE",
		Entity:    "User",
		Message:   "Insufficient permissions to delete users",
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "DELETE") {
		t.Errorf("Error message should contain operation")
	}
	if !strings.Contains(errMsg, "User") {
		t.Errorf("Error message should contain entity name")
	}
	if !strings.Contains(errMsg, "Insufficient permissions") {
		t.Errorf("Error message should contain message")
	}

	if err.Code() != "AUTHORIZATION_DENIED" {
		t.Errorf("Expected code AUTHORIZATION_DENIED, got %s", err.Code())
	}

	var _ MutationError = err
}
