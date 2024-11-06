//go:build unit || !integration

package watchers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type MockEventHandler struct {
	Called    bool
	LastEvent watcher.Event
	Error     error
}

func (m *MockEventHandler) HandleEvent(_ context.Context, event watcher.Event) error {
	m.Called = true
	m.LastEvent = event
	return m.Error
}

type DispatcherTestSuite struct {
	suite.Suite
	dispatcher  *Dispatcher
	handlers    map[models.Protocol]watcher.EventHandler
	bProtocol   *MockEventHandler
	nclProtocol *MockEventHandler
}

func (s *DispatcherTestSuite) SetupTest() {
	s.bProtocol = &MockEventHandler{}
	s.nclProtocol = &MockEventHandler{}

	s.handlers = map[models.Protocol]watcher.EventHandler{
		models.ProtocolBProtocolV2: s.bProtocol,
		models.ProtocolNCLV1:       s.nclProtocol,
	}

	dispatcher, err := NewDispatcher(s.handlers)
	s.Require().NoError(err)
	s.dispatcher = dispatcher
}

func (s *DispatcherTestSuite) createEvent(execution *models.Execution) watcher.Event {
	return watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
		},
	}
}

func (s *DispatcherTestSuite) TestNewDispatcher_Validation() {
	// Test with empty handlers
	_, err := NewDispatcher(map[models.Protocol]watcher.EventHandler{})
	s.Error(err, "should error with empty handlers")

	// Test with nil handler
	_, err = NewDispatcher(map[models.Protocol]watcher.EventHandler{
		models.ProtocolBProtocolV2: nil,
	})
	s.Error(err, "should error with nil handler")

	// Test with valid handlers
	dispatcher, err := NewDispatcher(s.handlers)
	s.NoError(err, "should not error with valid handlers")
	s.NotNil(dispatcher, "should create dispatcher")
}

func (s *DispatcherTestSuite) TestHandleEvent_InvalidEvent() {
	// Test with invalid event object
	err := s.dispatcher.HandleEvent(context.Background(), watcher.Event{
		Object: "invalid",
	})
	s.Error(err, "should error with invalid event object")
}

func (s *DispatcherTestSuite) TestHandleEvent_DefaultProtocol() {
	// Create execution without protocol in meta
	execution := mock.Execution()
	delete(execution.Job.Meta, models.MetaOrchestratorProtocol)

	event := s.createEvent(execution)
	err := s.dispatcher.HandleEvent(context.Background(), event)
	s.NoError(err, "should not error with default protocol")

	// Verify BProtocol handler was called
	s.True(s.bProtocol.Called, "should call bprotocol handler")
	s.False(s.nclProtocol.Called, "should not call nclProtocol handler")
}

func (s *DispatcherTestSuite) TestHandleEvent_SpecificProtocol() {
	// Test NCL protocol
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorProtocol] = string(models.ProtocolNCLV1)

	event := s.createEvent(execution)
	err := s.dispatcher.HandleEvent(context.Background(), event)
	s.NoError(err, "should not error with NCL protocol")

	// Verify NCL handler was called
	s.False(s.bProtocol.Called, "should not call bprotocol handler")
	s.True(s.nclProtocol.Called, "should call nclProtocol handler")

	// Test BProtocol
	s.SetupTest() // Reset handlers
	execution.Job.Meta[models.MetaOrchestratorProtocol] = string(models.ProtocolBProtocolV2)

	event = s.createEvent(execution)
	err = s.dispatcher.HandleEvent(context.Background(), event)
	s.NoError(err, "should not error with BProtocol")

	// Verify BProtocol handler was called
	s.True(s.bProtocol.Called, "should call bprotocol handler")
	s.False(s.nclProtocol.Called, "should not call nclProtocol handler")
}

func (s *DispatcherTestSuite) TestHandleEvent_UnknownProtocol() {
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorProtocol] = "unknown"

	event := s.createEvent(execution)
	err := s.dispatcher.HandleEvent(context.Background(), event)
	s.Error(err, "should error with unknown protocol")

	// Verify no handlers were called
	s.False(s.bProtocol.Called, "should not call bprotocol handler")
	s.False(s.nclProtocol.Called, "should not call nclProtocol handler")
}

func (s *DispatcherTestSuite) TestHandleEvent_HandlerError() {
	// Set error on handler
	s.bProtocol.Error = context.Canceled

	execution := mock.Execution()
	delete(execution.Job.Meta, models.MetaOrchestratorProtocol) // Use default BProtocol

	event := s.createEvent(execution)
	err := s.dispatcher.HandleEvent(context.Background(), event)
	s.ErrorIs(err, context.Canceled, "should propagate handler error")

	// Verify handler was called despite error
	s.True(s.bProtocol.Called, "should call handler even with error")
}

func TestDispatcherTestSuite(t *testing.T) {
	suite.Run(t, new(DispatcherTestSuite))
}
