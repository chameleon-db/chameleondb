package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

func TestNormalizePlaygroundQuery_DefaultsAndOps(t *testing.T) {
	query := engine.QueryJSON{
		Entity: "User",
		Filters: []engine.FilterExpr{
			{
				Condition: &engine.FilterCondition{
					Field: engine.FieldPath{Segments: []string{"name"}},
					Op:    "eq",
					Value: engine.FilterValue{"String": "Ana"},
				},
			},
		},
		OrderBy: []engine.OrderByClause{{Field: "name", Direction: "desc"}},
	}

	normalized := normalizePlaygroundQuery(query)

	if normalized.Filters[0].Condition.Op != "Eq" {
		t.Fatalf("expected filter op Eq, got %q", normalized.Filters[0].Condition.Op)
	}

	if normalized.OrderBy[0].Direction != "Desc" {
		t.Fatalf("expected order direction Desc, got %q", normalized.OrderBy[0].Direction)
	}

	if normalized.Includes == nil || normalized.SelectFields == nil {
		t.Fatal("expected includes/select_fields defaults to be initialized")
	}
}

func TestProcessPlaygroundSQL_InvalidJSON(t *testing.T) {
	response := processPlaygroundSQL([]byte("{"))

	if response.OK {
		t.Fatal("expected response ok=false for invalid JSON")
	}

	if response.Error == nil || response.Error.Stage != "invalid_request" {
		t.Fatalf("expected invalid_request error stage, got %+v", response.Error)
	}
}

func TestProcessPlaygroundSQL_Success(t *testing.T) {
	request := playgroundSQLRequest{
		Schema: `
			entity User {
				id: uuid primary,
				name: string,
			}
		`,
		Query: &engine.QueryJSON{
			Entity:       "User",
			Filters:      []engine.FilterExpr{},
			Includes:     []engine.IncludePath{},
			OrderBy:      []engine.OrderByClause{},
			SelectFields: []string{"id", "name"},
		},
	}

	input, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	response := processPlaygroundSQL(input)
	if !response.OK {
		t.Fatalf("expected successful response, got error: %+v", response.Error)
	}

	if !strings.Contains(response.MainQuery, "SELECT") {
		t.Fatalf("expected main query to contain SELECT, got %q", response.MainQuery)
	}

	if !strings.Contains(response.MainQuery, `FROM "users"`) {
		t.Fatalf("expected main query to contain FROM users, got %q", response.MainQuery)
	}
}

func TestProcessPlaygroundSQL_MutationInsert(t *testing.T) {
	request := playgroundSQLRequest{
		Schema: `
			entity User {
				id: uuid primary,
				email: string,
				verified: bool,
			}
		`,
		Mutation: &playgroundSQLMutation{
			Type:   "insert",
			Entity: "User",
			Set: map[string]interface{}{
				"email":    "new@example.com",
				"verified": true,
			},
		},
	}

	input, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	response := processPlaygroundSQL(input)
	if !response.OK {
		t.Fatalf("expected successful response, got error: %+v", response.Error)
	}

	if !strings.Contains(response.MainQuery, "INSERT INTO users") {
		t.Fatalf("expected INSERT SQL, got %q", response.MainQuery)
	}
}

func TestProcessPlaygroundSQL_MutationUpdate(t *testing.T) {
	request := playgroundSQLRequest{
		Schema: `
			entity User {
				id: uuid primary,
				email: string,
				verified: bool,
			}
		`,
		Mutation: &playgroundSQLMutation{
			Type:   "update",
			Entity: "User",
			Set: map[string]interface{}{
				"verified": true,
			},
			Filters: []playgroundSQLMutationFilter{
				{Field: "email", Op: "eq", Value: "ana@mail.com"},
			},
		},
	}

	input, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	response := processPlaygroundSQL(input)
	if !response.OK {
		t.Fatalf("expected successful response, got error: %+v", response.Error)
	}

	if !strings.Contains(response.MainQuery, "UPDATE users SET") {
		t.Fatalf("expected UPDATE SQL, got %q", response.MainQuery)
	}
	if !strings.Contains(response.MainQuery, "WHERE email = $2") {
		t.Fatalf("expected WHERE clause, got %q", response.MainQuery)
	}
	if strings.Contains(response.MainQuery, "ana@mail.com") {
		t.Fatalf("value must not appear in SQL text, got %q", response.MainQuery)
	}
	if len(response.MainParams) != 2 || response.MainParams[1] != engine.NewSQLParam("ana@mail.com") {
		t.Fatalf("unexpected params: %#v", response.MainParams)
	}
}

func TestProcessPlaygroundSQL_MutationDelete(t *testing.T) {
	request := playgroundSQLRequest{
		Schema: `
			entity User {
				id: uuid primary,
				email: string,
			}
		`,
		Mutation: &playgroundSQLMutation{
			Type:   "delete",
			Entity: "User",
			Filters: []playgroundSQLMutationFilter{
				{Field: "email", Op: "eq", Value: "old@example.com"},
			},
		},
	}

	input, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	response := processPlaygroundSQL(input)
	if !response.OK {
		t.Fatalf("expected successful response, got error: %+v", response.Error)
	}

	if !strings.Contains(response.MainQuery, "DELETE FROM users") {
		t.Fatalf("expected DELETE SQL, got %q", response.MainQuery)
	}
}

func TestProcessPlaygroundSQL_MutationUnknownField(t *testing.T) {
	request := playgroundSQLRequest{
		Schema: `
			entity User {
				id: uuid primary,
				email: string,
			}
		`,
		Mutation: &playgroundSQLMutation{
			Type:   "insert",
			Entity: "User",
			Set: map[string]interface{}{
				"unknown": "x",
			},
		},
	}

	input, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	response := processPlaygroundSQL(input)
	if response.OK {
		t.Fatalf("expected failure for unknown field, got success: %+v", response)
	}

	if response.Error == nil || response.Error.Stage != "query_compilation" {
		t.Fatalf("expected query_compilation error, got %+v", response.Error)
	}
}

func TestPlaygroundSQLHTTPHandler_CompileMethodNotAllowed(t *testing.T) {
	handler := newPlaygroundSQLHTTPHandler()
	req := httptest.NewRequest(http.MethodGet, "/compile", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestPlaygroundSQLHTTPHandler_CompileSuccess(t *testing.T) {
	handler := newPlaygroundSQLHTTPHandler()

	payload := `{
		"schema": "entity User { id: uuid primary, name: string, }",
		"query": {
			"entity": "User",
			"filters": [],
			"includes": [],
			"order_by": [],
			"select_fields": ["id", "name"]
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/compile", strings.NewReader(payload))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var response playgroundSQLResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !response.OK {
		t.Fatalf("expected ok=true, got response=%+v", response)
	}

	if !strings.Contains(response.MainQuery, "SELECT") {
		t.Fatalf("expected main query to contain SELECT, got %q", response.MainQuery)
	}
}

func TestProcessPlaygroundSQL_ValuesAreNotInSQLText(t *testing.T) {
	values := []string{
		"o'brien", "''", "x; y", "a -- b", "/* c */", `back\slash`,
		"caf\u00e9 \u4e2d\u6587", "100%", "a_b", "line\nbreak",
	}

	for _, value := range values {
		request := playgroundSQLRequest{
			Schema: `
				entity User {
					id: uuid primary,
					email: string,
				}
			`,
			Mutation: &playgroundSQLMutation{
				Type:    "update",
				Entity:  "User",
				Set:     map[string]interface{}{"email": value},
				Filters: []playgroundSQLMutationFilter{{Field: "email", Op: "eq", Value: value}},
			},
		}
		input, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		response := processPlaygroundSQL(input)
		if !response.OK {
			t.Fatalf("expected successful response, got error: %+v", response.Error)
		}
		if strings.Contains(response.MainQuery, value) {
			t.Errorf("value %q appears in SQL text %q", value, response.MainQuery)
		}
		if len(response.MainParams) != 2 || response.MainParams[0] != engine.NewSQLParam(value) {
			t.Errorf("unexpected params for %q: %#v", value, response.MainParams)
		}
	}
}

func TestProcessPlaygroundSQL_QueryValuesAreNotInSQLText(t *testing.T) {
	request := playgroundSQLRequest{
		Schema: `
			entity User {
				id: uuid primary,
				email: string,
				orders: [Order] via user_id,
			}
			entity Order {
				id: uuid primary,
				user_id: uuid,
			}
		`,
		Query: &engine.QueryJSON{
			Entity: "User",
			Filters: []engine.FilterExpr{{Condition: &engine.FilterCondition{
				Field: engine.FieldPath{Segments: []string{"email"}},
				Op:    "Eq",
				Value: engine.FilterValue{"String": "o'brien; x"},
			}}},
			Includes: []engine.IncludePath{{Path: []string{"orders"}}},
		},
	}
	input, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	response := processPlaygroundSQL(input)
	if !response.OK {
		t.Fatalf("expected successful response, got error: %+v", response.Error)
	}
	if strings.Contains(response.MainQuery, "brien") {
		t.Errorf("value appears in SQL text %q", response.MainQuery)
	}
	if len(response.MainParams) != 1 || response.MainParams[0] != engine.NewSQLParam("o'brien; x") {
		t.Errorf("unexpected params: %#v", response.MainParams)
	}
	if len(response.EagerQueries) != 1 || len(response.EagerQueries[0].Params) != 1 ||
		response.EagerQueries[0].Params[0].Kind != engine.ParamParentIDs {
		t.Errorf("unexpected eager queries: %#v", response.EagerQueries)
	}
}
