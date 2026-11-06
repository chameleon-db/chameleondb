package mutation

import (
	"errors"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestExtractHelpers(t *testing.T) {
	if got := extractFieldFromDetail("Key (email)=(test@mail.com) already exists."); got != "email" {
		t.Fatalf("extractFieldFromDetail() = %q", got)
	}
	if got := extractFieldFromMessage(`column "unknown_field" of relation "users" does not exist`); got != "unknown_field" {
		t.Fatalf("extractFieldFromMessage() = %q", got)
	}
	if got := extractReferencedTable("fk_posts_author_id_users"); got != "users" {
		t.Fatalf("extractReferencedTable() = %q", got)
	}
	if got := extractReferencedTableFromDetail(`Key (author_id)=(x) is not present in table "users".`); got != "users" {
		t.Fatalf("extractReferencedTableFromDetail() = %q", got)
	}

	values := map[string]interface{}{"email": "ana@mail.com"}
	if got := extractValueForField("email", values); got != "ana@mail.com" {
		t.Fatalf("extractValueForField() = %#v", got)
	}
}

func TestMapDatabaseErrorNonPgError(t *testing.T) {
	err := mapDatabaseError(errors.New("boom"), "User", "insert", nil)
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if err.Error() == "boom" {
		t.Fatal("expected contextual wrapping for non-pg error")
	}
}

func TestMapDatabaseErrorUniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:   "23505",
		Detail: "Key (email)=(test@mail.com) already exists.",
	}

	err := mapDatabaseError(pgErr, "User", "insert", map[string]interface{}{"email": "test@mail.com"})
	if err == nil {
		t.Fatal("expected mapped error")
	}

	_, ok := err.(*engine.UniqueConstraintError)
	if !ok {
		t.Fatalf("expected UniqueConstraintError, got %T", err)
	}
}

func TestMapDatabaseErrorForeignKeyViolation(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:   "23503",
		Detail: `Key (author_id)=(uuid-999) is not present in table "users".`,
	}

	err := mapDatabaseError(pgErr, "Post", "insert", map[string]interface{}{"author_id": "uuid-999"})
	if err == nil {
		t.Fatal("expected mapped error")
	}

	_, ok := err.(*engine.ForeignKeyError)
	if !ok {
		t.Fatalf("expected ForeignKeyError, got %T", err)
	}
}

func TestMapDatabaseErrorForeignKeyOnDelete(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:           "23503",
		Message:        `update or delete on table "users" violates foreign key constraint "posts_author_id_fkey" on table "posts"`,
		Detail:         `Key (id)=(0d9e5f1c-7f3a-4a2e-9d3b-1f0e8c6a4b21) is still referenced from table "posts".`,
		ConstraintName: "posts_author_id_fkey",
	}

	err := mapDatabaseError(pgErr, "User", "DELETE", nil)

	fkErr, ok := err.(*engine.ForeignKeyConstraintError)
	if !ok {
		t.Fatalf("expected ForeignKeyConstraintError, got %T", err)
	}
	if fkErr.Entity != "User" {
		t.Errorf("Entity = %q, want User", fkErr.Entity)
	}
	if fkErr.DependentTable != "posts" {
		t.Errorf("DependentTable = %q, want posts", fkErr.DependentTable)
	}
	if fkErr.ID != "0d9e5f1c-7f3a-4a2e-9d3b-1f0e8c6a4b21" {
		t.Errorf("ID = %v", fkErr.ID)
	}
	if fkErr.Suggestion == "" {
		t.Error("expected a suggestion")
	}
}

func TestExtractDependentHelpers(t *testing.T) {
	detail := `Key (id)=(abc-123) is still referenced from table "order_items".`

	if got := extractDependentTableFromDetail(detail); got != "order_items" {
		t.Errorf("extractDependentTableFromDetail() = %q", got)
	}
	if got := extractKeyValueFromDetail(detail); got != "abc-123" {
		t.Errorf("extractKeyValueFromDetail() = %q", got)
	}

	// The insert-side detail is not a "still referenced" detail.
	insertDetail := `Key (author_id)=(x) is not present in table "users".`
	if got := extractDependentTableFromDetail(insertDetail); got != "" {
		t.Errorf("extractDependentTableFromDetail(insert detail) = %q, want empty", got)
	}
	if got := extractDependentTableFromDetail(""); got != "" {
		t.Errorf("empty detail = %q", got)
	}
	if got := extractKeyValueFromDetail("no key here"); got != "" {
		t.Errorf("extractKeyValueFromDetail(no key) = %q", got)
	}
}

func TestMapDatabaseErrorNotNullAndCheckAndUnknown(t *testing.T) {
	notNullErr := &pgconn.PgError{Code: "23502", ColumnName: "email"}
	err := mapDatabaseError(notNullErr, "User", "insert", nil)
	if _, ok := err.(*engine.NotNullError); !ok {
		t.Fatalf("expected NotNullError, got %T", err)
	}

	checkErr := &pgconn.PgError{Code: "23514", Message: `new row for relation "users" violates check constraint "age_positive"`, ConstraintName: "age_positive"}
	err = mapDatabaseError(checkErr, "User", "insert", nil)
	if _, ok := err.(*engine.ConstraintError); !ok {
		t.Fatalf("expected ConstraintError, got %T", err)
	}

	undefinedTableErr := &pgconn.PgError{Code: "42P01"}
	err = mapDatabaseError(undefinedTableErr, "User", "insert", nil)
	if _, ok := err.(*engine.ValidationError); !ok {
		t.Fatalf("expected ValidationError for undefined table, got %T", err)
	}

	undefinedColumnErr := &pgconn.PgError{Code: "42703", Message: `column "bad_field" does not exist`}
	err = mapDatabaseError(undefinedColumnErr, "User", "insert", nil)
	if _, ok := err.(*engine.UnknownFieldError); !ok {
		t.Fatalf("expected UnknownFieldError for undefined column, got %T", err)
	}

	unknownErr := &pgconn.PgError{Code: "99999", Message: "weird pg error"}
	err = mapDatabaseError(unknownErr, "User", "insert", nil)
	if err == nil {
		t.Fatal("expected fallback error")
	}
}
