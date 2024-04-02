//go:build unit || !integration

package heartbeat

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/benbjohnson/clock"
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
)

const (
	TestPort  = 8369
	TestTopic = "test"
)

type HeartbeatTestSuite struct {
	suite.Suite

	clock *clock.Mock

	nats   *server.Server
	client *nats.Conn
}

func TestHeartbeatTestSuite(t *testing.T) {
	suite.Run(t, new(HeartbeatTestSuite))
}

func (s *HeartbeatTestSuite) SetupTest() {
	opts := &natsserver.DefaultTestOptions
	opts.Port = TestPort
	opts.JetStream = true
	opts.StoreDir = s.T().TempDir()

	s.nats = natsserver.RunServer(opts)
	client, err := nats.Connect(s.nats.Addr().String())
	s.Require().NoError(err)

	s.client = client
}

func (s *HeartbeatTestSuite) TearDownTest() {
	s.nats.Shutdown()
}

func (s *HeartbeatTestSuite) TestSendHeartbeat() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s.clock = clock.NewMock()
	server, err := NewServer(HeartbeatServerParams{
		Clock:              s.clock,
		Client:             s.client,
		Topic:              TestTopic,
		CheckFrequency:     1 * time.Second,
		NodeUnhealthyAfter: 10 * time.Second,
		NodeUnknownAfter:   20 * time.Second,
	})
	s.Require().NoError(err)

	err = server.Start(ctx)
	s.Require().NoError(err)

	nodeInfo := models.NodeInfo{
		NodeID: "node1",
	}

	type testcase struct {
		name           string
		includeInitial bool
		heartbeats     []time.Duration
		expectedState  models.NodeState
	}

	testcases := []testcase{
		// No heartbeats, node should be HEALTHY after the initial connection
		{name: "simple", includeInitial: true, heartbeats: []time.Duration{}, expectedState: models.NodeStates.HEALTHY},

		// Node should be HEALTHY after the initial connection and a heartbeat but then misses second
		{
			name:           "unhealhy",
			includeInitial: true,
			heartbeats: []time.Duration{
				time.Duration(1 * time.Second),
				time.Duration(15 * time.Second),
			},
			expectedState: models.NodeStates.UNHEALTHY,
		},

		// Node should be UNKNOWN after missing schedule
		{
			name:           "unknown",
			includeInitial: true,
			heartbeats: []time.Duration{
				time.Duration(1 * time.Second),
				time.Duration(30 * time.Second),
			},
			expectedState: models.NodeStates.UNKNOWN,
		},

		// Nodes that have never been seen should be UNKNOWN
		{
			name:           "never seen",
			includeInitial: false,
			heartbeats:     []time.Duration{},
			expectedState:  models.NodeStates.UNKNOWN,
		},
	}

	for i, tc := range testcases {
		nodeInfo.NodeID = "node-" + strconv.Itoa(i)

		s.T().Run(tc.name, func(t *testing.T) {
			// Wait for the first heartbeat to be sent
			client, err := NewClient(s.client, nodeInfo.NodeID, TestTopic)
			s.Require().NoError(err)
			defer client.Close(ctx)

			var seq uint64 = 1

			// Optionally send initial connection heartbeat
			if tc.includeInitial {
				err = client.SendHeartbeat(ctx, seq)
				s.Require().NoError(err)
			}

			// Wait for the first check frequency to pass before we check the state
			s.clock.Add(1 * time.Second)

			// Send heartbeats after each duration in the test case
			for _, duration := range tc.heartbeats {
				s.clock.Add(duration) // wait for
				seq += 1
				err = client.SendHeartbeat(ctx, seq)
				s.Require().NoError(err)
			}

			server.UpdateNodeInfo(&nodeInfo)
			s.Require().Equal(nodeInfo.State, tc.expectedState)
		})
	}
}
