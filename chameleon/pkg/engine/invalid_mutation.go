package engine

import (
	"context"
)

type invalidInsertMutation struct {
	err error
}

func newInvalidInsertMutation(err error) InsertMutation {
	return &invalidInsertMutation{err: err}
}

func (m *invalidInsertMutation) Set(field string, value interface{}) InsertMutation {
	return m
}

func (m *invalidInsertMutation) Debug() InsertMutation {
	return m
}

func (m *invalidInsertMutation) Execute(ctx context.Context) (*InsertResult, error) {
	return nil, m.err
}

type invalidUpdateMutation struct {
	err error
}

func newInvalidUpdateMutation(err error) UpdateMutation {
	return &invalidUpdateMutation{err: err}
}

func (m *invalidUpdateMutation) Set(field string, value interface{}) UpdateMutation {
	return m
}

func (m *invalidUpdateMutation) Filter(field string, operator string, value interface{}) UpdateMutation {
	return m
}

func (m *invalidUpdateMutation) Debug() UpdateMutation {
	return m
}

func (m *invalidUpdateMutation) Execute(ctx context.Context) (*UpdateResult, error) {
	return nil, m.err
}

type invalidDeleteMutation struct {
	err error
}

func newInvalidDeleteMutation(err error) DeleteMutation {
	return &invalidDeleteMutation{err: err}
}

func (m *invalidDeleteMutation) Filter(field string, operator string, value interface{}) DeleteMutation {
	return m
}

func (m *invalidDeleteMutation) Debug() DeleteMutation {
	return m
}

func (m *invalidDeleteMutation) Execute(ctx context.Context) (*DeleteResult, error) {
	return nil, m.err
}
