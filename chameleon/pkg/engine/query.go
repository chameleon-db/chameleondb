package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chameleon-db/chameleondb/chameleon/internal/ffi"
)

// --- Query types (mirror Rust Query AST) ---

/*
	 type FilterValue struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	}
*/
type FilterValue map[string]interface{}

type FilterCondition struct {
	Field FieldPath   `json:"field"`
	Op    string      `json:"op"` // "Eq", "Neq", "Gt", etc.
	Value FilterValue `json:"value"`
}

type FieldPath struct {
	Segments []string `json:"segments"`
}

type FilterExpr struct {
	Condition *FilterCondition `json:"Condition,omitempty"`
	Binary    *BinaryExpr      `json:"Binary,omitempty"`
}

type BinaryExpr struct {
	Left  FilterExpr `json:"left"`
	Op    string     `json:"op"` // "And", "Or"
	Right FilterExpr `json:"right"`
}

type IncludePath struct {
	Path []string `json:"path"`
}

type OrderByClause struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // "Asc", "Desc"
}

// QueryJSON is the serialization format matching Rust's Query
type QueryJSON struct {
	Entity       string          `json:"entity"`
	Filters      []FilterExpr    `json:"filters"`
	Includes     []IncludePath   `json:"includes"`
	OrderBy      []OrderByClause `json:"order_by"`
	Limit        *uint64         `json:"limit,omitempty"`
	Offset       *uint64         `json:"offset,omitempty"`
	SelectFields []string        `json:"select_fields"`
}

// GeneratedSQL mirrors Rust's GeneratedSQL.
//
// The SQL text only contains placeholders ($1, $2, ...). Values are carried
// separately in MainParams and, for each eager query, in the matching entry
// of EagerParams.
type GeneratedSQL struct {
	MainQuery    string       `json:"main_query"`
	EagerQueries [][]string   `json:"eager_queries"`
	MainParams   []SQLParam   `json:"main_params,omitempty"`
	EagerParams  [][]SQLParam `json:"eager_params,omitempty"`
}

type EagerQuery struct {
	Relation string     `json:"0"`
	SQL      string     `json:"1"`
	Params   []SQLParam `json:"params,omitempty"`
}

// SQL parameter kinds produced by the generator.
const (
	ParamString    = "String"
	ParamInt       = "Int"
	ParamFloat     = "Float"
	ParamBool      = "Bool"
	ParamNull      = "Null"
	ParamParentIDs = "ParentIds"
)

// SQLParam is a value bound to a placeholder of a generated statement.
// ParentIds is resolved by the executor from the rows of the parent query.
type SQLParam struct {
	Kind  string
	Value interface{} // string, int64, float64 or bool; nil for Null and ParentIds
}

// NewSQLParam converts a Go value into a parameter.
func NewSQLParam(value interface{}) SQLParam {
	switch v := value.(type) {
	case nil:
		return SQLParam{Kind: ParamNull}
	case bool:
		return SQLParam{Kind: ParamBool, Value: v}
	case int:
		return SQLParam{Kind: ParamInt, Value: int64(v)}
	case int64:
		return SQLParam{Kind: ParamInt, Value: v}
	case float64:
		return SQLParam{Kind: ParamFloat, Value: v}
	case string:
		return SQLParam{Kind: ParamString, Value: v}
	default:
		return SQLParam{Kind: ParamString, Value: fmt.Sprintf("%v", v)}
	}
}

// UnmarshalJSON reads the externally tagged form used by the Rust core:
// {"String":"x"}, {"Int":1}, {"Float":1.5}, {"Bool":true}, "Null", "ParentIds".
func (p *SQLParam) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		if name != ParamNull && name != ParamParentIDs {
			return fmt.Errorf("unknown parameter kind %q", name)
		}
		*p = SQLParam{Kind: name}
		return nil
	}

	var tagged map[string]json.RawMessage
	if err := json.Unmarshal(data, &tagged); err != nil {
		return fmt.Errorf("invalid parameter: %w", err)
	}
	if len(tagged) != 1 {
		return fmt.Errorf("invalid parameter: expected a single kind")
	}

	for kind, raw := range tagged {
		var target interface{}
		switch kind {
		case ParamString:
			var v string
			target = &v
		case ParamInt:
			var v int64
			target = &v
		case ParamFloat:
			var v float64
			target = &v
		case ParamBool:
			var v bool
			target = &v
		case ParamNull:
			*p = SQLParam{Kind: ParamNull}
			return nil
		default:
			return fmt.Errorf("unknown parameter kind %q", kind)
		}
		if err := json.Unmarshal(raw, target); err != nil {
			return fmt.Errorf("invalid %s parameter: %w", kind, err)
		}
		switch v := target.(type) {
		case *string:
			*p = SQLParam{Kind: kind, Value: *v}
		case *int64:
			*p = SQLParam{Kind: kind, Value: *v}
		case *float64:
			*p = SQLParam{Kind: kind, Value: *v}
		case *bool:
			*p = SQLParam{Kind: kind, Value: *v}
		}
	}
	return nil
}

// MarshalJSON writes the same form UnmarshalJSON reads.
func (p SQLParam) MarshalJSON() ([]byte, error) {
	if p.Kind == ParamNull || p.Kind == ParamParentIDs {
		return json.Marshal(p.Kind)
	}
	return json.Marshal(map[string]interface{}{p.Kind: p.Value})
}

// String renders a parameter for display next to placeholder SQL.
func (p SQLParam) String() string {
	switch p.Kind {
	case ParamString:
		return fmt.Sprintf("String(%q)", p.Value)
	case ParamInt, ParamFloat, ParamBool:
		return fmt.Sprintf("%s(%v)", p.Kind, p.Value)
	default:
		return p.Kind
	}
}

// --- Query Builder ---

// QueryBuilder provides a chainable API for building queries
type QueryBuilder struct {
	engine     *Engine
	query      QueryJSON
	entityName string

	// debugLevel overrides the engine debug level for this query.
	debugLevel *DebugLevel
}

// Query starts a new query for the given entity
func (e *Engine) Query(entity string) *QueryBuilder {
	return &QueryBuilder{
		engine: e,
		query: QueryJSON{
			Entity:       entity,
			Filters:      []FilterExpr{},
			Includes:     []IncludePath{},
			OrderBy:      []OrderByClause{},
			SelectFields: []string{},
		},
	}
}

// Filter adds a filter condition
// field: "email" or "orders.total" (supports relation navigation)
// op: "eq", "neq", "gt", "gte", "lt", "lte", "like"
// value: string, int, float, or bool
func (qb *QueryBuilder) Filter(field string, op string, value interface{}) *QueryBuilder {
	rustOp := goOpToRust(op)

	qb.query.Filters = append(qb.query.Filters, FilterExpr{
		Condition: &FilterCondition{
			Field: parseFieldPath(field),
			Op:    rustOp,
			Value: goValueToFilter(value),
		},
	})
	return qb
}

// Include adds eager loading for a relation
// Supports nested paths: "orders", "orders.items"
func (qb *QueryBuilder) Include(path string) *QueryBuilder {
	qb.query.Includes = append(qb.query.Includes, IncludePath{
		Path: splitPath(path),
	})
	return qb
}

// OrderBy adds a sort clause
// direction: "asc" or "desc"
func (qb *QueryBuilder) OrderBy(field string, direction string) *QueryBuilder {
	dir := "Asc"
	if direction == "desc" {
		dir = "Desc"
	}
	qb.query.OrderBy = append(qb.query.OrderBy, OrderByClause{
		Field:     field,
		Direction: dir,
	})
	return qb
}

// Limit sets the maximum number of results
func (qb *QueryBuilder) Limit(n uint64) *QueryBuilder {
	qb.query.Limit = &n
	return qb
}

// Offset sets the number of results to skip
func (qb *QueryBuilder) Offset(n uint64) *QueryBuilder {
	qb.query.Offset = &n
	return qb
}

// ToSQL generates SQL without executing.
func (qb *QueryBuilder) ToSQL() (*GeneratedSQL, error) {
	if qb.engine.schema == nil {
		return nil, fmt.Errorf("no schema loaded")
	}

	queryJSON, err := json.Marshal(qb.query)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize query: %w", err)
	}

	schemaJSON, err := json.Marshal(qb.engine.schema)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize schema: %w", err)
	}

	resultJSON, err := ffi.GenerateSQL(string(queryJSON), string(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("SQL generation failed: %w", err)
	}

	var result GeneratedSQL
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse generated SQL: %w", err)
	}

	return &result, nil
}

// Execute generates SQL and runs it against the database
func (qb *QueryBuilder) Execute(ctx context.Context) (*QueryResult, error) {
	if qb.engine.executor == nil {
		return nil, fmt.Errorf("executor not initialized - call engine.Connect() first")
	}

	start := time.Now()

	generated, err := qb.ToSQL()
	if err != nil {
		return nil, err
	}

	debugCtx := qb.getDebugContext()
	debugCtx.LogSQL(generated.MainQuery)
	debugCtx.LogParamsForSQL(generated.MainQuery, generated.MainParams)

	result, err := qb.engine.executor.Execute(ctx, qb, generated)
	if err != nil {
		return nil, err
	}

	duration := time.Since(start)
	debugCtx.LogQuery(generated.MainQuery, duration, len(result.Rows))

	return result, nil
}

// Select specifies which fields to retrieve
// If not called, defaults to SELECT * (all fields)
//
// Examples:
//
//	db.Query("User").Select("id", "name").Execute(ctx)
//	db.Query("User").Select("id", "email", "created_at").Execute(ctx)
func (qb *QueryBuilder) Select(fields ...string) *QueryBuilder {
	qb.query.SelectFields = fields
	return qb
}

// Debug enables debug mode for this query
func (qb *QueryBuilder) Debug() *QueryBuilder {
	level := DebugSQL
	qb.debugLevel = &level
	return qb
}

// DebugTrace enables full trace for this query
func (qb *QueryBuilder) DebugTrace() *QueryBuilder {
	level := DebugTrace
	qb.debugLevel = &level
	return qb
}

func (qb *QueryBuilder) getDebugContext() *DebugContext {
	base := qb.engine.Debug
	if base == nil {
		base = DefaultDebugContext()
	}

	if qb.debugLevel != nil {
		return &DebugContext{
			Level:       *qb.debugLevel,
			Writer:      base.Writer,
			ColorOutput: base.ColorOutput,
		}
	}
	return base
}

// --- Helpers ---
func parseFieldPath(path string) FieldPath {
	return FieldPath{Segments: splitPath(path)}
}

func splitPath(path string) []string {
	segments := []string{}
	current := ""
	for _, ch := range path {
		if ch == '.' {
			if current != "" {
				segments = append(segments, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		segments = append(segments, current)
	}
	return segments
}

func goOpToRust(op string) string {
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
	if rustOp, ok := ops[op]; ok {
		return rustOp
	}
	return op
}

func goValueToFilter(value interface{}) FilterValue {
	switch v := value.(type) {
	case string:
		return FilterValue{"String": v}
	case int:
		return FilterValue{"Int": v}
	case int64:
		return FilterValue{"Int": v}
	case float64:
		return FilterValue{"Float": v}
	case bool:
		return FilterValue{"Bool": v}
	case nil:
		return FilterValue{"Null": nil}
	default:
		return FilterValue{"String": fmt.Sprintf("%v", v)}
	}
}
