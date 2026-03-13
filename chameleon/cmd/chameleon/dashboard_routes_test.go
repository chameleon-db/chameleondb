package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
)

func TestDashboardRoutesAndHelpers(t *testing.T) {
	eng := engine.NewEngineWithoutSchema()
	workDir := t.TempDir()
	srv := &dashboardServer{eng: eng, workDir: workDir, v: vault.NewVault(workDir)}

	h := srv.routes()

	// route dispatch to health handler
	healthReq := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	healthRec := httptest.NewRecorder()
	h.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("expected /health 200, got %d", healthRec.Code)
	}

	// route dispatch to schema handler when schema missing
	schemaReq := httptest.NewRequest(http.MethodGet, "/api/v1/schema", nil)
	schemaRec := httptest.NewRecorder()
	h.ServeHTTP(schemaRec, schemaReq)
	if schemaRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected /schema 503 without schema, got %d", schemaRec.Code)
	}

	// queryDBInfo should return empty values without connector
	dbName, dbVersion := srv.queryDBInfo(schemaReq.Context())
	if dbName != "" || dbVersion != "" {
		t.Fatalf("expected empty db info without connector, got dbName=%q dbVersion=%q", dbName, dbVersion)
	}

	// countRows without connector should fail
	if _, err := srv.countRows(schemaReq.Context(), "users"); err == nil {
		t.Fatal("expected countRows error without connector")
	}

	if got := dashboardFindEntity(nil, "User"); got != nil {
		t.Fatalf("expected nil entity for nil schema, got %#v", got)
	}
	if got := dashboardFindPK(&engine.Entity{}); got != "" {
		t.Fatalf("expected empty pk for entity without fields, got %q", got)
	}

	oldDSN := dashboardDSN
	defer func() { dashboardDSN = oldDSN }()
	dashboardDSN = "postgresql://user:pass@localhost:5432/chameleon"
	if cfg, err := dashboardConnectorConfig(); err != nil || cfg.Database != "chameleon" {
		t.Fatalf("expected connector config from dsn, cfg=%+v err=%v", cfg, err)
	}
}
