package test

import (
	"context"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
)

// MockResponderBehavior configures how the mock responder handles different request types.
// It allows customizing responses and errors for testing different scenarios.
type MockResponderBehavior struct {
	// HandshakeResponse controls behavior for handshake requests
	HandshakeResponse struct {
		Error    error                      // Error to return, if any
		Response messages.HandshakeResponse // Response to return if no error
	}

	// HeartbeatResponse controls behavior for heartbeat requests
	HeartbeatResponse struct {
		Error    error                      // Error to return, if any
		Response messages.HeartbeatResponse // Response to return if no error
	}

	// NodeInfoResponse controls behavior for node info update requests
	NodeInfoResponse struct {
		Error    error                           // Error to return, if any
		Response messages.UpdateNodeInfoResponse // Response to return if no error
	}

	// ShutdownResponse controls behavior for shutdown notifications
	ShutdownResponse struct {
		Error    error                           // Error to return, if any
		Response messages.ShutdownNoticeResponse // Response to return if no error
	}

	// Callbacks for request inspection
	OnHandshake func(messages.HandshakeRequest)      // Called when handshake received
	OnHeartbeat func(messages.HeartbeatRequest)      // Called when heartbeat received
	OnNodeInfo  func(messages.UpdateNodeInfoRequest) // Called when node info update received
	OnShutdown  func(messages.ShutdownNoticeRequest)
}

// MockResponder provides a configurable mock implementation of the control plane responder.
// It tracks requests received and provides configurable responses for testing.
type MockResponder struct {
	behavior  *MockResponderBehavior
	responder ncl.Responder
	mu        sync.RWMutex

	// Request history
	handshakes []messages.HandshakeRequest
	heartbeats []messages.HeartbeatRequest
	nodeInfos  []messages.UpdateNodeInfoRequest
	shutdowns  []messages.ShutdownNoticeRequest
}

// NewMockResponder creates a new mock responder with the given behavior.
// If behavior is nil, default success responses are used.
func NewMockResponder(ctx context.Context, conn *nats.Conn, behavior *MockResponderBehavior) (*MockResponder, error) {
	if behavior == nil {
		behavior = &MockResponderBehavior{
			HandshakeResponse: struct {
				Error    error
				Response messages.HandshakeResponse
			}{
				Response: messages.HandshakeResponse{Accepted: true},
			},
			HeartbeatResponse: struct {
				Error    error
				Response messages.HeartbeatResponse
			}{
				Response: messages.HeartbeatResponse{},
			},
			NodeInfoResponse: struct {
				Error    error
				Response messages.UpdateNodeInfoResponse
			}{
				Response: messages.UpdateNodeInfoResponse{Accepted: true},
			},
			ShutdownResponse: struct {
				Error    error
				Response messages.ShutdownNoticeResponse
			}{
				Response: messages.ShutdownNoticeResponse{}, // Empty response for success
			},
		}
	}

	responder, err := ncl.NewResponder(conn, ncl.ResponderConfig{
		Name:              "mock-responder",
		MessageRegistry:   nclprotocol.MustCreateMessageRegistry(),
		MessageSerializer: envelope.NewSerializer(),
		Subject:           nclprotocol.NatsSubjectOrchestratorInCtrl(),
	})
	if err != nil {
		return nil, fmt.Errorf("create responder: %w", err)
	}

	mr := &MockResponder{
		behavior:  behavior,
		responder: responder,
	}

	if err := mr.setupHandlers(ctx); err != nil {
		_ = responder.Close(ctx)
		return nil, err
	}

	return mr, nil
}

func (m *MockResponder) setupHandlers(ctx context.Context) error {
	// Handshake handler
	if err := m.responder.Listen(ctx, messages.HandshakeRequestMessageType,
		ncl.RequestHandlerFunc(func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			req := *msg.Payload.(*messages.HandshakeRequest)
			m.recordHandshake(req)

			if m.behavior.HandshakeResponse.Error != nil {
				return nil, m.behavior.HandshakeResponse.Error
			}
			return envelope.NewMessage(m.behavior.HandshakeResponse.Response).
				WithMetadataValue(envelope.KeyMessageType, messages.HandshakeResponseType), nil
		})); err != nil {
		return err
	}

	// Heartbeat handler
	if err := m.responder.Listen(ctx, messages.HeartbeatRequestMessageType,
		ncl.RequestHandlerFunc(func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			req := *msg.Payload.(*messages.HeartbeatRequest)
			m.recordHeartbeat(req)

			if m.behavior.HeartbeatResponse.Error != nil {
				return nil, m.behavior.HeartbeatResponse.Error
			}
			return envelope.NewMessage(m.behavior.HeartbeatResponse.Response).
				WithMetadataValue(envelope.KeyMessageType, messages.HeartbeatResponseType), nil
		})); err != nil {
		return err
	}

	// Node info handler
	if err := m.responder.Listen(ctx, messages.NodeInfoUpdateRequestMessageType,
		ncl.RequestHandlerFunc(func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			req := *msg.Payload.(*messages.UpdateNodeInfoRequest)
			m.recordNodeInfo(req)

			if m.behavior.NodeInfoResponse.Error != nil {
				return nil, m.behavior.NodeInfoResponse.Error
			}
			return envelope.NewMessage(m.behavior.NodeInfoResponse.Response).
				WithMetadataValue(envelope.KeyMessageType, messages.NodeInfoUpdateResponseType), nil
		})); err != nil {
		return err
	}

	// Shutdown notification handler
	if err := m.responder.Listen(ctx, messages.ShutdownNoticeRequestMessageType,
		ncl.RequestHandlerFunc(func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			req := *msg.Payload.(*messages.ShutdownNoticeRequest)
			m.recordShutdown(req)

			if m.behavior.ShutdownResponse.Error != nil {
				return nil, m.behavior.ShutdownResponse.Error
			}
			return envelope.NewMessage(m.behavior.ShutdownResponse.Response).
				WithMetadataValue(envelope.KeyMessageType, messages.ShutdownNoticeResponseType), nil
		})); err != nil {
		return err
	}

	return nil
}

// Record methods for inspection
func (m *MockResponder) recordHandshake(req messages.HandshakeRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handshakes = append(m.handshakes, req)
	if m.behavior.OnHandshake != nil {
		m.behavior.OnHandshake(req)
	}
}

func (m *MockResponder) recordHeartbeat(req messages.HeartbeatRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.heartbeats = append(m.heartbeats, req)
	if m.behavior.OnHeartbeat != nil {
		m.behavior.OnHeartbeat(req)
	}
}

func (m *MockResponder) recordNodeInfo(req messages.UpdateNodeInfoRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeInfos = append(m.nodeInfos, req)
	if m.behavior.OnNodeInfo != nil {
		m.behavior.OnNodeInfo(req)
	}
}

func (m *MockResponder) recordShutdown(req messages.ShutdownNoticeRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shutdowns = append(m.shutdowns, req)
	if m.behavior.OnShutdown != nil {
		m.behavior.OnShutdown(req)
	}
}

// GetHandshakes returns a copy of all handshake requests received
func (m *MockResponder) GetHandshakes() []messages.HandshakeRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]messages.HandshakeRequest, len(m.handshakes))
	copy(result, m.handshakes)
	return result
}

// GetHeartbeats returns a copy of all heartbeat requests received
func (m *MockResponder) GetHeartbeats() []messages.HeartbeatRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]messages.HeartbeatRequest, len(m.heartbeats))
	copy(result, m.heartbeats)
	return result
}

// GetNodeInfos returns a copy of all node info update requests received
func (m *MockResponder) GetNodeInfos() []messages.UpdateNodeInfoRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]messages.UpdateNodeInfoRequest, len(m.nodeInfos))
	copy(result, m.nodeInfos)
	return result
}

func (m *MockResponder) GetShutdowns() []messages.ShutdownNoticeRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]messages.ShutdownNoticeRequest, len(m.shutdowns))
	copy(result, m.shutdowns)
	return result
}

// Behaviour returns the behavior configuration
func (m *MockResponder) Behaviour() *MockResponderBehavior {
	return m.behavior
}

// Close shuts down the responder
func (m *MockResponder) Close(ctx context.Context) error {
	return m.responder.Close(ctx)
}

// MockCheckpointer provides a thread-safe mock implementation of Checkpointer for testing.
// It tracks checkpoints and allows configuring errors and validation behavior.
type MockCheckpointer struct {
	mu          sync.RWMutex
	checkpoints map[string]uint64    // Stored checkpoint values by name
	setErrors   map[string]error     // Errors to return for Checkpoint calls by name
	getErrors   map[string]error     // Errors to return for GetCheckpoint calls by name
	onSet       func(string, uint64) // Optional callback when checkpoint is set
	onGet       func(string)         // Optional callback when checkpoint is retrieved
}

// NewMockCheckpointer creates a new mock checkpointer instance
func NewMockCheckpointer() *MockCheckpointer {
	return &MockCheckpointer{
		checkpoints: make(map[string]uint64),
		setErrors:   make(map[string]error),
		getErrors:   make(map[string]error),
	}
}

// Checkpoint implements the Checkpointer interface
func (m *MockCheckpointer) Checkpoint(ctx context.Context, name string, sequenceNumber uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for configured error
	if err := m.setErrors[name]; err != nil {
		return err
	}

	// Store checkpoint
	m.checkpoints[name] = sequenceNumber

	// Call optional callback
	if m.onSet != nil {
		m.onSet(name, sequenceNumber)
	}

	return nil
}

// GetCheckpoint implements the Checkpointer interface
func (m *MockCheckpointer) GetCheckpoint(ctx context.Context, name string) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check for configured error
	if err := m.getErrors[name]; err != nil {
		return 0, err
	}

	// Call optional callback
	if m.onGet != nil {
		m.onGet(name)
	}

	// Return stored value or 0 if not found
	return m.checkpoints[name], nil
}

// Helper methods for test configuration

// SetCheckpoint directly sets a checkpoint value
func (m *MockCheckpointer) SetCheckpoint(name string, value uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkpoints[name] = value
}

// SetCheckpointError configures an error to be returned by Checkpoint
func (m *MockCheckpointer) SetCheckpointError(name string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setErrors[name] = err
}

// SetGetCheckpointError configures an error to be returned by GetCheckpoint
func (m *MockCheckpointer) SetGetCheckpointError(name string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getErrors[name] = err
}

// OnCheckpointSet sets a callback to be called when Checkpoint is called
func (m *MockCheckpointer) OnCheckpointSet(callback func(name string, value uint64)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onSet = callback
}

// OnCheckpointGet sets a callback to be called when GetCheckpoint is called
func (m *MockCheckpointer) OnCheckpointGet(callback func(name string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onGet = callback
}

// GetStoredCheckpoint returns the currently stored checkpoint value
func (m *MockCheckpointer) GetStoredCheckpoint(name string) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.checkpoints[name]
	if !exists {
		return 0, fmt.Errorf("no checkpoint found for %s", name)
	}
	return value, nil
}

// Reset clears all stored checkpoints and configured errors
func (m *MockCheckpointer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkpoints = make(map[string]uint64)
	m.setErrors = make(map[string]error)
	m.getErrors = make(map[string]error)
	m.onSet = nil
	m.onGet = nil
}

// compile-time check for interface implementation
var _ nclprotocol.Checkpointer = &MockCheckpointer{}
