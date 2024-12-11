//go:build unit || !integration

package compute_test

import (
	"context"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
	nclprotocolcompute "github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/compute"
	ncltest "github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/test"
)

type DataPlaneTestSuite struct {
	suite.Suite
	ctx        context.Context
	cancel     context.CancelFunc
	natsConn   *nats.Conn
	natsServer *natsserver.Server
	dataPlane  *nclprotocolcompute.DataPlane
	config     nclprotocolcompute.Config
	logServer  *ncltest.MockLogStreamServer
	msgChan    chan *envelope.Message
	sub        ncl.Subscriber
}

func TestDataPlaneTestSuite(t *testing.T) {
	suite.Run(t, new(DataPlaneTestSuite))
}

func (s *DataPlaneTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Start NATS server and get client connection
	s.natsServer, s.natsConn = testutils.StartNats(s.T())

	// Create basic config
	s.config = nclprotocolcompute.Config{
		NodeID:                  "test-node",
		LogStreamServer:         &ncltest.MockLogStreamServer{},
		DataPlaneMessageCreator: &ncltest.MockMessageCreator{},
		EventStore:              testutils.CreateComputeEventStore(s.T()),
	}
	s.config.SetDefaults()

	s.setupSubscriber()

	// Create data plane
	dp, err := nclprotocolcompute.NewDataPlane(nclprotocolcompute.DataPlaneParams{
		Config:             s.config,
		Client:             s.natsConn,
		LastReceivedSeqNum: 0,
	})
	s.Require().NoError(err)
	s.dataPlane = dp
}

func (s *DataPlaneTestSuite) setupSubscriber() {
	s.msgChan = make(chan *envelope.Message, 10) // Buffer multiple messages
	sub, err := ncl.NewSubscriber(s.natsConn, ncl.SubscriberConfig{
		Name:            "test-subscriber",
		MessageRegistry: s.config.MessageRegistry,
		MessageHandler: ncl.MessageHandlerFunc(func(ctx context.Context, msg *envelope.Message) error {
			s.msgChan <- msg
			return nil
		}),
	})
	s.Require().NoError(err)

	err = sub.Subscribe(s.ctx, nclprotocol.NatsSubjectComputeOutMsgs(s.config.NodeID))
	s.Require().NoError(err)
	s.sub = sub
}

func (s *DataPlaneTestSuite) TearDownTest() {
	if s.sub != nil {
		s.sub.Close(context.Background())
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
	close(s.msgChan)
}

func (s *DataPlaneTestSuite) TestLifecycle() {
	testCases := []struct {
		name        string
		operation   func() error
		verifyState func() bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "first start succeeds",
			operation:   func() error { return s.dataPlane.Start(s.ctx) },
			verifyState: func() bool { return s.dataPlane.IsRunning() },
			expectError: false,
		},
		{
			name:        "second start fails",
			operation:   func() error { return s.dataPlane.Start(s.ctx) },
			verifyState: func() bool { return s.dataPlane.IsRunning() },
			expectError: true,
			errorMsg:    "already running",
		},
		{
			name:        "first stop succeeds",
			operation:   func() error { return s.dataPlane.Stop(s.ctx) },
			verifyState: func() bool { return !s.dataPlane.IsRunning() },
			expectError: false,
		},
		{
			name:        "second stop is noop",
			operation:   func() error { return s.dataPlane.Stop(s.ctx) },
			verifyState: func() bool { return !s.dataPlane.IsRunning() },
			expectError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.operation()
			if tc.expectError {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errorMsg)
			} else {
				s.Require().NoError(err)
			}
			s.Require().True(tc.verifyState())
		})
	}
}

func (s *DataPlaneTestSuite) TestStartupFailureCleanup() {
	testCases := []struct {
		name        string
		preStart    func()
		verifyFail  func(error)
		expectError string
	}{
		{
			name:        "NATS connection failure",
			preStart:    func() { s.natsConn.Close() },
			expectError: "connection closed",
		},
		{
			name: "context cancellation",
			preStart: func() {
				s.cancel()
				select {
				case <-s.ctx.Done():
				    // Context cancellation has propagated
				case <-time.After(100 * time.Millisecond):
				    s.Require().Fail("Timeout waiting for context cancellation")
				}
			},
			expectError: "context canceled",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.preStart()
			err := s.dataPlane.Start(s.ctx)
			s.Require().Error(err)
			s.Require().Contains(err.Error(), tc.expectError)
			s.Require().False(s.dataPlane.IsRunning())
			s.Require().Nil(s.dataPlane.Publisher)
			s.Require().Nil(s.dataPlane.Dispatcher)
		})
	}
}

func (s *DataPlaneTestSuite) TestMessageHandling() {
	s.Require().NoError(s.dataPlane.Start(s.ctx))

	testCases := []struct {
		name           string
		event          models.ExecutionUpsert
		expectMessage  bool
		expectedMsgTyp string
	}{
		{
			name: "valid execution upsert",
			event: models.ExecutionUpsert{
				Current: &models.Execution{
					ID:     "test-job-1",
					NodeID: "test-node",
				},
			},
			expectMessage:  true,
			expectedMsgTyp: messages.BidResultMessageType,
		},
		{
			name: "another execution upsert",
			event: models.ExecutionUpsert{
				Current: &models.Execution{
					ID:     "test-job-2",
					NodeID: "test-node",
				},
			},
			expectMessage:  true,
			expectedMsgTyp: messages.BidResultMessageType,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := s.config.EventStore.StoreEvent(s.ctx, watcher.StoreEventRequest{
				Operation:  watcher.OperationCreate,
				ObjectType: compute.EventObjectExecutionUpsert,
				Object:     tc.event,
			})
			s.Require().NoError(err)

			if tc.expectMessage {
				select {
				case msg := <-s.msgChan:
					s.Require().Equal(tc.expectedMsgTyp, msg.Metadata.Get(envelope.KeyMessageType))
				case <-time.After(time.Second):
					s.Require().Fail("Timeout waiting for message")
				}
			}
		})
	}
}
