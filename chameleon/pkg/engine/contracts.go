package engine

import "context"

// ============================================================
// MUTATION TYPES
// ============================================================

type MutationType int

const (
	MutationInsert MutationType = iota
	MutationUpdate
	MutationDelete
)

type Mutation struct {
	Type         MutationType
	Entity       string
	HasFilter    bool
	AffectedRows int64
}

// ============================================================
// MUTATION RESULT TYPES
// ============================================================

type InsertResult struct {
	ID       interface{}            // Primary key
	Record   map[string]interface{} // Full record (if RETURNING)
	Affected int
}

type UpdateResult struct {
	Records  []map[string]interface{}
	Affected int
}

type DeleteResult struct {
	Affected int
}

// ============================================================
// MUTATION BUILDER INTERFACES
// ============================================================

// InsertMutation builds and executes INSERT operations
type InsertMutation interface {
	// Set adds a field to insert
	Set(field string, value interface{}) InsertMutation

	// Debug enables debug output for this mutation
	Debug() InsertMutation

	// Execute validates and runs the mutation
	Execute(ctx context.Context) (*InsertResult, error)
}

// UpdateMutation builds and executes UPDATE operations
type UpdateMutation interface {
	// Set adds a field to update
	Set(field string, value interface{}) UpdateMutation

	// Filter adds a filter condition (WHERE clause)
	Filter(field string, operator string, value interface{}) UpdateMutation

	// Debug enables debug output for this mutation
	Debug() UpdateMutation

	// Execute validates and runs the mutation
	Execute(ctx context.Context) (*UpdateResult, error)
}

// DeleteMutation builds and executes DELETE operations
type DeleteMutation interface {
	// Filter adds a filter condition (WHERE clause)
	Filter(field string, operator string, value interface{}) DeleteMutation

	// Debug enables debug output for this mutation
	Debug() DeleteMutation

	// Execute validates and runs the mutation
	Execute(ctx context.Context) (*DeleteResult, error)
}

// ============================================================
// FACTORY
// ============================================================

// MutationFactory creates mutation builders
//
// CRITICAL: Factory is STATELESS.
// Schema and Connector are passed in each call to allow registry pattern.
// This avoids import cycles (engine <-> mutation).
//
// Factory is registered once via init() in mutation package.
// Engine uses it via getMutationFactory() from registry.
type MutationFactory interface {
	// NewInsert creates a builder for INSERT operations
	// Schema and Connector are passed in to keep factory stateless
	NewInsert(entity string, schema *Schema, connector *Connector) InsertMutation

	// NewUpdate creates a builder for UPDATE operations
	NewUpdate(entity string, schema *Schema, connector *Connector) UpdateMutation

	// NewDelete creates a builder for DELETE operations
	NewDelete(entity string, schema *Schema, connector *Connector) DeleteMutation
}

// ============================================================
// AUXILIARY CONTRACTS
// ============================================================

type ExecutionResult struct {
	RowsAffected int64
	LastInsertID interface{}
	Rows         []map[string]interface{}
}
