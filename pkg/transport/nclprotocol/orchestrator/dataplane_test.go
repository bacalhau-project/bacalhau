//go:build unit || !integration

package orchestrator_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/orchestrator"
	ncltest "github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/test"
)

// TestMessage represents a test message to be sent/received
type TestMessage struct {
	Name            string
	Message         interface{}
	Type            string
	Sequence        uint64
	ExpectProcessed bool
}

type DataPlaneTestSuite struct {
	suite.Suite
	ctx               context.Context
	cancel            context.CancelFunc
	natsConn          *nats.Conn
	natsServer        *natsserver.Server
	dataPlane         *orchestrator.DataPlane
	config            orchestrator.DataPlaneConfig
	msgHandler        *ncltest.MockMessageHandler
	msgCreatorFactory *ncltest.MockMessageCreatorFactory
	msgCreator        *ncltest.MockMessageCreator

	// Test message passing
	publisher ncl.Publisher  // For sending test messages
	consumer  ncl.Subscriber // For receiving published messages
}

func (s *DataPlaneTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Start NATS server and get client connection
	s.natsServer, s.natsConn = testutils.StartNats(s.T())

	// Create mocks
	s.msgHandler = ncltest.NewMockMessageHandler()
	s.msgCreatorFactory = ncltest.NewMockMessageCreatorFactory("test-node")
	s.msgCreator = s.msgCreatorFactory.GetCreator()

	// Create basic config
	s.config = orchestrator.DataPlaneConfig{
		NodeID:                "test-node",
		Client:                s.natsConn,
		MessageHandler:        s.msgHandler,
		MessageCreatorFactory: s.msgCreatorFactory,
		MessageRegistry:       nclprotocol.MustCreateMessageRegistry(),
		MessageSerializer:     envelope.NewSerializer(),
		EventStore:            testutils.CreateJobEventStore(s.T()),
	}

	// Setup test message passing
	s.setupMessagePassing()

	// Create data plane
	dp, err := orchestrator.NewDataPlane(s.config)
	s.Require().NoError(err)
	s.dataPlane = dp
}

func (s *DataPlaneTestSuite) setupMessagePassing() {
	var err error

	// Create publisher for sending test messages
	s.publisher, err = ncl.NewPublisher(s.natsConn, ncl.PublisherConfig{
		Name:            "test-publisher",
		MessageRegistry: s.config.MessageRegistry,
		Destination:     nclprotocol.NatsSubjectOrchestratorInMsgs(s.config.NodeID),
	})
	s.Require().NoError(err)

	// Create subscriber for consuming outgoing messages
	s.consumer, err = ncl.NewSubscriber(s.natsConn, ncl.SubscriberConfig{
		Name:              "test-consumer",
		MessageRegistry:   s.config.MessageRegistry,
		MessageSerializer: s.config.MessageSerializer,
		MessageHandler:    s.msgHandler,
	})
	s.Require().NoError(err)

	err = s.consumer.Subscribe(s.ctx, nclprotocol.NatsSubjectOrchestratorOutMsgs(s.config.NodeID))
	s.Require().NoError(err)
}

func (s *DataPlaneTestSuite) TearDownTest() {
	if s.consumer != nil {
		s.consumer.Close(context.Background())
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.dataPlane != nil {
		s.dataPlane.Stop(context.Background())
	}
	if s.natsConn != nil {
		s.natsConn.Close()
	}
	if s.natsServer != nil {
		s.natsServer.Shutdown()
	}
}

func TestDataPlaneTestSuite(t *testing.T) {
	suite.Run(t, new(DataPlaneTestSuite))
}

func (s *DataPlaneTestSuite) TestLifecycle() {
	testCases := []struct {
		name        string
		operation   func() error
		expectError bool
		errorMsg    string
	}{
		{
			name:        "first start succeeds",
			operation:   func() error { return s.dataPlane.Start(s.ctx) },
			expectError: false,
		},
		{
			name:        "second start fails",
			operation:   func() error { return s.dataPlane.Start(s.ctx) },
			expectError: true,
			errorMsg:    "already running",
		},
		{
			name:        "first stop succeeds",
			operation:   func() error { return s.dataPlane.Stop(s.ctx) },
			expectError: false,
		},
		{
			name:        "second stop is noop",
			operation:   func() error { return s.dataPlane.Stop(s.ctx) },
			expectError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.operation()
			if tc.expectError {
				s.Error(err)
				s.Contains(err.Error(), tc.errorMsg)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *DataPlaneTestSuite) TestIncomingMessageProcessing() {
	s.Require().NoError(s.dataPlane.Start(s.ctx))

	testMessages := []TestMessage{
		{
			Name: "bid result message",
			Message: messages.BidResult{
				BaseResponse: messages.BaseResponse{ExecutionID: "test-1"},
			},
			Type:            messages.BidResultMessageType,
			Sequence:        1,
			ExpectProcessed: true,
		},
		{
			Name: "run result message",
			Message: messages.RunResult{
				BaseResponse: messages.BaseResponse{ExecutionID: "test-2"},
			},
			Type:            messages.RunResultMessageType,
			Sequence:        2,
			ExpectProcessed: true,
		},
	}

	// Send test messages
	for _, tm := range testMessages {
		s.Run(tm.Name, func() {
			s.sendTestMessage(tm)

			if tm.ExpectProcessed {
				s.verifyMessageProcessed(tm)
			}
		})
	}

	// Verify sequence tracking
	s.Equal(uint64(2), s.dataPlane.GetLastProcessedSequence())
}

func (s *DataPlaneTestSuite) TestEventToMessageDispatch() {
	s.Require().NoError(s.dataPlane.Start(s.ctx))

	execution := &models.Execution{
		ID:     "test-1",
		NodeID: s.config.NodeID,
	}

	// Configure message to be created
	expectedMsg := envelope.NewMessage(messages.BidResult{
		BaseResponse: messages.BaseResponse{ExecutionID: execution.ID},
	}).WithMetadataValue(envelope.KeyMessageType, messages.BidResultMessageType)
	s.msgCreator.SetNextMessage(expectedMsg)

	// Store execution event
	err := s.config.EventStore.StoreEvent(s.ctx, watcher.StoreEventRequest{
		Operation:  watcher.OperationCreate,
		ObjectType: jobstore.EventObjectExecutionUpsert,
		Object: models.ExecutionUpsert{
			Current: execution,
		},
	})
	s.Require().NoError(err)

	// Wait for message to be published
	s.Eventually(func() bool {
		msgs := s.msgHandler.GetMessages()
		return len(msgs) > 0 && msgs[0].Metadata.Get(envelope.KeyMessageType) == messages.BidResultMessageType
	}, time.Second, 10*time.Millisecond)
}

func (s *DataPlaneTestSuite) TestSequenceTracking() {
	s.Require().NoError(s.dataPlane.Start(s.ctx))

	// Send messages with sequential sequence numbers
	numMessages := 5
	for i := 1; i <= numMessages; i++ {
		msg := envelope.NewMessage(messages.BidResult{
			BaseResponse: messages.BaseResponse{ExecutionID: fmt.Sprintf("test-%d", i)},
		}).
			WithMetadataValue(envelope.KeyMessageType, messages.BidResultMessageType).
			WithMetadataValue(nclprotocol.KeySeqNum, fmt.Sprint(i))

		err := s.publisher.Publish(s.ctx, ncl.NewPublishRequest(msg))
		s.Require().NoError(err)
	}

	// Verify final sequence number
	s.Eventually(func() bool {
		return s.dataPlane.GetLastProcessedSequence() == uint64(numMessages)
	}, time.Second, 10*time.Millisecond)

	// Verify messages were processed in order
	msgs := s.msgHandler.GetMessages()
	s.Len(msgs, numMessages)
	for i, msg := range msgs {
		s.Equal(fmt.Sprint(i+1), msg.Metadata.Get(nclprotocol.KeySeqNum))
	}
}

// Helper methods

func (s *DataPlaneTestSuite) sendTestMessage(tm TestMessage) {
	msg := envelope.NewMessage(tm.Message).
		WithMetadataValue(envelope.KeyMessageType, tm.Type).
		WithMetadataValue(nclprotocol.KeySeqNum, fmt.Sprint(tm.Sequence))

	err := s.publisher.Publish(s.ctx, ncl.NewPublishRequest(msg))
	s.Require().NoError(err)
}

func (s *DataPlaneTestSuite) verifyMessageProcessed(tm TestMessage) {
	s.Eventually(func() bool {
		msgs := s.msgHandler.GetMessages()
		for _, msg := range msgs {
			if msg.Metadata.Get(envelope.KeyMessageType) == tm.Type &&
				msg.Metadata.Get(nclprotocol.KeySeqNum) == fmt.Sprint(tm.Sequence) {
				return true
			}
		}
		return false
	}, time.Second, 10*time.Millisecond)
}
