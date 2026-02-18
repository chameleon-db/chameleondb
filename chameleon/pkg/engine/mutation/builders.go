package mutation

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// ============================================================
// INSERT BUILDER
// ============================================================

type InsertBuilder struct {
	schema    *engine.Schema
	connector *engine.Connector
	entity    string
	values    map[string]interface{}
	config    engine.ValidatorConfig

	// Debug settings
	debugLevel *engine.DebugLevel
}

func NewInsertBuilder(schema *engine.Schema, connector *engine.Connector, entity string) *InsertBuilder {
	return &InsertBuilder{
		schema:    schema,
		connector: connector,
		entity:    entity,
		values:    make(map[string]interface{}),
		config:    engine.DefaultValidatorConfig(),
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
	sql, orderedValues := ib.generateSQL()

	// Debug output
	if ib.shouldDebug() {
		fmt.Printf("[ENTITY] INSERT INTO %s\n", ib.entity)
		fmt.Printf("[SQL] %s\n", sql)
		fmt.Printf("[VALUES] %v\n\n", orderedValues)
	}

	// Execute via pgx
	rows, err := ib.connector.Pool().Query(ctx, sql, orderedValues...)
	if err != nil {
		return nil, mapDatabaseError(err, ib.entity, "INSERT", ib.values)
	}
	defer rows.Close()

	// Parse RETURNING *
	if !rows.Next() {
		// Check for errors first
		if err := rows.Err(); err != nil {
			// ← AQUÍ está el error de UNIQUE violation
			return nil, mapDatabaseError(err, ib.entity, "INSERT", ib.values)
		}
		// No rows and no error = problem
		return nil, fmt.Errorf("INSERT executed but returned no rows (check required fields)")
	}

	values, err := rows.Values()
	if err != nil {
		return nil, fmt.Errorf("failed to scan result: %w", err)
	}

	// Build record map
	record := make(map[string]interface{})
	columns := rows.FieldDescriptions()
	for i, col := range columns {
		record[col.Name] = values[i]
	}

	// Extract ID
	var id interface{}
	if len(values) > 0 {
		id = values[0]
		for i, col := range columns {
			if col.Name == "id" {
				id = values[i]
				break
			}
		}
	}

	result := &engine.InsertResult{
		ID:       id,
		Record:   record,
		Affected: 1,
	}

	duration := time.Since(start)

	if ib.shouldTrace() {
		fmt.Printf("[TRACE] INSERT on %s: %v, 1 row\n", ib.entity, duration)
	}

	return result, nil
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

func (ib *InsertBuilder) generateSQL() (string, []interface{}) {
	// Get entity definition for table name
	ent := ib.schema.GetEntity(ib.entity)
	if ent == nil {
		// Fallback: simple lowercase + "s"
		return ib.generateSQLFallback()
	}

	// Use entity table name (handles pluralization correctly)
	tableName := entityToTableName(ib.entity)

	var fields []string
	var placeholders []string
	var values []interface{}

	// Get field names and sort them for consistent order
	for field := range ib.values {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	// Build placeholders and values in sorted order
	for i, field := range fields {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		values = append(values, ib.values[field])
	}

	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	return sql, values
}

func (ib *InsertBuilder) generateSQLFallback() (string, []interface{}) {
	tableName := strings.ToLower(ib.entity) + "s"

	var fields []string
	var placeholders []string
	var values []interface{}

	// Get field names and sort them for consistent order
	for field := range ib.values {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	// Build placeholders and values in sorted order
	for i, field := range fields {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		values = append(values, ib.values[field])
	}

	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	return sql, values
}

// ============================================================
// UPDATE BUILDER
// ============================================================

type UpdateBuilder struct {
	schema    *engine.Schema
	connector *engine.Connector
	entity    string
	filters   map[string]interface{}
	updates   map[string]interface{}
	config    engine.ValidatorConfig

	// Debug settings
	debugLevel *engine.DebugLevel
	forceAll   bool
}

func NewUpdateBuilder(schema *engine.Schema, connector *engine.Connector, entity string) *UpdateBuilder {
	return &UpdateBuilder{
		schema:    schema,
		connector: connector,
		entity:    entity,
		filters:   make(map[string]interface{}),
		updates:   make(map[string]interface{}),
		config:    engine.DefaultValidatorConfig(),
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
	sql, orderedValues := ub.generateSQL()

	// Debug output
	if ub.shouldDebug() {
		fmt.Printf("\n[SQL] UPDATE %s\n%s\n", ub.entity, sql)
		fmt.Printf("[VALUES] %v\n\n", orderedValues)
	}

	// Execute via pgx
	rows, err := ub.connector.Pool().Query(ctx, sql, orderedValues...)
	if err != nil {
		return nil, mapDatabaseError(err, ub.entity, "UPDATE", ub.updates)
	}
	defer rows.Close()

	// Parse RETURNING * (all updated rows)
	var records []map[string]interface{}
	columns := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		record := make(map[string]interface{})
		for i, col := range columns {
			record[col.Name] = values[i]
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err, ub.entity, "UPDATE", ub.updates)
	}

	duration := time.Since(start)

	if ub.shouldTrace() {
		fmt.Printf("[TRACE] UPDATE on %s: %v, %d rows\n", ub.entity, duration, len(records))
	}

	return &engine.UpdateResult{
		Records:  records,
		Affected: len(records),
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

func (ub *UpdateBuilder) generateSQL() (string, []interface{}) {
	tableName := entityToTableName(ub.entity)

	var setClauses []string
	var values []interface{}
	paramIndex := 1

	// SET clauses - sort fields for consistent order
	var updateFields []string
	for field := range ub.updates {
		updateFields = append(updateFields, field)
	}
	sort.Strings(updateFields)

	for _, field := range updateFields {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, paramIndex))
		values = append(values, ub.updates[field])
		paramIndex++
	}

	// WHERE clauses - sort filters for consistent order
	var whereFields []string
	for filterKey := range ub.filters {
		whereFields = append(whereFields, filterKey)
	}
	sort.Strings(whereFields)

	var whereClauses []string
	for _, filterKey := range whereFields {
		parts := strings.Split(filterKey, ":")
		field := parts[0]
		// For now, only support "eq" operator
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", field, paramIndex))
		values = append(values, ub.filters[filterKey])
		paramIndex++
	}

	sql := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s RETURNING *",
		tableName,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
	)

	return sql, values
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
	connector      *engine.Connector
	entity         string
	filters        map[string]interface{}
	config         engine.ValidatorConfig
	forceDeleteAll bool

	// Debug settings
	debugLevel *engine.DebugLevel
}

func NewDeleteBuilder(schema *engine.Schema, connector *engine.Connector, entity string) *DeleteBuilder {
	return &DeleteBuilder{
		schema:    schema,
		connector: connector,
		entity:    entity,
		filters:   make(map[string]interface{}),
		config:    engine.DefaultValidatorConfig(),
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
	sql, orderedValues := db.generateSQL()

	// Debug output
	if db.shouldDebug() {
		fmt.Printf("\n[SQL] DELETE FROM %s\n%s\n", db.entity, sql)
		fmt.Printf("[VALUES] %v\n\n", orderedValues)
	}

	// Execute via pgx
	commandTag, err := db.connector.Pool().Exec(ctx, sql, orderedValues...)
	if err != nil {
		// ← CAMBIO: Map database error to ChameleonDB error
		return nil, mapDatabaseError(err, db.entity, "DELETE", nil)
	}

	affected := int(commandTag.RowsAffected())

	duration := time.Since(start)

	// Debug trace
	if db.shouldTrace() {
		fmt.Printf("[TRACE] DELETE on %s: %v, %d rows\n", db.entity, duration, affected)
	}

	return &engine.DeleteResult{
		Affected: affected,
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

func (db *DeleteBuilder) generateSQL() (string, []interface{}) {
	tableName := entityToTableName(db.entity)

	var whereClauses []string
	var values []interface{}
	paramIndex := 1

	for filterKey, value := range db.filters {
		parts := strings.Split(filterKey, ":")
		field := parts[0]
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", field, paramIndex))
		values = append(values, value)
		paramIndex++
	}

	sql := fmt.Sprintf(
		"DELETE FROM %s WHERE %s",
		tableName,
		strings.Join(whereClauses, " AND "),
	)

	return sql, values
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

// ============================================================
// UTILITIES
// ============================================================

// entityToTableName converts entity name to table name
// Handles pluralization and snake_case conversion
//
// Examples:
//
//	User → users
//	OrderItem → order_items
//	TodoList → todo_lists
func entityToTableName(entity string) string {
	// TODO: Use FFI call to Rust for proper naming
	// For now, simple implementation:

	// Convert PascalCase to snake_case
	var result []rune
	for i, r := range entity {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}

	name := strings.ToLower(string(result))

	// Check for irregular plural
	if plural, ok := irregularPlurals[name]; ok {
		return plural
	}

	// Simple pluralization
	if !strings.HasSuffix(name, "s") {
		name += "s"
	}

	return name
}
