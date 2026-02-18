package mutation

import "github.com/chameleon-db/chameleondb/chameleon/pkg/engine"

// Factory is STATELESS - schema and connector passed per-call
// This allows registry pattern without import cycles
type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

// NewInsert creates an insert builder with provided schema and connector
func (f *Factory) NewInsert(entity string, schema *engine.Schema, connector *engine.Connector) engine.InsertMutation {
	return NewInsertBuilder(schema, connector, entity)
}

// NewUpdate creates an update builder with provided schema and connector
func (f *Factory) NewUpdate(entity string, schema *engine.Schema, connector *engine.Connector) engine.UpdateMutation {
	return NewUpdateBuilder(schema, connector, entity)
}

// NewDelete creates a delete builder with provided schema and connector
func (f *Factory) NewDelete(entity string, schema *engine.Schema, connector *engine.Connector) engine.DeleteMutation {
	return NewDeleteBuilder(schema, connector, entity)
}
