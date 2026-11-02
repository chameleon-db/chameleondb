package engine

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Host != "localhost" {
		t.Errorf("Expected host localhost, got %s", config.Host)
	}
	if config.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", config.Port)
	}
	if config.MaxConns != 10 {
		t.Errorf("Expected max conns 10, got %d", config.MaxConns)
	}
}

func TestConnectionString(t *testing.T) {
	config := ConnectorConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "chameleon",
		User:     "postgres",
		Password: "secret",
	}

	connStr := config.ConnectionString()

	assertContains(t, connStr, "host=localhost")
	assertContains(t, connStr, "port=5432")
	assertContains(t, connStr, "dbname=chameleon")
	assertContains(t, connStr, "user=postgres")
	assertContains(t, connStr, "password=secret")
	assertContains(t, connStr, "sslmode=disable")
}

func TestNewConnectorNotConnected(t *testing.T) {
	connector := NewConnector(DefaultConfig())

	if connector.IsConnected() {
		t.Error("New connector should not be connected")
	}
	if connector.Pool() != nil {
		t.Error("Pool should be nil before Connect()")
	}
}

func TestBindArgsParentIDs(t *testing.T) {
	ids := []interface{}{"uuid-1", "uuid-2", "uuid-3"}

	args, err := bindArgs([]SQLParam{{Kind: ParamParentIDs}}, ids)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(args))
	}
	bound, ok := args[0].([]interface{})
	if !ok || len(bound) != 3 || bound[0] != "uuid-1" || bound[2] != "uuid-3" {
		t.Fatalf("Unexpected bound ids: %#v", args[0])
	}
}

func TestBindArgsEmptyParentIDs(t *testing.T) {
	args, err := bindArgs([]SQLParam{{Kind: ParamParentIDs}}, []interface{}{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	bound, ok := args[0].([]interface{})
	if !ok || len(bound) != 0 {
		t.Fatalf("Expected empty id list, got %#v", args[0])
	}
}

func TestBindArgsValues(t *testing.T) {
	params := []SQLParam{
		{Kind: ParamString, Value: "x"},
		{Kind: ParamInt, Value: int64(7)},
		{Kind: ParamFloat, Value: 1.5},
		{Kind: ParamBool, Value: true},
		{Kind: ParamNull},
	}

	args, err := bindArgs(params, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	want := []interface{}{"x", int64(7), 1.5, true, nil}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("arg %d: got %#v, want %#v", i, args[i], want[i])
		}
	}
}

func TestBindArgsRejectsInvalidParams(t *testing.T) {
	if _, err := bindArgs([]SQLParam{{Kind: ParamParentIDs}}, nil); err == nil {
		t.Error("Expected error for parent ids without a parent query")
	}
	if _, err := bindArgs([]SQLParam{{Kind: "Other"}}, nil); err == nil {
		t.Error("Expected error for unknown parameter kind")
	}
}

func TestExtractIDs(t *testing.T) {
	rows := []Row{
		{"id": "uuid-1", "name": "User 1"},
		{"id": "uuid-2", "name": "User 2"},
		{"id": "uuid-3", "name": "User 3"},
	}

	ids := extractIDs(rows, "id")

	if len(ids) != 3 {
		t.Fatalf("Expected 3 IDs, got %d", len(ids))
	}
	if ids[0] != "uuid-1" {
		t.Errorf("Expected uuid-1, got %v", ids[0])
	}
}

func TestExtractIDsMissingField(t *testing.T) {
	rows := []Row{
		{"name": "User 1"},
		{"name": "User 2"},
	}

	ids := extractIDs(rows, "id")

	if len(ids) != 0 {
		t.Errorf("Expected 0 IDs for missing field, got %d", len(ids))
	}
}

func TestInferEntityNameFromRelation(t *testing.T) {
	if got := inferEntityNameFromRelation("posts"); got != "Post" {
		t.Errorf("Expected Post, got %s", got)
	}

	if got := inferEntityNameFromRelation("orders.items"); got != "Item" {
		t.Errorf("Expected Item, got %s", got)
	}
}

func TestRelationParentPath(t *testing.T) {
	parent, ok := relationParentPath("orders.items")
	if !ok {
		t.Fatal("Expected nested relation to have parent path")
	}
	if parent != "orders" {
		t.Errorf("Expected parent orders, got %s", parent)
	}

	_, ok = relationParentPath("orders")
	if ok {
		t.Error("Expected top-level relation to have no parent path")
	}
}

func TestRowHelpers(t *testing.T) {
	row := Row{
		"name":  "Ana",
		"age":   int64(25),
		"email": "ana@mail.com",
	}

	if row.String("name") != "Ana" {
		t.Errorf("Expected Ana, got %s", row.String("name"))
	}
	if row.Int("age") != 25 {
		t.Errorf("Expected 25, got %d", row.Int("age"))
	}
	if row.String("missing") != "" {
		t.Errorf("Expected empty string for missing field")
	}
	if row.Int("missing") != 0 {
		t.Errorf("Expected 0 for missing field")
	}
}

func TestQueryResultHelpers(t *testing.T) {
	result := &QueryResult{
		Entity: "User",
		Rows: []Row{
			{"id": "1", "name": "Ana"},
			{"id": "2", "name": "Bob"},
		},
		Relations: map[string][]Row{},
	}

	if result.Count() != 2 {
		t.Errorf("Expected count 2, got %d", result.Count())
	}
	if result.IsEmpty() {
		t.Error("Expected non-empty result")
	}

	empty := &QueryResult{Entity: "User", Rows: []Row{}}
	if !empty.IsEmpty() {
		t.Error("Expected empty result")
	}
}

func TestEngineNotConnected(t *testing.T) {
	eng := NewEngineWithoutSchema()

	if eng.IsConnected() {
		t.Error("New engine should not be connected")
	}
}
