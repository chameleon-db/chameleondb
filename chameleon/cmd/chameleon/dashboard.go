package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chameleon-db/chameleondb/chameleon/internal/admin"
	"github.com/chameleon-db/chameleondb/chameleon/internal/journal"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/vault"
	"github.com/spf13/cobra"
)

var (
	dashboardAddr string
	dashboardDSN  string
)

// ─────────────────────────────────────────────────────────────
// Command definitions
// ─────────────────────────────────────────────────────────────

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Local dashboard for ChameleonDB",
	Long:  `Commands for running and managing the local ChameleonDB dashboard.`,
}

var dashboardServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the local dashboard API server",
	Long: `Start a local HTTP API server for the ChameleonDB dashboard.

Binds to 127.0.0.1 by default (localhost access only).
Schema is loaded from the current project directory.
Database connection is optional on startup.

Available endpoints:
  GET    /api/v1/health
  GET    /api/v1/schema
  GET    /api/v1/schema/erd
  GET    /api/v1/db/status
  GET    /api/v1/entities
  GET    /api/v1/entities/{name}/records
  POST   /api/v1/entities/{name}/records
  PATCH  /api/v1/entities/{name}/records/{id}
  DELETE /api/v1/entities/{name}/records/{id}
  GET    /api/v1/migrations/status
  GET    /api/v1/migrations
  GET    /api/v1/journal`,
	RunE: runDashboardServe,
}

func init() {
	dashboardServeCmd.Flags().StringVar(&dashboardAddr, "addr", "127.0.0.1:8091", "HTTP listen address (default: localhost only)")
	dashboardServeCmd.Flags().StringVar(&dashboardDSN, "dsn", "", "Database DSN (overrides config and DATABASE_URL)")
	dashboardCmd.AddCommand(dashboardServeCmd)
	rootCmd.AddCommand(dashboardCmd)
}

// ─────────────────────────────────────────────────────────────
// Server state
// ─────────────────────────────────────────────────────────────

type dashboardServer struct {
	eng     *engine.Engine
	workDir string
	v       *vault.Vault
	jLogger *journal.Logger
}

// ─────────────────────────────────────────────────────────────
// Startup
// ─────────────────────────────────────────────────────────────

func runDashboardServe(cmd *cobra.Command, args []string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Load engine + schema (non-fatal if schema doesn't exist yet)
	eng, err := engine.NewEngine()
	if err != nil {
		printWarning("Schema not loaded: %v (continuing without schema)", err)
		eng = engine.NewEngineWithoutSchema()
	}

	// DB connection — try --dsn first, then config/env
	connCfg, cfgErr := dashboardConnectorConfig()
	if cfgErr == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := eng.Connect(ctx, connCfg); err != nil {
			printWarning("Database connection failed: %v (continuing without DB)", err)
		} else {
			printSuccess("Connected to database")
		}
	}

	// Vault + journal (non-fatal)
	v := vault.NewVault(workDir)
	factory := admin.NewManagerFactory(workDir)
	jLogger, _ := factory.CreateJournalLogger()

	srv := &dashboardServer{
		eng:     eng,
		workDir: workDir,
		v:       v,
		jLogger: jLogger,
	}

	httpSrv := &http.Server{
		Addr:         dashboardAddr,
		Handler:      srv.routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		printInfo("Dashboard API listening on http://%s", dashboardAddr)
		printInfo("API prefix: /api/v1/")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			printError("Dashboard server error: %v", err)
		}
	}()

	<-stop
	printInfo("Shutting down dashboard server...")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()

	if err := httpSrv.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	eng.Close()
	printSuccess("Dashboard server stopped")
	return nil
}

// dashboardConnectorConfig resolves DSN: --dsn flag → config/env fallback.
func dashboardConnectorConfig() (engine.ConnectorConfig, error) {
	if dashboardDSN != "" {
		return engine.ParseConnectionString(dashboardDSN)
	}
	return LoadConnectorConfig()
}

// ─────────────────────────────────────────────────────────────
// Router
// ─────────────────────────────────────────────────────────────

func (s *dashboardServer) routes() http.Handler {
	mux := http.NewServeMux()

	// Group 1 — stateless
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/schema", s.handleSchema)
	mux.HandleFunc("GET /api/v1/schema/erd", s.handleSchemaERD)

	// Group 2 — database
	mux.HandleFunc("GET /api/v1/db/status", s.handleDBStatus)
	mux.HandleFunc("GET /api/v1/entities", s.handleListEntities)
	mux.HandleFunc("GET /api/v1/entities/{name}/records", s.handleListRecords)
	mux.HandleFunc("POST /api/v1/entities/{name}/records", s.handleInsertRecord)
	mux.HandleFunc("PATCH /api/v1/entities/{name}/records/{id}", s.handleUpdateRecord)
	mux.HandleFunc("DELETE /api/v1/entities/{name}/records/{id}", s.handleDeleteRecord)

	// Group 3 — migrations + journal
	mux.HandleFunc("GET /api/v1/migrations/status", s.handleMigrationsStatus)
	mux.HandleFunc("GET /api/v1/migrations", s.handleMigrations)
	mux.HandleFunc("GET /api/v1/journal", s.handleJournal)

	return dashboardCORSMiddleware(mux)
}

// ─────────────────────────────────────────────────────────────
// Group 1 — Stateless handlers
// ─────────────────────────────────────────────────────────────

func (s *dashboardServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	dashboardJSON(w, 200, map[string]interface{}{
		"ok":            true,
		"version":       s.eng.Version(),
		"schema_loaded": s.eng.GetSchema() != nil,
		"db_connected":  s.eng.IsConnected(),
	})
}

func (s *dashboardServer) handleSchema(w http.ResponseWriter, r *http.Request) {
	schema := s.eng.GetSchema()
	if schema == nil {
		dashboardError(w, 503, "schema_not_loaded", "No schema loaded in current directory")
		return
	}
	dashboardJSON(w, 200, map[string]interface{}{
		"ok":     true,
		"schema": schema,
	})
}

func (s *dashboardServer) handleSchemaERD(w http.ResponseWriter, r *http.Request) {
	schema := s.eng.GetSchema()
	if schema == nil {
		dashboardError(w, 503, "schema_not_loaded", "No schema loaded in current directory")
		return
	}

	type erdField struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		Nullable   bool   `json:"nullable"`
		Unique     bool   `json:"unique"`
		PrimaryKey bool   `json:"primary_key"`
	}
	type erdNode struct {
		ID     string     `json:"id"`
		Table  string     `json:"table"`
		PK     string     `json:"pk"`
		Fields []erdField `json:"fields"`
	}
	type erdEdge struct {
		From string `json:"from"`
		To   string `json:"to"`
		Kind string `json:"kind"`
		Via  string `json:"via,omitempty"`
	}

	nodes := []erdNode{}
	edges := []erdEdge{}

	for _, entity := range schema.Entities {
		node := erdNode{
			ID:     entity.Name,
			Table:  dashboardTableName(entity.Name),
			Fields: []erdField{},
		}
		for _, field := range entity.Fields {
			if field.PrimaryKey {
				node.PK = field.Name
			}
			node.Fields = append(node.Fields, erdField{
				Name:       field.Name,
				Type:       field.Type.String(),
				Nullable:   field.Nullable,
				Unique:     field.Unique,
				PrimaryKey: field.PrimaryKey,
			})
		}
		nodes = append(nodes, node)

		for _, rel := range entity.Relations {
			edge := erdEdge{
				From: entity.Name,
				To:   rel.TargetEntity,
				Kind: string(rel.Kind),
			}
			if rel.ForeignKey != nil {
				edge.Via = *rel.ForeignKey
			}
			edges = append(edges, edge)
		}
	}

	dashboardJSON(w, 200, map[string]interface{}{
		"ok":    true,
		"nodes": nodes,
		"edges": edges,
	})
}

// ─────────────────────────────────────────────────────────────
// Group 2 — Database handlers
// ─────────────────────────────────────────────────────────────

func (s *dashboardServer) handleDBStatus(w http.ResponseWriter, r *http.Request) {
	if !s.eng.IsConnected() {
		dashboardJSON(w, 200, map[string]interface{}{
			"ok":        true,
			"connected": false,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	start := time.Now()
	pingErr := s.eng.Ping(ctx)
	latencyMs := time.Since(start).Milliseconds()

	if pingErr != nil {
		dashboardJSON(w, 200, map[string]interface{}{
			"ok":        true,
			"connected": false,
			"error":     pingErr.Error(),
		})
		return
	}

	dbName, dbVersion := s.queryDBInfo(ctx)
	dashboardJSON(w, 200, map[string]interface{}{
		"ok":         true,
		"connected":  true,
		"latency_ms": latencyMs,
		"db_name":    dbName,
		"db_version": dbVersion,
	})
}

func (s *dashboardServer) queryDBInfo(ctx context.Context) (dbName, dbVersion string) {
	connector := s.eng.Connector()
	if connector == nil {
		return
	}
	rows, err := connector.Query(ctx, "SELECT current_database() AS db_name, version() AS db_version")
	if err != nil || len(rows) == 0 {
		return
	}
	if v, ok := rows[0]["db_name"].(string); ok {
		dbName = v
	}
	if v, ok := rows[0]["db_version"].(string); ok {
		dbVersion = v
	}
	return
}

func (s *dashboardServer) handleListEntities(w http.ResponseWriter, r *http.Request) {
	schema := s.eng.GetSchema()
	if schema == nil {
		dashboardError(w, 503, "schema_not_loaded", "No schema loaded in current directory")
		return
	}

	type entitySummary struct {
		Name     string `json:"name"`
		Table    string `json:"table"`
		PK       string `json:"pk"`
		Fields   int    `json:"fields"`
		RowCount *int64 `json:"row_count,omitempty"`
	}

	result := make([]entitySummary, 0, len(schema.Entities))

	for _, entity := range schema.Entities {
		summary := entitySummary{
			Name:   entity.Name,
			Table:  dashboardTableName(entity.Name),
			Fields: len(entity.Fields),
		}
		for _, f := range entity.Fields {
			if f.PrimaryKey {
				summary.PK = f.Name
			}
		}
		if s.eng.IsConnected() {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			if count, err := s.countRows(ctx, summary.Table); err == nil {
				summary.RowCount = &count
			}
			cancel()
		}
		result = append(result, summary)
	}

	dashboardJSON(w, 200, map[string]interface{}{
		"ok":       true,
		"entities": result,
	})
}

func (s *dashboardServer) handleListRecords(w http.ResponseWriter, r *http.Request) {
	if !s.eng.IsConnected() {
		dashboardError(w, 503, "db_disconnected", "No database connection")
		return
	}

	name := r.PathValue("name")
	entity := dashboardFindEntity(s.eng.GetSchema(), name)
	if entity == nil {
		dashboardError(w, 404, "entity_not_found", fmt.Sprintf("Entity '%s' not found in schema", name))
		return
	}

	q := r.URL.Query()
	page := dashboardIntParam(q.Get("page"), 1)
	pageSize := dashboardIntParam(q.Get("page_size"), 50)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 50
	}

	ctx := r.Context()

	total, err := s.countRows(ctx, dashboardTableName(name))
	if err != nil {
		dashboardError(w, 500, "count_failed", err.Error())
		return
	}

	offset := uint64((page - 1) * pageSize)
	limit := uint64(pageSize)

	qb := s.eng.Query(name).Limit(limit).Offset(offset)
	if sortField := q.Get("sort"); sortField != "" {
		dir := q.Get("dir")
		if dir == "" {
			dir = "asc"
		}
		qb = qb.OrderBy(sortField, dir)
	}

	result, err := qb.Execute(ctx)
	if err != nil {
		dashboardError(w, 500, "query_failed", err.Error())
		return
	}

	rows := result.Rows
	if rows == nil {
		rows = []engine.Row{}
	}

	totalPages := int64(0)
	if pageSize > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}

	dashboardJSON(w, 200, map[string]interface{}{
		"ok": true,
		"data": map[string]interface{}{
			"records": rows,
			"pagination": map[string]interface{}{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": totalPages,
			},
		},
	})
}

func (s *dashboardServer) handleInsertRecord(w http.ResponseWriter, r *http.Request) {
	if !s.eng.IsConnected() {
		dashboardError(w, 503, "db_disconnected", "No database connection")
		return
	}

	name := r.PathValue("name")
	if dashboardFindEntity(s.eng.GetSchema(), name) == nil {
		dashboardError(w, 404, "entity_not_found", fmt.Sprintf("Entity '%s' not found in schema", name))
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		dashboardError(w, 400, "invalid_body", "Invalid JSON body")
		return
	}

	ins := s.eng.Insert(name)
	for k, v := range body {
		ins = ins.Set(k, v)
	}

	result, err := ins.Execute(r.Context())
	if err != nil {
		dashboardError(w, 500, "insert_failed", err.Error())
		return
	}

	dashboardJSON(w, 201, map[string]interface{}{
		"ok":   true,
		"data": result,
	})
}

func (s *dashboardServer) handleUpdateRecord(w http.ResponseWriter, r *http.Request) {
	if !s.eng.IsConnected() {
		dashboardError(w, 503, "db_disconnected", "No database connection")
		return
	}

	name := r.PathValue("name")
	id := r.PathValue("id")

	entity := dashboardFindEntity(s.eng.GetSchema(), name)
	if entity == nil {
		dashboardError(w, 404, "entity_not_found", fmt.Sprintf("Entity '%s' not found in schema", name))
		return
	}

	pkField := dashboardFindPK(entity)
	if pkField == "" {
		dashboardError(w, 500, "no_primary_key", "Entity has no primary key field")
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		dashboardError(w, 400, "invalid_body", "Invalid JSON body")
		return
	}

	upd := s.eng.Update(name).Filter(pkField, "eq", id)
	for k, v := range body {
		upd = upd.Set(k, v)
	}

	result, err := upd.Execute(r.Context())
	if err != nil {
		dashboardError(w, 500, "update_failed", err.Error())
		return
	}

	dashboardJSON(w, 200, map[string]interface{}{
		"ok":   true,
		"data": result,
	})
}

func (s *dashboardServer) handleDeleteRecord(w http.ResponseWriter, r *http.Request) {
	if !s.eng.IsConnected() {
		dashboardError(w, 503, "db_disconnected", "No database connection")
		return
	}

	name := r.PathValue("name")
	id := r.PathValue("id")

	entity := dashboardFindEntity(s.eng.GetSchema(), name)
	if entity == nil {
		dashboardError(w, 404, "entity_not_found", fmt.Sprintf("Entity '%s' not found in schema", name))
		return
	}

	pkField := dashboardFindPK(entity)
	if pkField == "" {
		dashboardError(w, 500, "no_primary_key", "Entity has no primary key field")
		return
	}

	result, err := s.eng.Delete(name).Filter(pkField, "eq", id).Execute(r.Context())
	if err != nil {
		dashboardError(w, 500, "delete_failed", err.Error())
		return
	}

	dashboardJSON(w, 200, map[string]interface{}{
		"ok":   true,
		"data": result,
	})
}

// ─────────────────────────────────────────────────────────────
// Group 3 — Migrations + journal handlers
// ─────────────────────────────────────────────────────────────

func (s *dashboardServer) handleMigrationsStatus(w http.ResponseWriter, r *http.Request) {
	if !s.v.Exists() {
		dashboardJSON(w, 200, map[string]interface{}{
			"ok":              true,
			"initialized":     false,
			"current_version": nil,
			"total_versions":  0,
		})
		return
	}

	status, err := s.v.GetStatus()
	if err != nil {
		dashboardError(w, 500, "vault_error", err.Error())
		return
	}

	dashboardJSON(w, 200, map[string]interface{}{
		"ok":              true,
		"initialized":     true,
		"current_version": status.CurrentVersion,
		"total_versions":  status.TotalVersions,
		"last_modified":   status.LastModified,
		"integrity_ok":    status.IntegrityOK,
	})
}

func (s *dashboardServer) handleMigrations(w http.ResponseWriter, r *http.Request) {
	if !s.v.Exists() {
		dashboardJSON(w, 200, map[string]interface{}{
			"ok":       true,
			"versions": []interface{}{},
		})
		return
	}

	if err := s.v.Load(); err != nil {
		dashboardError(w, 500, "vault_error", err.Error())
		return
	}

	dashboardJSON(w, 200, map[string]interface{}{
		"ok":       true,
		"versions": s.v.Manifest.Versions,
	})
}

func (s *dashboardServer) handleJournal(w http.ResponseWriter, r *http.Request) {
	if s.jLogger == nil {
		dashboardJSON(w, 200, map[string]interface{}{
			"ok":      true,
			"entries": []interface{}{},
		})
		return
	}

	limit := dashboardIntParam(r.URL.Query().Get("limit"), 50)
	if limit < 1 || limit > 500 {
		limit = 50
	}

	entries, err := s.jLogger.Last(limit)
	if err != nil {
		dashboardError(w, 500, "journal_error", err.Error())
		return
	}

	if entries == nil {
		entries = []*journal.Entry{}
	}

	dashboardJSON(w, 200, map[string]interface{}{
		"ok":      true,
		"entries": entries,
	})
}

// ─────────────────────────────────────────────────────────────
// Helpers — DB
// ─────────────────────────────────────────────────────────────

// countRows runs SELECT COUNT(*) against a table using the raw connector.
func (s *dashboardServer) countRows(ctx context.Context, table string) (int64, error) {
	connector := s.eng.Connector()
	if connector == nil {
		return 0, fmt.Errorf("no connector")
	}
	rows, err := connector.Query(ctx, "SELECT COUNT(*) AS count FROM "+dashboardQuoteIdent(table))
	if err != nil || len(rows) == 0 {
		return 0, err
	}
	switch v := rows[0]["count"].(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	default:
		return 0, nil
	}
}

// ─────────────────────────────────────────────────────────────
// Helpers — Schema
// ─────────────────────────────────────────────────────────────

func dashboardFindEntity(schema *engine.Schema, name string) *engine.Entity {
	if schema == nil {
		return nil
	}
	for _, e := range schema.Entities {
		if e.Name == name {
			return e
		}
	}
	return nil
}

func dashboardFindPK(entity *engine.Entity) string {
	for _, f := range entity.Fields {
		if f.PrimaryKey {
			return f.Name
		}
	}
	return ""
}

// dashboardTableName derives the SQL table name from an entity name.
// Reuses toSnakeCase + pluralize from playground_sql.go (same package).
func dashboardTableName(entityName string) string {
	return pluralize(toSnakeCase(entityName))
}

// dashboardQuoteIdent safely quotes a PostgreSQL identifier.
func dashboardQuoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// ─────────────────────────────────────────────────────────────
// Helpers — HTTP
// ─────────────────────────────────────────────────────────────

func dashboardJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func dashboardError(w http.ResponseWriter, status int, code, message string) {
	dashboardJSON(w, status, map[string]interface{}{
		"ok": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func dashboardIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// dashboardCORSMiddleware allows cross-origin requests from localhost only.
func dashboardCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isDashboardLocalOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isDashboardLocalOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		origin == "http://localhost" ||
		origin == "http://127.0.0.1"
}
