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
