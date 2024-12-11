package test

import (
	"context"
	"fmt"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
)

// MockMessageCreator provides a configurable implementation of MessageCreator for testing.
// It allows setting predefined messages or errors to be returned.
type MockMessageCreator struct {
	// Error if set, CreateMessage will return this error
	Error error

	// Message if set, CreateMessage will return this message
	// If nil and Error is nil, a default BidResult message is returned
	Message *envelope.Message
}

// CreateMessage implements nclprotocol.MessageCreator
func (c *MockMessageCreator) CreateMessage(event watcher.Event) (*envelope.Message, error) {
	if c.Error != nil {
		return nil, c.Error
	}
	if c.Message != nil {
		return c.Message, nil
	}
	// Return default message if no specific behavior configured
	return envelope.NewMessage(messages.BidResult{}).
		WithMetadataValue(envelope.KeyMessageType, messages.BidResultMessageType), nil
}

// SetNextMessage configures the next message to be returned by CreateMessage
func (c *MockMessageCreator) SetNextMessage(msg *envelope.Message) {
	c.Message = msg
}

// MockMessageCreatorFactory manages MockMessageCreator instances for testing.
// It provides a thread-safe way to create and configure message creators per node.
type MockMessageCreatorFactory struct {
	nodeID      string              // Expected node ID for validation
	mockCreator *MockMessageCreator // Shared mock creator instance
	createError error               // Error to return from CreateMessageCreator
	mu          sync.RWMutex        // Protects concurrent access
}

// NewMockMessageCreatorFactory creates a new factory that validates against the given nodeID
func NewMockMessageCreatorFactory(nodeID string) *MockMessageCreatorFactory {
	return &MockMessageCreatorFactory{
		nodeID:      nodeID,
		mockCreator: &MockMessageCreator{},
	}
}

// CreateMessageCreator implements nclprotocol.MessageCreatorFactory.
// Returns the mock creator if nodeID matches, error otherwise.
func (f *MockMessageCreatorFactory) CreateMessageCreator(ctx context.Context, nodeID string) (nclprotocol.MessageCreator, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.createError != nil {
		return nil, f.createError
	}

	if nodeID != f.nodeID {
		return nil, fmt.Errorf("unknown node ID: %s", nodeID)
	}

	return f.mockCreator, nil
}

// GetCreator provides access to the underlying mock creator for configuration
func (f *MockMessageCreatorFactory) GetCreator() *MockMessageCreator {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.mockCreator
}

// SetCreateError configures an error to be returned by CreateMessageCreator
func (f *MockMessageCreatorFactory) SetCreateError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createError = err
}

// Ensure interface implementations
var (
	_ nclprotocol.MessageCreator        = &MockMessageCreator{}
	_ nclprotocol.MessageCreatorFactory = &MockMessageCreatorFactory{}
)
