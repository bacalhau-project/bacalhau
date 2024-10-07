//go:build unit || !integration

package heartbeat

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

const TestTopic = "test_heartbeat"

type HeartbeatTestSuite struct {
	suite.Suite

	clock                *clock.Mock
	natsServer           *server.Server
	natsConn             *nats.Conn
	publisher            ncl.Publisher
	subscriber           ncl.Subscriber
	messageSerDeRegistry *ncl.MessageSerDeRegistry
	heartbeatServer      *HeartbeatServer
}

func TestHeartbeatTestSuite(t *testing.T) {
	suite.Run(t, new(HeartbeatTestSuite))
}

func (s *HeartbeatTestSuite) SetupTest() {
	var err error
	s.clock = clock.NewMock()

	// Setup NATS server and client
	s.natsServer, s.natsConn = testutils.StartNats(s.T())

	// Setup heartbeat server
	s.heartbeatServer, err = NewServer(HeartbeatServerParams{
		NodeID:                "server-node",
		Clock:                 s.clock,
		Client:                s.natsConn,
		NodeDisconnectedAfter: 5 * time.Second,
	})
	s.Require().NoError(err)
	s.Require().NoError(s.heartbeatServer.Start(context.Background()))

	// Setup NATS publisher and subscriber
	s.messageSerDeRegistry = ncl.NewMessageSerDeRegistry()
	s.Require().NoError(s.messageSerDeRegistry.Register(HeartbeatMessageType, Heartbeat{}))

	s.publisher, err = ncl.NewPublisher(s.natsConn,
		ncl.WithPublisherName("test-publisher"),
		ncl.WithPublisherDestination(TestTopic),
		ncl.WithPublisherMessageSerDeRegistry(s.messageSerDeRegistry),
	)
	s.Require().NoError(err)

	s.subscriber, err = ncl.NewSubscriber(s.natsConn,
		ncl.WithSubscriberMessageSerDeRegistry(s.messageSerDeRegistry),
		ncl.WithSubscriberMessageHandlers(s.heartbeatServer),
	)
	s.Require().NoError(err)
	s.Require().NoError(s.subscriber.Subscribe(TestTopic))
	s.Require().NoError(s.subscriber.Subscribe(legacyHeartbeatTopic))
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

func (s *HeartbeatTestSuite) TestUpdateNodeInfo() {
	testCases := []struct {
		name            string
		nodeID          string
		initialLiveness models.NodeConnectionState
		expectedState   models.NodeConnectionState
		hasInitialState bool
	}{
		{
			name:            "Own node",
			nodeID:          s.heartbeatServer.nodeID,
			expectedState:   models.NodeStates.CONNECTED,
			hasInitialState: false,
		},
		{
			name:            "Known node connected",
			nodeID:          "another-node",
			initialLiveness: models.NodeStates.CONNECTED,
			expectedState:   models.NodeStates.CONNECTED,
			hasInitialState: true,
		},
		{
			name:            "Known node unhealthy",
			nodeID:          "another-node",
			initialLiveness: models.NodeStates.DISCONNECTED,
			expectedState:   models.NodeStates.DISCONNECTED,
			hasInitialState: true,
		},
		{
			name:            "Unknown node no state",
			nodeID:          "another-node",
			expectedState:   models.NodeStates.DISCONNECTED,
			hasInitialState: false,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			if tc.hasInitialState {
				s.heartbeatServer.markNodeAs(tc.nodeID, tc.initialLiveness)
			}

			nodeState := &models.NodeState{Info: models.NodeInfo{NodeID: tc.nodeID}}
			s.heartbeatServer.UpdateNodeInfo(nodeState)

			s.Equal(tc.expectedState, nodeState.Connection, "expected %s, got %s", tc.expectedState, nodeState.Connection)
		})
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
		isOwnNode      bool
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
		{
			name:           "own node",
			includeInitial: true,
			heartbeats:     []time.Duration{time.Duration(30 * time.Second)},
			expectedState:  models.NodeStates.HEALTHY,
			waitUntil:      time.Duration(30 * time.Second),
			isOwnNode:      true,
		},
	}

	for i, tc := range testcases {
		nodeID := "node-" + strconv.Itoa(i)
		if tc.isOwnNode {
			nodeID = s.heartbeatServer.nodeID
		}
		client, err := NewClient(s.natsConn, nodeID, s.publisher)
		s.Require().NoError(err)

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
	client, err := NewClient(s.natsConn, "test-node", s.publisher)
	s.Require().NoError(err)

	// Close the NATS connection to force an error
	s.natsConn.Close()

	err = client.SendHeartbeat(ctx, 1)
	s.Error(err)
}

func (s *HeartbeatTestSuite) TestConcurrentHeartbeats() {
	ctx := context.Background()
	numNodes := 10
	numHeartbeatsPerNode := 100

	var wg sync.WaitGroup
	wg.Add(numNodes)

	for i := 0; i < numNodes; i++ {
		go func(nodeID string) {
			defer wg.Done()
			client, err := NewClient(s.natsConn, nodeID, s.publisher)
			require.NoError(s.T(), err)

			for j := 0; j < numHeartbeatsPerNode; j++ {
				s.Require().NoError(client.SendHeartbeat(ctx, uint64(j)))
				time.Sleep(time.Millisecond) // Small delay to simulate real-world scenario
			}
		}(fmt.Sprintf("node-%d", i))
	}

	wg.Wait()

	// Allow time for all heartbeats to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify that all nodes are marked as HEALTHY
	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		nodeState := &models.NodeState{Info: models.NodeInfo{NodeID: nodeID}}
		s.heartbeatServer.UpdateNodeInfo(nodeState)
		s.Require().Equal(models.NodeStates.HEALTHY, nodeState.Connection)
	}
}

func (s *HeartbeatTestSuite) TestConcurrentHeartbeatsWithDisconnection() {
	ctx := context.Background()
	numNodes := 5
	numHeartbeatsPerNode := 50

	var wg sync.WaitGroup
	wg.Add(numNodes)

	for i := 0; i < numNodes; i++ {
		go func(nodeID string) {
			defer wg.Done()
			client, err := NewClient(s.natsConn, nodeID, s.publisher)
			require.NoError(s.T(), err)

			for j := 0; j < numHeartbeatsPerNode; j++ {
				s.Require().NoError(client.SendHeartbeat(ctx, uint64(j)))
				time.Sleep(time.Millisecond)

				if j == numHeartbeatsPerNode/2 {
					// Simulate a disconnection by advancing the clock
					s.clock.Add(10 * time.Second)
				}
			}
		}(fmt.Sprintf("node-%d", i))
	}

	wg.Wait()

	// Allow time for all heartbeats to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify node states
	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		nodeState := &models.NodeState{Info: models.NodeInfo{NodeID: nodeID}}
		s.heartbeatServer.UpdateNodeInfo(nodeState)

		// The exact state might vary depending on timing, but it should be either HEALTHY or DISCONNECTED
		s.Require().Contains([]models.NodeConnectionState{models.NodeStates.HEALTHY, models.NodeStates.DISCONNECTED}, nodeState.Connection)
	}
}

func (s *HeartbeatTestSuite) TestConcurrentHeartbeatsAndChecks() {
	ctx := context.Background()
	numNodes := 5
	numHeartbeatsPerNode := 30
	checkInterval := 50 * time.Millisecond

	var wg sync.WaitGroup
	wg.Add(numNodes + 1) // +1 for the checker goroutine

	// Start the checker goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < numHeartbeatsPerNode; i++ {
			s.heartbeatServer.checkQueue(ctx)
			time.Sleep(checkInterval)
		}
	}()

	for i := 0; i < numNodes; i++ {
		go func(nodeID string) {
			defer wg.Done()
			client, err := NewClient(s.natsConn, nodeID, s.publisher)
			require.NoError(s.T(), err)

			for j := 0; j < numHeartbeatsPerNode; j++ {
				s.Require().NoError(client.SendHeartbeat(ctx, uint64(j)))
				time.Sleep(checkInterval / 2) // Send heartbeats faster than checks
			}
		}(fmt.Sprintf("node-%d", i))
	}

	wg.Wait()

	// Verify final node states
	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		nodeState := &models.NodeState{Info: models.NodeInfo{NodeID: nodeID}}
		s.heartbeatServer.UpdateNodeInfo(nodeState)
		s.Require().Equal(models.NodeStates.HEALTHY, nodeState.Connection)
	}
}
