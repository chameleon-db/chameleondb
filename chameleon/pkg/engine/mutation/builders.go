package mutation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// ============================================================
// INSERT BUILDER
// ============================================================

type InsertBuilder struct {
	schema *engine.Schema
	entity string
	values map[string]interface{}
	config engine.ValidatorConfig

	// Debug settings (inherited from engine or per-mutation)
	debugLevel *engine.DebugLevel
	debugCtx   *engine.DebugContext
}

func NewInsertBuilder(schema *engine.Schema, entity string) *InsertBuilder {
	return &InsertBuilder{
		schema: schema,
		entity: entity,
		values: make(map[string]interface{}),
		config: engine.DefaultValidatorConfig(),
	}
}

// Set implements engine.InsertMutation
func (ib *InsertBuilder) Set(field string, value interface{}) engine.InsertMutation {
	ib.values[field] = value
	return ib
}

// Debug implements engine.InsertMutation
func (ib *InsertBuilder) Debug() engine.InsertMutation {
	level := engine.DebugSQL
	ib.debugLevel = &level
	return ib
}

// Execute implements engine.InsertMutation
func (ib *InsertBuilder) Execute(ctx context.Context) (*engine.InsertResult, error) {
	start := time.Now()

	// Validate
	validator := engine.NewValidator(ib.schema, ib.config)
	if err := validator.ValidateInsertInput(ib.entity, ib.values); err != nil {
		return nil, err
	}

	// Generate SQL
	sql := ib.generateSQL()

	// Debug output
	if ib.shouldDebug() {
		ib.logSQL(sql)
	}

	// TODO: Execute via connector/executor
	// For now, return success without actual execution

	duration := time.Since(start)

	// Debug trace
	if ib.shouldTrace() {
		ib.logTrace("INSERT", duration, 1)
	}

	return &engine.InsertResult{
		ID:       nil, // Will be filled by actual executor
		Record:   nil, // Will be filled by actual executor
		Affected: 1,
	}, nil
}

func (ib *InsertBuilder) shouldDebug() bool {
	if ib.debugLevel != nil {
		return *ib.debugLevel >= engine.DebugSQL
	}
	return false
}

func (ib *InsertBuilder) shouldTrace() bool {
	if ib.debugLevel != nil {
		return *ib.debugLevel >= engine.DebugTrace
	}
	return false
}

func (ib *InsertBuilder) logSQL(sql string) {
	fmt.Printf("\n[SQL] INSERT INTO %s\n%s\n\n", ib.entity, sql)
}

func (ib *InsertBuilder) logTrace(operation string, duration time.Duration, affected int) {
	fmt.Printf("[TRACE] %s on %s: %v, %d rows\n", operation, ib.entity, duration, affected)
}

func (ib *InsertBuilder) generateSQL() string {
	var fields []string
	var placeholders []string
	i := 1

	for field := range ib.values {
		fields = append(fields, field)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		i++
	}

	table := strings.ToLower(ib.entity) + "s" // Simple pluralization

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		table,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)
}

// ============================================================
// UPDATE BUILDER
// ============================================================

type UpdateBuilder struct {
	schema  *engine.Schema
	entity  string
	filters map[string]interface{}
	updates map[string]interface{}
	config  engine.ValidatorConfig

	// Debug settings
	debugLevel *engine.DebugLevel
}

func NewUpdateBuilder(schema *engine.Schema, entity string) *UpdateBuilder {
	return &UpdateBuilder{
		schema:  schema,
		entity:  entity,
		filters: make(map[string]interface{}),
		updates: make(map[string]interface{}),
		config:  engine.DefaultValidatorConfig(),
	}
}

// Filter implements engine.UpdateMutation
func (ub *UpdateBuilder) Filter(field string, op string, value interface{}) engine.UpdateMutation {
	key := fmt.Sprintf("%s:%s", field, op)
	ub.filters[key] = value
	return ub
}

// Set implements engine.UpdateMutation
func (ub *UpdateBuilder) Set(field string, value interface{}) engine.UpdateMutation {
	ub.updates[field] = value
	return ub
}

// Debug implements engine.UpdateMutation
func (ub *UpdateBuilder) Debug() engine.UpdateMutation {
	level := engine.DebugSQL
	ub.debugLevel = &level
	return ub
}

// Execute implements engine.UpdateMutation
func (ub *UpdateBuilder) Execute(ctx context.Context) (*engine.UpdateResult, error) {
	start := time.Now()

	// Validate
	validator := engine.NewValidator(ub.schema, ub.config)
	if err := validator.ValidateUpdateInput(
		ub.entity,
		ub.parseFilters(),
		ub.updates,
	); err != nil {
		return nil, err
	}

	// Generate SQL
	sql := ub.generateSQL()

	// Debug output
	if ub.shouldDebug() {
		fmt.Printf("\n[SQL] UPDATE %s\n%s\n\n", ub.entity, sql)
	}

	// TODO: Execute via connector/executor

	duration := time.Since(start)

	if ub.shouldTrace() {
		fmt.Printf("[TRACE] UPDATE on %s: %v, %d rows\n", ub.entity, duration, 0)
	}

	return &engine.UpdateResult{
		Records:  nil,
		Affected: 0,
	}, nil
}

func (ub *UpdateBuilder) shouldDebug() bool {
	if ub.debugLevel != nil {
		return *ub.debugLevel >= engine.DebugSQL
	}
	return false
}

func (ub *UpdateBuilder) shouldTrace() bool {
	if ub.debugLevel != nil {
		return *ub.debugLevel >= engine.DebugTrace
	}
	return false
}

func (ub *UpdateBuilder) generateSQL() string {
	var setClauses []string
	i := 1

	for field := range ub.updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, i))
		i++
	}

	var filterClauses []string
	for field := range ub.filters {
		fieldName := strings.Split(field, ":")[0]
		filterClauses = append(filterClauses, fmt.Sprintf("%s = $%d", fieldName, i))
		i++
	}

	table := strings.ToLower(ub.entity) + "s"

	return fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s RETURNING *",
		table,
		strings.Join(setClauses, ", "),
		strings.Join(filterClauses, " AND "),
	)
}

func (ub *UpdateBuilder) parseFilters() map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range ub.filters {
		parts := strings.Split(key, ":")
		if len(parts) > 0 {
			result[parts[0]] = value
		}
	}
	return result
}

// ============================================================
// DELETE BUILDER
// ============================================================

type DeleteBuilder struct {
	schema         *engine.Schema
	entity         string
	filters        map[string]interface{}
	config         engine.ValidatorConfig
	forceDeleteAll bool

	// Debug settings
	debugLevel *engine.DebugLevel
}

func NewDeleteBuilder(schema *engine.Schema, entity string) *DeleteBuilder {
	return &DeleteBuilder{
		schema:  schema,
		entity:  entity,
		filters: make(map[string]interface{}),
		config:  engine.DefaultValidatorConfig(),
	}
}

// Filter implements engine.DeleteMutation
func (db *DeleteBuilder) Filter(field string, op string, value interface{}) engine.DeleteMutation {
	key := fmt.Sprintf("%s:%s", field, op)
	db.filters[key] = value
	return db
}

// Debug implements engine.DeleteMutation
func (db *DeleteBuilder) Debug() engine.DeleteMutation {
	level := engine.DebugSQL
	db.debugLevel = &level
	return db
}

// Execute implements engine.DeleteMutation
func (db *DeleteBuilder) Execute(ctx context.Context) (*engine.DeleteResult, error) {
	start := time.Now()

	// Validate
	validator := engine.NewValidator(db.schema, db.config)
	if err := validator.ValidateDeleteInput(
		db.entity,
		db.parseFilters(),
		db.forceDeleteAll,
	); err != nil {
		return nil, err
	}

	// Generate SQL
	sql := db.generateSQL()

	// Debug output
	if db.shouldDebug() {
		fmt.Printf("\n[SQL] DELETE FROM %s\n%s\n\n", db.entity, sql)
	}

	// TODO: Execute via connector/executor

	duration := time.Since(start)

	if db.shouldTrace() {
		fmt.Printf("[TRACE] DELETE on %s: %v, %d rows\n", db.entity, duration, 0)
	}

	return &engine.DeleteResult{
		Affected: 0,
	}, nil
}

func (db *DeleteBuilder) shouldDebug() bool {
	if db.debugLevel != nil {
		return *db.debugLevel >= engine.DebugSQL
	}
	return false
}

func (db *DeleteBuilder) shouldTrace() bool {
	if db.debugLevel != nil {
		return *db.debugLevel >= engine.DebugTrace
	}
	return false
}

func (db *DeleteBuilder) generateSQL() string {
	var filterClauses []string
	i := 1

	for field := range db.filters {
		fieldName := strings.Split(field, ":")[0]
		filterClauses = append(filterClauses, fmt.Sprintf("%s = $%d", fieldName, i))
		i++
	}

	table := strings.ToLower(db.entity) + "s"

	if len(filterClauses) == 0 {
		return fmt.Sprintf("DELETE FROM %s", table)
	}

	return fmt.Sprintf(
		"DELETE FROM %s WHERE %s",
		table,
		strings.Join(filterClauses, " AND "),
	)
}

func (db *DeleteBuilder) parseFilters() map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range db.filters {
		parts := strings.Split(key, ":")
		if len(parts) > 0 {
			result[parts[0]] = value
		}
	}
	return result
}
