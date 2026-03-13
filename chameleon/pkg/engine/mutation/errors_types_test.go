package mutation

import (
	"errors"
	"strings"
	"testing"
)

func TestMutationErrorTypesImplementContract(t *testing.T) {
	testCases := []struct {
		name         string
		err          MutationError
		expectedCode string
		containsText string
	}{
		{
			name:         "ValidationError",
			err:          &ValidationError{Field: "email", Type: "invalid_format", Expected: "string", Value: 123, Message: "bad value"},
			expectedCode: "VALIDATION_ERROR",
			containsText: "ValidationError",
		},
		{
			name:         "TypeMismatchError",
			err:          &TypeMismatchError{Field: "id", ExpectedType: "uuid", ReceivedType: "string", Value: "x", Suggestion: "use uuid"},
			expectedCode: "TYPE_MISMATCH",
			containsText: "TypeMismatchError",
		},
		{
			name:         "LengthExceededError",
			err:          &LengthExceededError{Field: "name", MaxLen: 10, Actual: 20, Value: "too long"},
			expectedCode: "LENGTH_EXCEEDED",
			containsText: "LengthExceededError",
		},
		{
			name:         "FormatError",
			err:          &FormatError{Field: "email", Format: "email", Value: "nope", Suggestion: "fix"},
			expectedCode: "FORMAT_ERROR",
			containsText: "FormatError",
		},
		{
			name:         "ConstraintError",
			err:          &ConstraintError{Type: "check", Field: "age", Value: -1, Suggestion: "age >= 0"},
			expectedCode: "check_CONSTRAINT",
			containsText: "ConstraintError",
		},
		{
			name:         "UniqueConstraintError",
			err:          &UniqueConstraintError{Field: "email", Value: "a@b.com", Table: "users", ConflictingRow: map[string]interface{}{"id": 1}, Suggestion: "change email"},
			expectedCode: "UNIQUE_CONSTRAINT_VIOLATION",
			containsText: "UniqueConstraintError",
		},
		{
			name:         "NotNullError",
			err:          &NotNullError{Field: "name", Suggestion: "set name"},
			expectedCode: "NOT_NULL_VIOLATION",
			containsText: "NotNullError",
		},
		{
			name:         "ForeignKeyError",
			err:          &ForeignKeyError{Field: "authorId", Value: "missing", ReferencedTable: "users", ReferencedField: "id", ReferencedEntity: "User", Suggestion: "create user"},
			expectedCode: "FOREIGN_KEY_VIOLATION",
			containsText: "ForeignKeyError",
		},
		{
			name:         "ForeignKeyConstraintError",
			err:          &ForeignKeyConstraintError{Entity: "User", ID: "u1", DependentTable: "Post", DependentCount: 3, Suggestion: "delete posts"},
			expectedCode: "FOREIGN_KEY_CONSTRAINT_VIOLATION",
			containsText: "ForeignKeyConstraintError",
		},
		{
			name:         "UnknownFieldError",
			err:          &UnknownFieldError{Entity: "User", Field: "oops", Available: []string{"id", "email"}},
			expectedCode: "UNKNOWN_FIELD",
			containsText: "UnknownFieldError",
		},
		{
			name:         "UnknownEntityError",
			err:          &UnknownEntityError{Entity: "Nope", Available: []string{"User"}},
			expectedCode: "UNKNOWN_ENTITY",
			containsText: "UnknownEntityError",
		},
		{
			name:         "NotFoundError",
			err:          &NotFoundError{Entity: "User", ID: 1},
			expectedCode: "NOT_FOUND",
			containsText: "NotFoundError",
		},
		{
			name:         "ConflictError",
			err:          &ConflictError{Entity: "User", ID: 1, ExpectedVersion: 1, ActualVersion: 2, Suggestion: "retry"},
			expectedCode: "CONFLICT",
			containsText: "ConflictError",
		},
		{
			name:         "SafetyError",
			err:          &SafetyError{Operation: "delete_all", Rows: 100, Threshold: 10, Message: "too many", Suggestion: "add filter"},
			expectedCode: "SAFETY_VIOLATION",
			containsText: "SafetyError",
		},
		{
			name:         "AuthorizationError",
			err:          &AuthorizationError{Operation: "delete", Entity: "User", Message: "denied"},
			expectedCode: "AUTHORIZATION_DENIED",
			containsText: "AuthorizationError",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			message := testCase.err.Error()
			if !strings.Contains(message, testCase.containsText) {
				t.Fatalf("error message does not contain marker %q: %q", testCase.containsText, message)
			}
			if code := testCase.err.Code(); code != testCase.expectedCode {
				t.Fatalf("expected code %q, got %q", testCase.expectedCode, code)
			}
			if !IsMutationError(testCase.err) {
				t.Fatal("expected IsMutationError=true")
			}
			if got := ErrorCode(testCase.err); got != testCase.expectedCode {
				t.Fatalf("expected ErrorCode %q, got %q", testCase.expectedCode, got)
			}
		})
	}
}

func TestMutationErrorHelpers(t *testing.T) {
	plainError := errors.New("plain")
	if IsMutationError(plainError) {
		t.Fatal("expected IsMutationError=false for plain error")
	}
	if got := ErrorCode(plainError); got != "UNKNOWN_ERROR" {
		t.Fatalf("expected UNKNOWN_ERROR, got %q", got)
	}

	safetyError := &SafetyError{Operation: "delete_all", Rows: 12, Threshold: 10, Message: "too many", Suggestion: "filter"}
	if !IsSafetyError(safetyError) {
		t.Fatal("expected IsSafetyError=true")
	}
	if IsSafetyError(&ValidationError{Field: "id"}) {
		t.Fatal("expected IsSafetyError=false for non-safety error")
	}

	constraintErrors := []error{
		&UniqueConstraintError{},
		&NotNullError{},
		&ForeignKeyError{},
		&ForeignKeyConstraintError{},
	}
	for _, constraintError := range constraintErrors {
		if !IsConstraintError(constraintError) {
			t.Fatalf("expected IsConstraintError=true for %T", constraintError)
		}
	}
	if IsConstraintError(&ValidationError{}) {
		t.Fatal("expected IsConstraintError=false for validation error")
	}
}
