//go:build unit || !integration

package orchestrator_test

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

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes/inmemory"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol/compute"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol/orchestrator"
)

const TestTopic = "test_heartbeat"

type HeartbeatTestSuite struct {
	suite.Suite

	clock                *clock.Mock
	natsServer           *server.Server
	natsConn             *nats.Conn
	publisher            ncl.Publisher
	subscriber           ncl.Subscriber
	messageSerDeRegistry *envelope.Registry
	heartbeatServer      *orchestrator.Server
	nodeManager          nodes.Manager
	eventStore           watcher.EventStore
}

func TestHeartbeatTestSuite(t *testing.T) {
	suite.Run(t, new(HeartbeatTestSuite))
}

func (s *HeartbeatTestSuite) SetupTest() {
	var err error
	s.clock = clock.NewMock()

	// Setup NATS server and client
	s.natsServer, s.natsConn = testutils.StartNats(s.T())
	s.eventStore, _ = testutils.CreateStringEventStore(s.T())

	// Setup real node manager
	s.nodeManager, err = nodes.NewManager(nodes.ManagerParams{
		Clock:                 s.clock,
		Store:                 inmemory.NewNodeStore(inmemory.NodeStoreParams{TTL: 1 * time.Hour}),
		NodeDisconnectedAfter: 5 * time.Second,
		EventStore:            s.eventStore,
	})
	s.Require().NoError(err)
	s.Require().NoError(s.nodeManager.Start(context.Background()))

	// Setup heartbeat server
	s.heartbeatServer = orchestrator.NewServer(s.nodeManager)

	// Setup NATS publisher and subscriber
	s.messageSerDeRegistry = envelope.NewRegistry()
	s.Require().NoError(s.messageSerDeRegistry.Register(legacy.HeartbeatMessageType, legacy.Heartbeat{}))

	s.publisher, err = ncl.NewPublisher(s.natsConn, ncl.PublisherConfig{
		Name:            "test-publisher",
		Destination:     TestTopic,
		MessageRegistry: s.messageSerDeRegistry,
	})

	s.Require().NoError(err)

	s.subscriber, err = ncl.NewSubscriber(s.natsConn, ncl.SubscriberConfig{
		Name:            "test-subscriber",
		MessageRegistry: s.messageSerDeRegistry,
		MessageHandler:  s.heartbeatServer,
	})
	s.Require().NoError(err)
	s.Require().NoError(s.subscriber.Subscribe(context.Background(), TestTopic))
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
	if s.nodeManager != nil {
		s.nodeManager.Stop(context.Background())
	}
	if s.eventStore != nil {
		s.eventStore.Close(context.Background())
	}
}

func (s *HeartbeatTestSuite) TestUpdateNodeInfo() {
	testCases := []struct {
		name      string
		nodeID    string
		handshake bool
	}{
		{
			name:      "Known node",
			nodeID:    "known-node",
			handshake: true,
		},
		{
			name:      "Unknown node",
			nodeID:    "another-node",
			handshake: false,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			nodeInfo := models.NodeInfo{NodeID: tc.nodeID, NodeType: models.NodeTypeCompute}
			if tc.handshake {
				resp, err := s.nodeManager.Handshake(context.Background(), messages.HandshakeRequest{
					NodeInfo: nodeInfo,
				})
				s.Require().NoError(err)
				s.Require().True(resp.Accepted)
			}

			infoResponse, err := s.heartbeatServer.UpdateInfo(context.Background(), legacy.UpdateInfoRequest{
				Info: nodeInfo,
			})

			if tc.handshake {
				s.Require().NoError(err)
				s.Require().True(infoResponse.Accepted)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *HeartbeatTestSuite) TestHeartbeatScenarios() {
	ctx := context.Background()

	type testcase struct {
		name          string
		handshake     bool
		heartbeats    []time.Duration
		expectedState models.NodeConnectionState
		waitUntil     time.Duration
	}

	testcases := []testcase{
		{
			name:          "simple",
			handshake:     true,
			heartbeats:    []time.Duration{},
			expectedState: models.NodeStates.CONNECTED,
			waitUntil:     time.Duration(5 * time.Second)},
		{
			name:          "disconnected",
			handshake:     true,
			heartbeats:    []time.Duration{time.Duration(30 * time.Second)},
			expectedState: models.NodeStates.DISCONNECTED,
			waitUntil:     time.Duration(30 * time.Second),
		},
		{
			name:          "unknown",
			handshake:     true,
			heartbeats:    []time.Duration{time.Duration(1 * time.Second)},
			expectedState: models.NodeStates.DISCONNECTED,
			waitUntil:     time.Duration(30 * time.Second),
		},
		{
			name:          "never seen (default)",
			handshake:     false,
			heartbeats:    []time.Duration{},
			expectedState: models.NodeStates.DISCONNECTED,
			waitUntil:     time.Duration(10 * time.Second),
		},
	}

	for i, tc := range testcases {
		nodeID := "node-" + strconv.Itoa(i)

		client, err := compute.NewHeartbeatClient(nodeID, s.publisher)
		s.Require().NoError(err)

		nodeInfo := models.NodeInfo{NodeID: nodeID, NodeType: models.NodeTypeCompute}

		s.T().Run(tc.name, func(t *testing.T) {
			var seq uint64 = 1

			if tc.handshake {
				// handshake first
				resp, err := s.nodeManager.Handshake(ctx, messages.HandshakeRequest{
					NodeInfo: nodeInfo,
				})
				s.Require().NoError(err)
				s.Require().True(resp.Accepted)
			}

			for _, duration := range tc.heartbeats {
				s.clock.Add(duration)
				seq++
				err = client.SendHeartbeat(ctx, seq)
				s.Require().NoError(err)
			}

			s.clock.Add(tc.waitUntil)

			nodeState, err := s.nodeManager.Get(ctx, nodeInfo.NodeID)
			if tc.handshake {
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedState, nodeState.ConnectionState.Status, fmt.Sprintf("incorrect state in %s", tc.name))
			} else {
				s.Require().Error(err)
			}
		})
	}
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

			// handshake first
			nodeInfo := models.NodeInfo{NodeID: nodeID, NodeType: models.NodeTypeCompute}
			resp, err := s.nodeManager.Handshake(ctx, messages.HandshakeRequest{
				NodeInfo: nodeInfo,
			})
			s.Require().NoError(err)
			s.Require().True(resp.Accepted)

			// start heartbeating
			client, err := compute.NewHeartbeatClient(nodeID, s.publisher)
			require.NoError(s.T(), err)

			for j := 0; j < numHeartbeatsPerNode; j++ {
				s.Require().NoError(client.SendHeartbeat(ctx, uint64(j)))
				s.clock.Add(time.Millisecond)
			}
		}(fmt.Sprintf("node-%d", i))
	}

	wg.Wait()

	// Allow time for all heartbeats to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify that all nodes are marked as HEALTHY
	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		nodeState, err := s.nodeManager.Get(ctx, nodeID)
		s.Require().NoError(err)
		s.Require().Equal(models.NodeStates.CONNECTED, nodeState.ConnectionState.Status)
	}
}
