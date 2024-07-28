//go:build unit || !integration

package heartbeat

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

const TestTopic = "test_heartbeat"

type HeartbeatTestSuite struct {
	suite.Suite

	clock           *clock.Mock
	natsServer      *server.Server
	natsConn        *nats.Conn
	publisher       ncl.Publisher
	subscriber      ncl.Subscriber
	payloadRegistry *ncl.PayloadRegistry
	heartbeatServer *HeartbeatServer
}

func TestHeartbeatTestSuite(t *testing.T) {
	suite.Run(t, new(HeartbeatTestSuite))
}

func (s *HeartbeatTestSuite) SetupTest() {
	var err error
	s.clock = clock.NewMock()

	// Setup heartbeat server
	s.heartbeatServer, err = NewServer(HeartbeatServerParams{
		Clock:                 s.clock,
		CheckFrequency:        1 * time.Second,
		NodeDisconnectedAfter: 10 * time.Second,
	})
	s.Require().NoError(err)
	s.Require().NoError(s.heartbeatServer.Start(context.Background()))

	// setup nats server and client
	s.natsServer, s.natsConn = testutils.StartNats(s.T())

	// Setup NATS publisher and subscriber
	s.payloadRegistry = ncl.NewPayloadRegistry()
	s.Require().NoError(s.payloadRegistry.Register(HeartbeatMessageType, Heartbeat{}))

	s.publisher, err = ncl.NewPublisher(s.natsConn,
		ncl.WithPublisherName("test-publisher"),
		ncl.WithPublisherDestination(TestTopic),
		ncl.WithPublisherPayloadRegistry(s.payloadRegistry),
	)
	s.Require().NoError(err)

	s.subscriber, err = ncl.NewSubscriber(s.natsConn,
		ncl.WithSubscriberPayloadRegistry(s.payloadRegistry),
		ncl.WithSubscriberMessageHandlers(s.heartbeatServer),
	)
	s.Require().NoError(err)
	s.Require().NoError(s.subscriber.Subscribe(TestTopic))

}

func (s *HeartbeatTestSuite) TearDownTest() {
	if s.subscriber != nil {
		s.subscriber.Close(context.Background())
	}
	if s.natsConn != nil {
		s.natsConn.Close()
	}
	if s.natsServer != nil {
		s.natsServer.Shutdown()
	}
}

func (s *HeartbeatTestSuite) TestHeartbeatScenarios() {
	ctx := context.Background()

	type testcase struct {
		name           string
		includeInitial bool
		heartbeats     []time.Duration
		expectedState  models.NodeConnectionState
		waitUntil      time.Duration
	}

	testcases := []testcase{
		{name: "simple", includeInitial: true, heartbeats: []time.Duration{}, expectedState: models.NodeStates.CONNECTED, waitUntil: time.Duration(5 * time.Second)},
		{
			name:           "disconnected",
			includeInitial: true,
			heartbeats:     []time.Duration{time.Duration(30 * time.Second)},
			expectedState:  models.NodeStates.DISCONNECTED,
			waitUntil:      time.Duration(30 * time.Second),
		},
		{
			name:           "unknown",
			includeInitial: true,
			heartbeats:     []time.Duration{time.Duration(1 * time.Second)},
			expectedState:  models.NodeStates.DISCONNECTED,
			waitUntil:      time.Duration(30 * time.Second),
		},
		{
			name:           "never seen (default)",
			includeInitial: false,
			heartbeats:     []time.Duration{},
			expectedState:  models.NodeStates.DISCONNECTED,
			waitUntil:      time.Duration(10 * time.Second),
		},
	}

	for i, tc := range testcases {
		nodeID := "node-" + strconv.Itoa(i)
		client := NewClient(nodeID, s.publisher)

		s.T().Run(tc.name, func(t *testing.T) {
			var seq uint64 = 1

			if tc.includeInitial {
				err := client.SendHeartbeat(ctx, seq)
				s.Require().NoError(err)
			}

			s.clock.Add(1 * time.Second)

			for _, duration := range tc.heartbeats {
				s.clock.Add(duration)
				seq++
				err := client.SendHeartbeat(ctx, seq)
				s.Require().NoError(err)
			}

			s.clock.Add(tc.waitUntil)

			nodeState := &models.NodeState{Info: models.NodeInfo{NodeID: nodeID}}
			s.heartbeatServer.UpdateNodeInfo(nodeState)
			s.Require().Equal(tc.expectedState, nodeState.Connection, fmt.Sprintf("incorrect state in %s", tc.name))
		})
	}
}

func (s *HeartbeatTestSuite) TestSendHeartbeatError() {
	ctx := context.Background()
	client := NewClient("test-node", s.publisher)

	// Close the NATS connection to force an error
	s.natsConn.Close()

	err := client.SendHeartbeat(ctx, 1)
	s.Error(err)
}
