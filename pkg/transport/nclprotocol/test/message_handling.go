package test

import (
	"context"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
)

// MockMessageHandler provides a configurable mock message handler for testing.
// It records messages received and allows configuring errors and processing behavior.
type MockMessageHandler struct {
	mu            sync.RWMutex
	messages      []envelope.Message
	shouldProcess bool
	error         error
}

// NewMockMessageHandler creates a new mock message handler
func NewMockMessageHandler() *MockMessageHandler {
	return &MockMessageHandler{
		messages:      make([]envelope.Message, 0),
		shouldProcess: true,
	}
}

// HandleMessage implements ncl.MessageHandler
func (m *MockMessageHandler) HandleMessage(ctx context.Context, msg *envelope.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.error != nil {
		return m.error
	}
	m.messages = append(m.messages, *msg)
	return nil
}

// ShouldProcess implements ncl.MessageHandler
func (m *MockMessageHandler) ShouldProcess(ctx context.Context, msg *envelope.Message) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.shouldProcess
}

// GetMessages returns a copy of all messages received
func (m *MockMessageHandler) GetMessages() []envelope.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]envelope.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// SetError configures an error to be returned by HandleMessage
func (m *MockMessageHandler) SetError(err error) {
	m.mu.Lock()
	m.error = err
	m.mu.Unlock()
}

// compile-time check for interface implementation
var _ ncl.MessageHandler = &MockMessageHandler{}
