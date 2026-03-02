package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/chameleon-db/chameleondb/chameleon/internal/ffi"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/spf13/cobra"
)

var (
	playgroundSQLInputFile string
	playgroundSQLServeAddr string
	playgroundSQLMu        sync.Mutex
)

type playgroundSQLRequest struct {
	Schema   string                 `json:"schema"`
	Query    *engine.QueryJSON      `json:"query,omitempty"`
	Mutation *playgroundSQLMutation `json:"mutation,omitempty"`
}

type playgroundSQLMutation struct {
	Type    string                        `json:"type"`
	Entity  string                        `json:"entity"`
	Set     map[string]interface{}        `json:"set,omitempty"`
	Filters []playgroundSQLMutationFilter `json:"filters,omitempty"`
}

type playgroundSQLMutationFilter struct {
	Field string      `json:"field"`
	Op    string      `json:"op"`
	Value interface{} `json:"value"`
}

type playgroundSQLError struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

type playgroundSQLResponse struct {
	OK           bool                `json:"ok"`
	MainQuery    string              `json:"main_query,omitempty"`
	EagerQueries []engine.EagerQuery `json:"eager_queries,omitempty"`
	Error        *playgroundSQLError `json:"error,omitempty"`
}

var playgroundSQLCmd = &cobra.Command{
	Use:   "playground-sql [--input payload.json]",
	Short: "Compile SQL from JSON payload (schema + query) for Playground MVP",
	Long: `Compile SQL without DB connection using a single JSON payload.

The command validates the schema from the payload, compiles the query against
that schema, and returns JSON with generated SQL.

Payload format:
  {
    "schema": "entity User { id: uuid primary, name: string, }",
    "query": {
      "entity": "User",
      "filters": [],
      "includes": [],
      "order_by": [],
      "select_fields": ["id", "name"]
    }
  }

Mutation payload format:
	{
		"schema": "entity User { id: uuid primary, email: string, verified: bool, }",
		"mutation": {
			"type": "update",
			"entity": "User",
			"set": { "verified": true },
			"filters": [
				{ "field": "email", "op": "eq", "value": "ana@mail.com" }
			]
		}
	}

Examples:
  cat payload.json | chameleon playground-sql
  chameleon playground-sql --input payload.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		input, err := readPlaygroundInput(playgroundSQLInputFile)
		if err != nil {
			return writePlaygroundResponse(playgroundSQLResponse{
				OK: false,
				Error: &playgroundSQLError{
					Stage:   "invalid_request",
					Message: err.Error(),
				},
			})
		}

		playgroundSQLMu.Lock()
		response := processPlaygroundSQL(input)
		playgroundSQLMu.Unlock()

		return writePlaygroundResponse(response)
	},
}

func init() {
	playgroundSQLCmd.Flags().StringVar(&playgroundSQLInputFile, "input", "", "path to JSON payload file (if omitted, reads from stdin)")
	playgroundSQLServeCmd.Flags().StringVar(&playgroundSQLServeAddr, "addr", ":8090", "HTTP listen address")

	playgroundSQLCmd.AddCommand(playgroundSQLServeCmd)
	rootCmd.AddCommand(playgroundSQLCmd)
}

var playgroundSQLServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run a minimal HTTP server for playground SQL compilation",
	Long: `Run a local HTTP server that receives schema+query JSON and returns SQL.

Endpoints:
  GET  /healthz
  POST /compile

This mode is intentionally simple for MVP playground usage and serializes
compile requests with a process-level mutex.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		handler := newPlaygroundSQLHTTPHandler()

		printInfo("Playground SQL server listening on %s", playgroundSQLServeAddr)
		printInfo("POST /compile")

		srv := &http.Server{
			Addr:    playgroundSQLServeAddr,
			Handler: handler,
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("playground SQL server error: %w", err)
		}

		return nil
	},
}

func readPlaygroundInput(inputFile string) ([]byte, error) {
	if strings.TrimSpace(inputFile) != "" {
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read input file: %w", err)
		}
		return data, nil
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect stdin: %w", err)
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, fmt.Errorf("no input provided: use --input <file> or pipe JSON via stdin")
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}

	return data, nil
}

func processPlaygroundSQL(input []byte) playgroundSQLResponse {
	var request playgroundSQLRequest
	if err := json.Unmarshal(input, &request); err != nil {
		return playgroundSQLResponse{
			OK: false,
			Error: &playgroundSQLError{
				Stage:   "invalid_request",
				Message: fmt.Sprintf("invalid JSON payload: %v", err),
			},
		}
	}

	if strings.TrimSpace(request.Schema) == "" {
		return playgroundSQLResponse{
			OK: false,
			Error: &playgroundSQLError{
				Stage:   "invalid_request",
				Message: "schema is required",
			},
		}
	}

	if request.Query == nil && request.Mutation == nil {
		return playgroundSQLResponse{
			OK: false,
			Error: &playgroundSQLError{
				Stage:   "invalid_request",
				Message: "either query or mutation is required",
			},
		}
	}

	if request.Query != nil && request.Mutation != nil {
		return playgroundSQLResponse{
			OK: false,
			Error: &playgroundSQLError{
				Stage:   "invalid_request",
				Message: "query and mutation are mutually exclusive",
			},
		}
	}

	eng := engine.NewEngineForCLI()
	if _, rawErr, err := eng.LoadSchemaFromStringRaw(request.Schema); err != nil {
		message := strings.TrimSpace(rawErr)
		if message == "" {
			message = err.Error()
		}

		return playgroundSQLResponse{
			OK: false,
			Error: &playgroundSQLError{
				Stage:   "schema_validation",
				Message: message,
			},
		}
	}

	if request.Mutation != nil {
		sql, err := buildMutationSQL(eng.GetSchema(), request.Mutation)
		if err != nil {
			return playgroundSQLResponse{
				OK: false,
				Error: &playgroundSQLError{
					Stage:   "query_compilation",
					Message: strings.TrimSpace(err.Error()),
				},
			}
		}

		return playgroundSQLResponse{
			OK:        true,
			MainQuery: sql,
		}
	}

	if strings.TrimSpace(request.Query.Entity) == "" {
		return playgroundSQLResponse{
			OK: false,
			Error: &playgroundSQLError{
				Stage:   "invalid_request",
				Message: "query.entity is required",
			},
		}
	}

	query := normalizePlaygroundQuery(*request.Query)

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return playgroundSQLResponse{
			OK: false,
			Error: &playgroundSQLError{
				Stage:   "invalid_request",
				Message: fmt.Sprintf("failed to serialize query: %v", err),
			},
		}
	}

	schemaJSON, err := json.Marshal(eng.GetSchema())
	if err != nil {
		return playgroundSQLResponse{
			OK: false,
			Error: &playgroundSQLError{
				Stage:   "internal",
				Message: fmt.Sprintf("failed to serialize schema: %v", err),
			},
		}
	}

	resultJSON, err := ffi.GenerateSQL(string(queryJSON), string(schemaJSON))
	if err != nil {
		return playgroundSQLResponse{
			OK: false,
			Error: &playgroundSQLError{
				Stage:   "query_compilation",
				Message: strings.TrimSpace(err.Error()),
			},
		}
	}

	var generated engine.GeneratedSQL
	if err := json.Unmarshal([]byte(resultJSON), &generated); err != nil {
		return playgroundSQLResponse{
			OK: false,
			Error: &playgroundSQLError{
				Stage:   "internal",
				Message: fmt.Sprintf("failed to parse SQL output: %v", err),
			},
		}
	}

	response := playgroundSQLResponse{
		OK:        true,
		MainQuery: generated.MainQuery,
	}

	if len(generated.EagerQueries) > 0 {
		eagerQueries := make([]engine.EagerQuery, 0, len(generated.EagerQueries))
		for _, item := range generated.EagerQueries {
			if len(item) >= 2 {
				eagerQueries = append(eagerQueries, engine.EagerQuery{
					Relation: item[0],
					SQL:      item[1],
				})
			}
		}
		response.EagerQueries = eagerQueries
	}

	return response
}

func newPlaygroundSQLHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		writePlaygroundHTTPResponse(w, http.StatusOK, playgroundSQLResponse{OK: true})
	})

	mux.HandleFunc("/compile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024))
		if err != nil {
			writePlaygroundHTTPResponse(w, http.StatusBadRequest, playgroundSQLResponse{
				OK: false,
				Error: &playgroundSQLError{
					Stage:   "invalid_request",
					Message: fmt.Sprintf("failed to read request body: %v", err),
				},
			})
			return
		}

		playgroundSQLMu.Lock()
		response := processPlaygroundSQL(body)
		playgroundSQLMu.Unlock()

		status := http.StatusOK
		if !response.OK {
			switch response.Error.Stage {
			case "invalid_request":
				status = http.StatusBadRequest
			case "schema_validation", "query_compilation":
				status = http.StatusUnprocessableEntity
			default:
				status = http.StatusInternalServerError
			}
		}

		writePlaygroundHTTPResponse(w, status, response)
	})

	return mux
}

func writePlaygroundResponse(response playgroundSQLResponse) error {
	output, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize response: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

func writePlaygroundHTTPResponse(w http.ResponseWriter, status int, response playgroundSQLResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	output, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		http.Error(w, "failed to serialize response", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(output)
	_, _ = w.Write([]byte("\n"))
}

func normalizePlaygroundQuery(q engine.QueryJSON) engine.QueryJSON {
	if q.Filters == nil {
		q.Filters = []engine.FilterExpr{}
	}
	if q.Includes == nil {
		q.Includes = []engine.IncludePath{}
	}
	if q.OrderBy == nil {
		q.OrderBy = []engine.OrderByClause{}
	}
	if q.SelectFields == nil {
		q.SelectFields = []string{}
	}

	for idx := range q.Filters {
		normalizeFilterExpr(&q.Filters[idx])
	}

	for idx := range q.OrderBy {
		if strings.EqualFold(q.OrderBy[idx].Direction, "desc") {
			q.OrderBy[idx].Direction = "Desc"
		} else if strings.EqualFold(q.OrderBy[idx].Direction, "asc") || q.OrderBy[idx].Direction == "" {
			q.OrderBy[idx].Direction = "Asc"
		}
	}

	return q
}

func normalizeFilterExpr(expr *engine.FilterExpr) {
	if expr == nil {
		return
	}

	if expr.Condition != nil {
		expr.Condition.Op = normalizeFilterOp(expr.Condition.Op)
	}

	if expr.Binary != nil {
		normalizeFilterExpr(&expr.Binary.Left)
		normalizeFilterExpr(&expr.Binary.Right)
		if strings.EqualFold(expr.Binary.Op, "or") {
			expr.Binary.Op = "Or"
		} else if strings.EqualFold(expr.Binary.Op, "and") {
			expr.Binary.Op = "And"
		}
	}
}

func normalizeFilterOp(op string) string {
	ops := map[string]string{
		"eq":   "Eq",
		"neq":  "Neq",
		"gt":   "Gt",
		"gte":  "Gte",
		"lt":   "Lt",
		"lte":  "Lte",
		"like": "Like",
		"in":   "In",
	}

	if v, ok := ops[strings.ToLower(strings.TrimSpace(op))]; ok {
		return v
	}
	return op
}

func buildMutationSQL(schema *engine.Schema, mutation *playgroundSQLMutation) (string, error) {
	if mutation == nil {
		return "", fmt.Errorf("mutation payload is nil")
	}

	entityName := strings.TrimSpace(mutation.Entity)
	if entityName == "" {
		return "", fmt.Errorf("mutation.entity is required")
	}

	entity := findSchemaEntity(schema, entityName)
	if entity == nil {
		return "", fmt.Errorf("entity '%s' not found in schema", entityName)
	}

	mutationType := strings.ToLower(strings.TrimSpace(mutation.Type))
	switch mutationType {
	case "insert":
		return buildInsertSQL(entity, mutation)
	case "update":
		return buildUpdateSQL(entity, mutation)
	case "delete":
		return buildDeleteSQL(entity, mutation)
	default:
		return "", fmt.Errorf("unsupported mutation.type '%s': use insert, update or delete", mutation.Type)
	}
}

func buildInsertSQL(entity *engine.Entity, mutation *playgroundSQLMutation) (string, error) {
	if len(mutation.Set) == 0 {
		return "", fmt.Errorf("insert requires non-empty mutation.set")
	}

	keys := sortedMapKeys(mutation.Set)
	columns := make([]string, 0, len(keys))
	values := make([]string, 0, len(keys))

	for _, key := range keys {
		fieldName, ok := resolveFieldName(entity, key)
		if !ok {
			return "", fmt.Errorf("field '%s' not found in entity %s", key, entity.Name)
		}
		columns = append(columns, toSnakeCase(fieldName))
		values = append(values, toSQLLiteral(mutation.Set[key]))
	}

	table := pluralize(toSnakeCase(entity.Name))
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);", table, strings.Join(columns, ", "), strings.Join(values, ", ")), nil
}

func buildUpdateSQL(entity *engine.Entity, mutation *playgroundSQLMutation) (string, error) {
	if len(mutation.Set) == 0 {
		return "", fmt.Errorf("update requires non-empty mutation.set")
	}

	keys := sortedMapKeys(mutation.Set)
	setClauses := make([]string, 0, len(keys))
	for _, key := range keys {
		fieldName, ok := resolveFieldName(entity, key)
		if !ok {
			return "", fmt.Errorf("field '%s' not found in entity %s", key, entity.Name)
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = %s", toSnakeCase(fieldName), toSQLLiteral(mutation.Set[key])))
	}

	whereClauses, err := buildWhereClauses(entity, mutation.Filters)
	if err != nil {
		return "", err
	}

	table := pluralize(toSnakeCase(entity.Name))
	sql := fmt.Sprintf("UPDATE %s SET %s", table, strings.Join(setClauses, ", "))
	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	sql += ";"
	return sql, nil
}

func buildDeleteSQL(entity *engine.Entity, mutation *playgroundSQLMutation) (string, error) {
	whereClauses, err := buildWhereClauses(entity, mutation.Filters)
	if err != nil {
		return "", err
	}

	table := pluralize(toSnakeCase(entity.Name))
	sql := fmt.Sprintf("DELETE FROM %s", table)
	if len(whereClauses) > 0 {
		sql += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	sql += ";"
	return sql, nil
}

func buildWhereClauses(entity *engine.Entity, filters []playgroundSQLMutationFilter) ([]string, error) {
	clauses := make([]string, 0, len(filters))
	for _, filter := range filters {
		fieldName, ok := resolveFieldName(entity, filter.Field)
		if !ok {
			return nil, fmt.Errorf("field '%s' not found in entity %s", filter.Field, entity.Name)
		}

		sqlOp, err := mutationSQLOperator(filter.Op)
		if err != nil {
			return nil, err
		}

		clauses = append(clauses, fmt.Sprintf("%s %s %s", toSnakeCase(fieldName), sqlOp, toSQLLiteral(filter.Value)))
	}

	return clauses, nil
}

func mutationSQLOperator(op string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "eq":
		return "=", nil
	case "neq", "ne":
		return "!=", nil
	case "gt":
		return ">", nil
	case "gte":
		return ">=", nil
	case "lt":
		return "<", nil
	case "lte":
		return "<=", nil
	case "like":
		return "LIKE", nil
	default:
		return "", fmt.Errorf("unsupported filter operator '%s' in mutation", op)
	}
}

func findSchemaEntity(schema *engine.Schema, entityName string) *engine.Entity {
	if schema == nil {
		return nil
	}
	for _, entity := range schema.Entities {
		if strings.EqualFold(entity.Name, entityName) {
			return entity
		}
	}
	return nil
}

func resolveFieldName(entity *engine.Entity, field string) (string, bool) {
	if entity == nil {
		return "", false
	}
	trimmed := strings.TrimSpace(field)
	if trimmed == "" {
		return "", false
	}

	if _, ok := entity.Fields[trimmed]; ok {
		return trimmed, true
	}

	for name := range entity.Fields {
		if strings.EqualFold(name, trimmed) || strings.EqualFold(toSnakeCase(name), toSnakeCase(trimmed)) {
			return name, true
		}
	}

	return "", false
}

func sortedMapKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func toSQLLiteral(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return "NULL"
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	case int:
		return strconv.Itoa(v)
	case int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case float64:
		if float64(int64(v)) == v {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	default:
		return "'" + strings.ReplaceAll(fmt.Sprintf("%v", v), "'", "''") + "'"
	}
}

func toSnakeCase(input string) string {
	if input == "" {
		return input
	}

	var b strings.Builder
	for i, r := range input {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}

	return strings.ToLower(b.String())
}

func pluralize(name string) string {
	if strings.HasSuffix(name, "s") {
		return name
	}
	return name + "s"
}
