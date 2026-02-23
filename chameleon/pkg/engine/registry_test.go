package engine

import (
	"context"
	"testing"
)

type mockMutationFactory struct{}

type mockInsertMutation struct{}

func (m *mockInsertMutation) Set(field string, value interface{}) InsertMutation {
	return m
}
func (m *mockInsertMutation) Debug() InsertMutation {
	return m
}
func (m *mockInsertMutation) Execute(ctx context.Context) (*InsertResult, error) {
	return &InsertResult{}, nil
}

type mockUpdateMutation struct{}

func (m *mockUpdateMutation) Set(field string, value interface{}) UpdateMutation {
	return m
}
func (m *mockUpdateMutation) Filter(field string, operator string, value interface{}) UpdateMutation {
	return m
}
func (m *mockUpdateMutation) Debug() UpdateMutation {
	return m
}
func (m *mockUpdateMutation) Execute(ctx context.Context) (*UpdateResult, error) {
	return &UpdateResult{}, nil
}

type mockDeleteMutation struct{}

func (m *mockDeleteMutation) Filter(field string, operator string, value interface{}) DeleteMutation {
	return m
}
func (m *mockDeleteMutation) Debug() DeleteMutation {
	return m
}
func (m *mockDeleteMutation) Execute(ctx context.Context) (*DeleteResult, error) {
	return &DeleteResult{}, nil
}

func (m *mockMutationFactory) NewInsert(entity string, schema *Schema, connector *Connector) InsertMutation {
	return &mockInsertMutation{}
}

func (m *mockMutationFactory) NewUpdate(entity string, schema *Schema, connector *Connector) UpdateMutation {
	return &mockUpdateMutation{}
}

func (m *mockMutationFactory) NewDelete(entity string, schema *Schema, connector *Connector) DeleteMutation {
	return &mockDeleteMutation{}
}

func TestRegisterMutationFactory(t *testing.T) {
	// Reset global state
	mutationFactory = nil

	factory := &mockMutationFactory{}
	RegisterMutationFactory(factory)

	if mutationFactory == nil {
		t.Error("Expected mutationFactory to be set to the registered factory")
	}
}

func TestRegisterMutationFactory_NilFactory(t *testing.T) {
	// Reset global state
	mutationFactory = nil

	RegisterMutationFactory(nil)

	if mutationFactory != nil {
		t.Error("Expected mutationFactory to remain nil when registering nil factory")
	}
}

func TestRegisterMutationFactory_OnlyOnce(t *testing.T) {
	// Reset global state
	mutationFactory = nil

	firstFactory := &mockMutationFactory{}
	secondFactory := &mockMutationFactory{}

	RegisterMutationFactory(firstFactory)
	RegisterMutationFactory(secondFactory)

	// The factory should still be the first one (implementation only allows registering once)
	if mutationFactory == nil {
		t.Error("Expected mutationFactory to be set")
	}

	// Verify it's the first factory by checking that it wasn't replaced
	// Since we can't compare directly, we test the behavior
	RegisterMutationFactory(nil) // This should not change it
	if mutationFactory == nil {
		t.Error("Factory should not be replaced")
	}
}

func TestGetMutationFactory(t *testing.T) {
	// Reset global state
	mutationFactory = nil

	factory := &mockMutationFactory{}
	RegisterMutationFactory(factory)

	retrieved := getMutationFactory()

	if retrieved == nil {
		t.Error("Expected getMutationFactory to return the registered factory")
	}
}

func TestGetMutationFactory_NotSet(t *testing.T) {
	// Reset global state
	mutationFactory = nil

	retrieved := getMutationFactory()

	if retrieved != nil {
		t.Error("Expected getMutationFactory to return nil when no factory is registered")
	}
}
