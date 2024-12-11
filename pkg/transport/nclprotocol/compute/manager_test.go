//go:build unit || !integration

package compute_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	natsutil "github.com/bacalhau-project/bacalhau/pkg/nats"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
	nclprotocolcompute "github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/compute"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/dispatcher"
	ncltest "github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/test"
)

type ConnectionManagerTestSuite struct {
	suite.Suite
	ctx              context.Context
	cancel           context.CancelFunc
	clock            clock.Clock
	clientFactory    natsutil.ClientFactory
	nodeInfoProvider *ncltest.MockNodeInfoProvider
	messageHandler   *ncltest.MockMessageHandler
	checkpointer     *ncltest.MockCheckpointer
	manager          *nclprotocolcompute.ConnectionManager
	mockResponder    *ncltest.MockResponder
	config           nclprotocolcompute.Config
	natsServer       *server.Server
	natsConn         *nats.Conn
}

func (s *ConnectionManagerTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.clock = clock.New() // tickers didn't work properly with mock clock

	// Setup NATS server and client
	s.natsServer, s.natsConn = testutils.StartNats(s.T())

	// Fresh client with each call
	s.clientFactory = natsutil.ClientFactoryFunc(func(ctx context.Context) (*nats.Conn, error) {
		return testutils.CreateNatsClient(s.T(), s.natsServer.ClientURL()), nil
	})

	// Create mocks
	s.nodeInfoProvider = ncltest.NewMockNodeInfoProvider()
	s.messageHandler = ncltest.NewMockMessageHandler()
	s.checkpointer = ncltest.NewMockCheckpointer()

	// Setup base configuration
	s.config = nclprotocolcompute.Config{
		NodeID:           "test-node",
		NodeInfoProvider: s.nodeInfoProvider,
		ClientFactory:    s.clientFactory,
		Checkpointer:     s.checkpointer,
		EventStore:       testutils.CreateComputeEventStore(s.T()),
		LogStreamServer:  &ncltest.MockLogStreamServer{},

		DataPlaneMessageHandler: s.messageHandler,
		DataPlaneMessageCreator: &ncltest.MockMessageCreator{},

		Clock:                  s.clock,
		HeartbeatInterval:      100 * time.Millisecond,
		HeartbeatMissFactor:    3,
		NodeInfoUpdateInterval: 100 * time.Millisecond,
		CheckpointInterval:     1 * time.Second,
		ReconnectInterval:      100 * time.Millisecond,
		RequestTimeout:         50 * time.Millisecond,
		ReconnectBackoff:       backoff.NewExponential(50*time.Millisecond, 100*time.Millisecond),
		DispatcherConfig:       dispatcher.DefaultConfig(),
	}

	// Setup mock responder
	mockResponder, err := ncltest.NewMockResponder(s.ctx, s.natsConn, nil)
	s.Require().NoError(err)
	s.mockResponder = mockResponder

	// Create manager
	manager, err := nclprotocolcompute.NewConnectionManager(s.config)
	s.Require().NoError(err)
	s.manager = manager
}

// TearDownTest
func (s *ConnectionManagerTestSuite) TearDownTest() {
	if s.manager != nil {
		s.Require().NoError(s.manager.Close(context.Background()))
	}
	if s.mockResponder != nil {
		s.Require().NoError(s.mockResponder.Close(context.Background()))
	}
	if s.natsConn != nil {
		s.natsConn.Close()
	}
	if s.natsServer != nil {
		s.natsServer.Shutdown()
	}

	s.cancel()
}

func (s *ConnectionManagerTestSuite) TestSuccessfulConnection() {
	// Setup initial checkpoint
	lastOrchestratorSeqNum := uint64(124)
	s.checkpointer.SetCheckpoint("incoming-test-node", lastOrchestratorSeqNum)

	err := s.manager.Start(s.ctx)
	s.Require().NoError(err)

	s.Require().Eventually(func() bool {
		return len(s.mockResponder.GetHandshakes()) > 0
	}, time.Second, 10*time.Millisecond, "handshake not received")

	// Verify handshake request
	handshakes := s.mockResponder.GetHandshakes()
	s.Require().Len(handshakes, 1)
	s.Require().Equal(s.config.NodeID, handshakes[0].NodeInfo.ID())
	s.Require().Equal(lastOrchestratorSeqNum, handshakes[0].LastOrchestratorSeqNum)

	// Verify connection established
	s.Require().Eventually(func() bool {
		health := s.manager.GetHealth()
		return health.CurrentState == nclprotocol.Connected
	}, time.Second, 10*time.Millisecond, "manager did not connect")

	// verify no heartbeats yet
	s.Require().Empty(s.mockResponder.GetHeartbeats())

	// trigger heartbeat
	previousTick := s.manager.GetHealth().LastSuccessfulHeartbeat
	time.Sleep(s.config.HeartbeatInterval)

	// wait for some heartbeats
	s.Require().Eventually(func() bool {
		return len(s.mockResponder.GetHeartbeats()) > 0
	}, time.Second, 10*time.Millisecond, "manager did not send heartbeats")

	// Verify heartbeat content
	nodeInfo := s.nodeInfoProvider.GetNodeInfo(s.ctx)
	heartbeats := s.mockResponder.GetHeartbeats()
	s.Require().Len(heartbeats, 1)
	s.Require().Equal(messages.HeartbeatRequest{
		NodeID:                 nodeInfo.NodeID,
		AvailableCapacity:      nodeInfo.ComputeNodeInfo.AvailableCapacity,
		QueueUsedCapacity:      nodeInfo.ComputeNodeInfo.QueueUsedCapacity,
		LastOrchestratorSeqNum: lastOrchestratorSeqNum,
	}, heartbeats[0])

	// verify state
	s.Require().Greater(s.manager.GetHealth().LastSuccessfulHeartbeat, previousTick)

	// update node info and heartbeat again
	nodeInfo.ComputeNodeInfo.AvailableCapacity = models.Resources{CPU: 100, Memory: 1000, GPU: 3}
	nodeInfo.ComputeNodeInfo.QueueUsedCapacity = models.Resources{CPU: 10, Memory: 100, GPU: 1}
	s.nodeInfoProvider.SetNodeInfo(nodeInfo)

	// trigger heartbeat
	time.Sleep(s.config.HeartbeatInterval)
	s.Require().Eventually(func() bool {
		lastHeartbeat := s.mockResponder.GetHeartbeats()[len(s.mockResponder.GetHeartbeats())-1]
		return reflect.DeepEqual(lastHeartbeat, messages.HeartbeatRequest{
			NodeID:                 nodeInfo.NodeID,
			AvailableCapacity:      nodeInfo.ComputeNodeInfo.AvailableCapacity,
			QueueUsedCapacity:      nodeInfo.ComputeNodeInfo.QueueUsedCapacity,
			LastOrchestratorSeqNum: lastOrchestratorSeqNum,
		})
	}, time.Second, 10*time.Millisecond, "manager did not send heartbeats")

}

func (s *ConnectionManagerTestSuite) TestRejectedHandshake() {
	// Configure responder to reject handshake
	s.mockResponder.Behaviour().HandshakeResponse.Response = messages.HandshakeResponse{
		Accepted: false,
		Reason:   "node not allowed",
	}

	err := s.manager.Start(s.ctx)
	s.Require().NoError(err)

	// Verify disconnected state
	s.Require().Eventually(func() bool {
		health := s.manager.GetHealth()
		return health.CurrentState == nclprotocol.Disconnected &&
			health.LastError != nil &&
			health.ConsecutiveFailures > 0
	}, time.Second, 10*time.Millisecond)

	// Allow handshake and verify reconnection
	s.mockResponder.Behaviour().HandshakeResponse.Response = messages.HandshakeResponse{
		Accepted: true,
	}

	// Retry handshake
	time.Sleep(s.config.ReconnectInterval)
	s.Require().Eventually(func() bool {
		health := s.manager.GetHealth()
		return health.CurrentState == nclprotocol.Connected
	}, time.Second, 10*time.Millisecond, "manager should be connected")
}

func (s *ConnectionManagerTestSuite) TestHeartbeatFailure() {
	err := s.manager.Start(s.ctx)
	s.Require().NoError(err)

	// Wait for initial connection
	s.Require().Eventually(func() bool {
		health := s.manager.GetHealth()
		return health.CurrentState == nclprotocol.Connected
	}, time.Second, 10*time.Millisecond)

	// Configure heartbeat failure
	s.mockResponder.Behaviour().HeartbeatResponse.Error = fmt.Errorf("heartbeat failed")

	// Wait for disconnect after missed heartbeats
	time.Sleep(s.config.HeartbeatInterval * time.Duration(s.config.HeartbeatMissFactor+1))

	// Should disconnect after missing heartbeats
	s.Require().Eventually(func() bool {
		health := s.manager.GetHealth()
		return health.CurrentState == nclprotocol.Disconnected &&
			health.LastError != nil
	}, time.Second, 10*time.Millisecond)
}

func (s *ConnectionManagerTestSuite) TestNodeInfoUpdates() {
	// Configure heartbeat callback to trigger node info updates
	s.mockResponder.Behaviour().OnHeartbeat = func(req messages.HeartbeatRequest) {
		newInfo := s.nodeInfoProvider.GetNodeInfo(s.ctx)
		newInfo.Labels = map[string]string{"heartbeat": time.Now().String()}
		s.nodeInfoProvider.SetNodeInfo(newInfo)
	}

	err := s.manager.Start(s.ctx)
	s.Require().NoError(err)

	// Wait for connection
	s.Require().Eventually(func() bool {
		health := s.manager.GetHealth()
		return health.CurrentState == nclprotocol.Connected
	}, time.Second, 10*time.Millisecond)

	// Verify node info updates received
	s.Require().Eventually(func() bool {
		return len(s.mockResponder.GetNodeInfos()) > 0
	}, time.Second, 10*time.Millisecond)

	nodeInfos := s.mockResponder.GetNodeInfos()
	s.Require().Len(nodeInfos, 1)
	s.Require().Equal(
		s.nodeInfoProvider.GetNodeInfo(s.ctx).ID(),
		nodeInfos[0].NodeInfo.ID(),
	)
}

func TestConnectionManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionManagerTestSuite))
}
