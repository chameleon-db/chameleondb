package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"unsafe"

	"github.com/chameleon-db/chameleondb/chameleon/internal/ffi"
)

// Engine is the main entry point for ChameleonDB
type Engine struct {
	schema    *Schema
	connector *Connector
	executor  *Executor
	ffiHandle unsafe.Pointer

	// Debug context
	Debug *DebugContext
}

func (e *Engine) Schema() *Schema {
	return e.schema
}

// ============================================================
// ENGINE INITIALIZATION
// ============================================================

// NewEngine creates and initializes a new ChameleonDB engine
//
// Default behavior:
//   - Loads schema from "schema.cham" if it exists
//   - Ready to use immediately
//
// If "schema.cham" doesn't exist, returns engine without schema
// (user must call LoadSchemaFromFile manually)
func NewEngine() *Engine {
	return newEngineWithPath("schema.cham")
}

// NewEngineWithSchema creates and initializes engine with a specific schema file
func NewEngineWithSchema(schemaPath string) (*Engine, error) {
	eng := newEngineWithPath(schemaPath)

	// If file doesn't exist, return error explicitly
	if _, err := os.Stat(schemaPath); err != nil {
		return nil, fmt.Errorf(
			"schema file not found: %s\n"+
				"Create a schema.cham file or use engine.NewEngineWithoutSchema()",
			schemaPath,
		)
	}

	return eng, nil
}

// NewEngineWithoutSchema creates an engine without loading a schema
func NewEngineWithoutSchema() *Engine {
	return &Engine{
		Debug: DefaultDebugContext(),
	}
}

// newEngineWithPath is the internal helper
func newEngineWithPath(schemaPath string) *Engine {
	eng := &Engine{
		Debug: DefaultDebugContext(),
	}

	// Try to load schema silently (don't fail if missing)
	if _, err := os.Stat(schemaPath); err == nil {
		eng.LoadSchemaFromFile(schemaPath)
	}

	return eng
}

// WithDebug returns a new engine with debug enabled
func (e *Engine) WithDebug(level DebugLevel) *Engine {
	e.Debug = &DebugContext{
		Level:       level,
		Writer:      os.Stdout,
		ColorOutput: true,
	}
	return e
}

// ─────────────────────────────────────────────────────────────
// Schema handling
// ─────────────────────────────────────────────────────────────

// LoadSchemaFromString parses a schema from a string
func (e *Engine) LoadSchemaFromString(input string) (*Schema, error) {
	schemaJSON, err := ffi.ParseSchema(input)
	if err != nil {
		formattedErr := FormatError(err.Error())
		return nil, fmt.Errorf("%s", formattedErr)
	}

	var schema Schema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return nil, fmt.Errorf("failed to deserialize schema: %w", err)
	}

	e.schema = &schema
	return &schema, nil
}

// LoadSchemaFromFile loads a schema from a .cham file
func (e *Engine) LoadSchemaFromFile(filepath string) (*Schema, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	return e.LoadSchemaFromString(string(content))
}

// GetSchema returns the currently loaded schema
func (e *Engine) GetSchema() *Schema {
	return e.schema
}

// ─────────────────────────────────────────────────────────────
// Connection handling
// ─────────────────────────────────────────────────────────────

// Version returns the engine version
func (e *Engine) Version() string {
	return ffi.Version()
}

// Connect establishes a database connection
func (e *Engine) Connect(ctx context.Context, config ConnectorConfig) error {
	e.connector = NewConnector(config)
	if err := e.connector.Connect(ctx); err != nil {
		return err
	}
	e.executor = NewExecutor(e.connector)

	return nil
}

// Close closes the database connection
func (e *Engine) Close() {
	if e.connector != nil {
		e.connector.Close()
	}
}

// IsConnected returns true if connected to a database
func (e *Engine) IsConnected() bool {
	return e.connector != nil && e.connector.IsConnected()
}

// Ping verifies the database connection is alive
func (e *Engine) Ping(ctx context.Context) error {
	if e.connector == nil {
		return fmt.Errorf("not connected")
	}
	return e.connector.Ping(ctx)
}

// Connector returns the underlying connector for raw SQL access
func (e *Engine) Connector() *Connector {
	return e.connector
}

// ─────────────────────────────────────────────────────────────
// Migrations
// ─────────────────────────────────────────────────────────────

// GenerateMigration generates DDL SQL from the loaded schema
func (e *Engine) GenerateMigration() (string, error) {
	if e.schema == nil {
		return "", fmt.Errorf("no schema loaded")
	}

	schemaJSON, err := json.Marshal(e.schema)
	if err != nil {
		return "", fmt.Errorf("failed to serialize schema: %w", err)
	}

	return ffi.GenerateMigration(string(schemaJSON))
}

// ─────────────────────────────────────────────────────────────
// Mutation API (uses registry pattern)
// ─────────────────────────────────────────────────────────────

// Insert starts a new INSERT mutation
func (e *Engine) Insert(entity string) InsertMutation {
	if e.schema == nil {
		return newInvalidInsertMutation(fmt.Errorf("schema not loaded - call LoadSchemaFromFile first"))
	}
	if e.connector == nil {
		return newInvalidInsertMutation(fmt.Errorf("not connected - call Connect() first"))
	}

	factory := getMutationFactory()
	if factory == nil {
		return newInvalidInsertMutation(fmt.Errorf("no mutation factory registered"))
	}
	return factory.NewInsert(entity, e.schema, e.connector)
}

// Update starts a new UPDATE mutation
func (e *Engine) Update(entity string) UpdateMutation {
	if e.schema == nil {
		return newInvalidUpdateMutation(fmt.Errorf("schema not loaded - call LoadSchemaFromFile first"))
	}
	if e.connector == nil {
		return newInvalidUpdateMutation(fmt.Errorf("not connected - call Connect() first"))
	}

	factory := getMutationFactory()
	if factory == nil {
		return newInvalidUpdateMutation(fmt.Errorf("no mutation factory registered"))
	}
	return factory.NewUpdate(entity, e.schema, e.connector)
}

// Delete starts a new DELETE mutation
func (e *Engine) Delete(entity string) DeleteMutation {
	if e.schema == nil {
		return newInvalidDeleteMutation(fmt.Errorf("schema not loaded - call LoadSchemaFromFile first"))
	}
	if e.connector == nil {
		return newInvalidDeleteMutation(fmt.Errorf("not connected - call Connect() first"))
	}

	factory := getMutationFactory()
	if factory == nil {
		return newInvalidDeleteMutation(fmt.Errorf("no mutation factory registered"))
	}
	return factory.NewDelete(entity, e.schema, e.connector)
}

// ─────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────

// GetEntity returns an entity by name, or nil if not found
func (s *Schema) GetEntity(name string) *Entity {
	for _, entity := range s.Entities {
		if entity.Name == name {
			return entity
		}
	}
	return nil
}
