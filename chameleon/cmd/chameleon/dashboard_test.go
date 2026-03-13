package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
)

func newDashboardTestServer(t *testing.T, loadSchema bool) *dashboardServer {
	t.Helper()

	eng := engine.NewEngineWithoutSchema()
	if loadSchema {
		_, err := eng.LoadSchemaFromString(`
entity User {
  id: uuid primary,
  email: string,
}`)
		if err != nil {
			t.Fatalf("failed to load test schema: %v", err)
		}
	}

	workDir := t.TempDir()
	return &dashboardServer{
		eng:     eng,
		workDir: workDir,
		v:       vault.NewVault(workDir),
		jLogger: nil,
	}
}

func decodeJSONBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	return payload
}

func TestDashboardHelpers(t *testing.T) {
	if dashboardIntParam("10", 5) != 10 {
		t.Fatal("expected parsed int param")
	}
	if dashboardIntParam("bad", 5) != 5 {
		t.Fatal("expected default value for bad int")
	}
	if !isDashboardLocalOrigin("http://localhost:3000") {
		t.Fatal("expected localhost origin to be allowed")
	}
	if isDashboardLocalOrigin("https://example.com") {
		t.Fatal("expected non-local origin to be denied")
	}
	if got := dashboardQuoteIdent(`we"ird`); got != `"we""ird"` {
		t.Fatalf("unexpected quoted identifier: %s", got)
	}
	if got := dashboardTableName("OrderItem"); got != "order_items" {
		t.Fatalf("unexpected table name: %s", got)
	}
}

func TestHandleHealth(t *testing.T) {
	srv := newDashboardTestServer(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	srv.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	payload := decodeJSONBody(t, rec)
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", payload["ok"])
	}
	if payload["schema_loaded"] != false {
		t.Fatalf("expected schema_loaded=false, got %#v", payload["schema_loaded"])
	}
	if payload["db_connected"] != false {
		t.Fatalf("expected db_connected=false, got %#v", payload["db_connected"])
	}
}

func TestHandleSchemaNotLoaded(t *testing.T) {
	srv := newDashboardTestServer(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema", nil)
	rec := httptest.NewRecorder()
	srv.handleSchema(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	payload := decodeJSONBody(t, rec)
	if payload["ok"] != false {
		t.Fatalf("expected ok=false, got %#v", payload["ok"])
	}
}

func TestHandleSchemaAndERDLoaded(t *testing.T) {
	srv := newDashboardTestServer(t, true)

	// /schema
	reqSchema := httptest.NewRequest(http.MethodGet, "/api/v1/schema", nil)
	recSchema := httptest.NewRecorder()
	srv.handleSchema(recSchema, reqSchema)
	if recSchema.Code != http.StatusOK {
		t.Fatalf("expected 200 for schema, got %d", recSchema.Code)
	}
	payloadSchema := decodeJSONBody(t, recSchema)
	if payloadSchema["ok"] != true {
		t.Fatalf("expected ok=true for schema, got %#v", payloadSchema["ok"])
	}

	// /schema/erd
	reqERD := httptest.NewRequest(http.MethodGet, "/api/v1/schema/erd", nil)
	recERD := httptest.NewRecorder()
	srv.handleSchemaERD(recERD, reqERD)
	if recERD.Code != http.StatusOK {
		t.Fatalf("expected 200 for erd, got %d", recERD.Code)
	}
	payloadERD := decodeJSONBody(t, recERD)
	if payloadERD["ok"] != true {
		t.Fatalf("expected ok=true for erd, got %#v", payloadERD["ok"])
	}
	nodes, ok := payloadERD["nodes"].([]interface{})
	if !ok || len(nodes) == 0 {
		t.Fatalf("expected non-empty nodes in erd response, got %#v", payloadERD["nodes"])
	}
}

func TestHandleDBStatusDisconnected(t *testing.T) {
	srv := newDashboardTestServer(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/db/status", nil)
	rec := httptest.NewRecorder()
	srv.handleDBStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	payload := decodeJSONBody(t, rec)
	if payload["connected"] != false {
		t.Fatalf("expected connected=false, got %#v", payload["connected"])
	}
}

func TestHandleMigrationsAndJournalWithoutData(t *testing.T) {
	srv := newDashboardTestServer(t, false)

	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/migrations/status", nil)
	recStatus := httptest.NewRecorder()
	srv.handleMigrationsStatus(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 for migrations status, got %d", recStatus.Code)
	}
	statusPayload := decodeJSONBody(t, recStatus)
	if statusPayload["initialized"] != false {
		t.Fatalf("expected initialized=false, got %#v", statusPayload["initialized"])
	}

	reqMigrations := httptest.NewRequest(http.MethodGet, "/api/v1/migrations", nil)
	recMigrations := httptest.NewRecorder()
	srv.handleMigrations(recMigrations, reqMigrations)
	if recMigrations.Code != http.StatusOK {
		t.Fatalf("expected 200 for migrations, got %d", recMigrations.Code)
	}

	reqJournal := httptest.NewRequest(http.MethodGet, "/api/v1/journal?limit=10", nil)
	recJournal := httptest.NewRecorder()
	srv.handleJournal(recJournal, reqJournal)
	if recJournal.Code != http.StatusOK {
		t.Fatalf("expected 200 for journal, got %d", recJournal.Code)
	}
	journalPayload := decodeJSONBody(t, recJournal)
	if journalPayload["ok"] != true {
		t.Fatalf("expected ok=true for journal, got %#v", journalPayload["ok"])
	}
}

func TestDashboardCORSMiddleware(t *testing.T) {
	h := dashboardCORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// local origin should be allowed
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("expected CORS allow origin header, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	// OPTIONS should return 204
	optReq := httptest.NewRequest(http.MethodOptions, "/", nil)
	optReq.Header.Set("Origin", "http://localhost:3000")
	optRec := httptest.NewRecorder()
	h.ServeHTTP(optRec, optReq)
	if optRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", optRec.Code)
	}
}
