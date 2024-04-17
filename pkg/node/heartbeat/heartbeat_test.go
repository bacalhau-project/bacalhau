//go:build unit || !integration

package heartbeat

import (
	"context"
	"fmt"
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
		Clock:                 s.clock,
		Client:                s.client,
		Topic:                 TestTopic,
		CheckFrequency:        1 * time.Second,
		NodeDisconnectedAfter: 10 * time.Second,
	})
	s.Require().NoError(err)

	err = server.Start(ctx)
	s.Require().NoError(err)

	type testcase struct {
		name           string
		includeInitial bool
		heartbeats     []time.Duration
		expectedState  models.NodeState
		waitUntil      time.Duration
	}

	testcases := []testcase{
		// No heartbeats, node should be HEALTHY after the initial connection
		{name: "simple", includeInitial: true, heartbeats: []time.Duration{}, expectedState: models.NodeStates.CONNECTED, waitUntil: time.Duration(5 * time.Second)},

		// Node should be CONNECTED after the initial connection and a heartbeat but then misses second
		{
			name:           "disconnected",
			includeInitial: true,
			heartbeats: []time.Duration{
				time.Duration(30 * time.Second),
			},
			expectedState: models.NodeStates.DISCONNECTED,
			waitUntil:     time.Duration(30 * time.Second),
		},

		// Node should be DISCONNECTED after missing schedule
		{
			name:           "unknown",
			includeInitial: true,
			heartbeats: []time.Duration{
				time.Duration(1 * time.Second),
				// time.Duration(30 * time.Second),
			},
			expectedState: models.NodeStates.DISCONNECTED,
			waitUntil:     time.Duration(30 * time.Second),
		},

		// Nodes that have never been seen should be DISCONNECTED
		{
			name:           "never seen (default)",
			includeInitial: false,
			heartbeats:     []time.Duration{},
			expectedState:  models.NodeStates.DISCONNECTED,
			waitUntil:      time.Duration(10 * time.Second),
		},
	}

	for i, tc := range testcases {
		nodeInfo := models.NodeInfo{
			NodeID: "node-" + strconv.Itoa(i),
		}

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

			s.clock.Add(tc.waitUntil)

			server.UpdateNodeInfo(&nodeInfo)
			s.Require().Equal(nodeInfo.State, tc.expectedState, fmt.Sprintf("incorrect state in %s", tc.name))
		})
	}
}
