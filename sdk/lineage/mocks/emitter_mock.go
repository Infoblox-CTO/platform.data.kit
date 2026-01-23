package mocks

import (
	"context"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/lineage"
)

// MockEmitter is a mock implementation of lineage.Emitter for testing.
type MockEmitter struct {
	EmitFunc  func(ctx context.Context, event *lineage.Event) error
	CloseFunc func() error
	Events    []*lineage.Event
}

func (m *MockEmitter) Emit(ctx context.Context, event *lineage.Event) error {
	m.Events = append(m.Events, event)
	if m.EmitFunc != nil {
		return m.EmitFunc(ctx, event)
	}
	return nil
}

func (m *MockEmitter) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func NewMockEmitter() *MockEmitter {
	return &MockEmitter{
		Events: make([]*lineage.Event, 0),
	}
}

var _ lineage.Emitter = (*MockEmitter)(nil)
