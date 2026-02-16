package mutation

import "github.com/chameleon-db/chameleondb/chameleon/pkg/engine"

type Factory struct {
	schema *engine.Schema
}

func NewFactory(schema *engine.Schema) *Factory {
	return &Factory{schema: schema}
}

// NewInsert creates an insert builder
func (f *Factory) NewInsert(entity string) engine.InsertMutation {
	return NewInsertBuilder(f.schema, entity)
}

// NewUpdate creates an update builder
func (f *Factory) NewUpdate(entity string) engine.UpdateMutation {
	return NewUpdateBuilder(f.schema, entity)
}

// NewDelete creates a delete builder
func (f *Factory) NewDelete(entity string) engine.DeleteMutation {
	return NewDeleteBuilder(f.schema, entity)
}
