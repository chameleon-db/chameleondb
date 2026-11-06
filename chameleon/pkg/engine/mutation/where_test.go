package mutation

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildWhere_EqualityAndOperators(t *testing.T) {
	clauses, values, err := buildWhere(map[string]interface{}{
		"id":      "abc",
		"age:gte": 18,
		"age:lt":  65,
	}, 1)
	if err != nil {
		t.Fatalf("buildWhere() error = %v", err)
	}

	// Keys are sorted: "age:gte", "age:lt", "id".
	wantClauses := []string{`"age" >= $1`, `"age" < $2`, `"id" = $3`}
	if !reflect.DeepEqual(clauses, wantClauses) {
		t.Fatalf("clauses = %v, want %v", clauses, wantClauses)
	}
	if !reflect.DeepEqual(values, []interface{}{18, 65, "abc"}) {
		t.Fatalf("values = %v", values)
	}
}

func TestBuildWhere_StartsAtGivenParameter(t *testing.T) {
	clauses, _, err := buildWhere(map[string]interface{}{"name": "x"}, 4)
	if err != nil {
		t.Fatalf("buildWhere() error = %v", err)
	}
	if clauses[0] != `"name" = $4` {
		t.Fatalf("clause = %q", clauses[0])
	}
}

func TestBuildWhere_IsDeterministic(t *testing.T) {
	filters := map[string]interface{}{"a": 1, "b": 2, "c:gt": 3, "d:neq": 4, "e:like": "x", "f": 6}

	first, firstValues, err := buildWhere(filters, 1)
	if err != nil {
		t.Fatalf("buildWhere() error = %v", err)
	}

	for i := 0; i < 50; i++ {
		again, againValues, _ := buildWhere(filters, 1)
		if !reflect.DeepEqual(first, again) || !reflect.DeepEqual(firstValues, againValues) {
			t.Fatalf("buildWhere is not deterministic:\n%v %v\n%v %v", first, firstValues, again, againValues)
		}
	}
}

func TestBuildWhere_NilMatchesNull(t *testing.T) {
	clauses, values, err := buildWhere(map[string]interface{}{
		"age":            nil,
		"deleted_at:neq": nil,
		"name":           "ana",
	}, 1)
	if err != nil {
		t.Fatalf("buildWhere() error = %v", err)
	}

	want := []string{`"age" IS NULL`, `"deleted_at" IS NOT NULL`, `"name" = $1`}
	if !reflect.DeepEqual(clauses, want) {
		t.Fatalf("clauses = %v, want %v", clauses, want)
	}
	if !reflect.DeepEqual(values, []interface{}{"ana"}) {
		t.Fatalf("NULL conditions must not consume parameters, values = %v", values)
	}
}

func TestBuildWhere_NilWithOtherOperatorIsAnError(t *testing.T) {
	for _, op := range []string{"gt", "gte", "lt", "lte", "like", "ilike"} {
		_, _, err := buildWhere(map[string]interface{}{"age:" + op: nil}, 1)
		if err == nil || !strings.Contains(err.Error(), "nil value") {
			t.Errorf("operator %s with nil: expected an error about nil, got %v", op, err)
		}
	}
}

func TestBuildWhere_LikeIsContainsWithEscaping(t *testing.T) {
	clauses, values, err := buildWhere(map[string]interface{}{
		"name:like":   `50%_off\sale`,
		"email:ilike": "ANA",
	}, 1)
	if err != nil {
		t.Fatalf("buildWhere() error = %v", err)
	}

	want := []string{`"email" ILIKE $1 ESCAPE '\'`, `"name" LIKE $2 ESCAPE '\'`}
	if !reflect.DeepEqual(clauses, want) {
		t.Fatalf("clauses = %v, want %v", clauses, want)
	}

	wantValues := []interface{}{`%ANA%`, `%50\%\_off\\sale%`}
	if !reflect.DeepEqual(values, wantValues) {
		t.Fatalf("values = %v, want %v", values, wantValues)
	}
}

func TestBuildWhere_LikeNeedsString(t *testing.T) {
	_, _, err := buildWhere(map[string]interface{}{"age:like": 5}, 1)
	if err == nil || !strings.Contains(err.Error(), "string value") {
		t.Fatalf("expected a string-value error, got %v", err)
	}
}

func TestBuildWhere_RejectsBadOperatorAndIdentifier(t *testing.T) {
	if _, _, err := buildWhere(map[string]interface{}{"age:between": 1}, 1); err == nil {
		t.Error("expected an error for an unsupported operator")
	}
	if _, _, err := buildWhere(map[string]interface{}{`id"; DROP TABLE users; --`: 1}, 1); err == nil {
		t.Error("expected an error for an invalid identifier")
	}
}

func TestEscapeLikePattern(t *testing.T) {
	tests := map[string]string{
		"plain":  "plain",
		"100%":   `100\%`,
		"a_b":    `a\_b`,
		`c:\dir`: `c:\\dir`,
		"":       "",
		"%_\\":   `\%\_\\`,
	}
	for in, want := range tests {
		if got := escapeLikePattern(in); got != want {
			t.Errorf("escapeLikePattern(%q) = %q, want %q", in, got, want)
		}
	}
}
