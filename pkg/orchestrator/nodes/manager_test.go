//go:build unit || !integration

package nodes_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes/inmemory"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type NodeManagerTestSuite struct {
	suite.Suite
	ctx          context.Context
	clock        *clock.Mock
	store        nodes.Store
	eventStore   watcher.EventStore
	manager      nodes.Manager
	disconnected time.Duration
}

func TestNodeManagerSuite(t *testing.T) {
	suite.Run(t, new(NodeManagerTestSuite))
}

func (s *NodeManagerTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.clock = clock.NewMock()
	s.disconnected = 30 * time.Second

	// Use in-memory store with 1 hour TTL
	s.store = inmemory.NewNodeStore(inmemory.NodeStoreParams{
		TTL: time.Hour,
	})

	s.eventStore, _ = testutils.CreateStringEventStore(s.T())

	manager, err := nodes.NewManager(nodes.ManagerParams{
		Store:                 s.store,
		EventStore:            s.eventStore,
		Clock:                 s.clock,
		NodeDisconnectedAfter: s.disconnected,
		HealthCheckFrequency:  1 * time.Second,
		ManualApproval:        false,
		PersistInterval:       5 * time.Second,
	})
	s.Require().NoError(err)

	err = manager.Start(s.ctx)
	s.Require().NoError(err)
	s.manager = manager
}

func (s *NodeManagerTestSuite) TearDownTest() {
	err := s.manager.Stop(s.ctx)
	s.Require().NoError(err)

	// Cleanup event store
	err = s.eventStore.Close(s.ctx)
	s.Require().NoError(err)
}

func (s *NodeManagerTestSuite) createNodeInfo(id string) models.NodeInfo {
	return models.NodeInfo{
		NodeID:   id,
		NodeType: models.NodeTypeCompute,
		ComputeNodeInfo: models.ComputeNodeInfo{
			AvailableCapacity: models.Resources{CPU: 4, Memory: 8192, GPU: 1},
			QueueUsedCapacity: models.Resources{},
		},
	}
}

// Basic Functionality Tests

func (s *NodeManagerTestSuite) TestSuccessfulHandshake() {
	nodeInfo := s.createNodeInfo("node1")
	req := messages.HandshakeRequest{
		NodeInfo:               nodeInfo,
		LastOrchestratorSeqNum: 1,
	}

	resp, err := s.manager.Handshake(s.ctx, req)
	s.Require().NoError(err)
	assert.True(s.T(), resp.Accepted)

	// Verify node state
	state, err := s.manager.Get(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)
	assert.Equal(s.T(), models.NodeStates.CONNECTED, state.ConnectionState.Status)
	assert.Equal(s.T(), models.NodeMembership.APPROVED, state.Membership)
}

func (s *NodeManagerTestSuite) TestReconnectHandshake() {
	nodeInfo := s.createNodeInfo("node1")

	// First connection
	resp1, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)
	assert.True(s.T(), resp1.Accepted)

	// Simulate disconnect by advancing time
	s.clock.Add(s.disconnected + time.Second)

	// Reconnect
	resp2, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)
	assert.True(s.T(), resp2.Accepted)
	assert.Contains(s.T(), resp2.Reason, "reconnected")
}

func (s *NodeManagerTestSuite) TestHeartbeatMaintainsConnection() {
	// Initial handshake
	nodeInfo := s.createNodeInfo("node1")
	_, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Send heartbeats just before timeout
	for i := 0; i < 3; i++ {
		s.clock.Add(s.disconnected - time.Second)

		_, err = s.manager.Heartbeat(s.ctx, nodes.ExtendedHeartbeatRequest{
			HeartbeatRequest: messages.HeartbeatRequest{
				NodeID:            nodeInfo.ID(),
				AvailableCapacity: models.Resources{CPU: 4},
			},
		})
		s.Require().NoError(err)

		state, err := s.manager.Get(s.ctx, nodeInfo.ID())
		s.Require().NoError(err)
		assert.Equal(s.T(), models.NodeStates.CONNECTED, state.ConnectionState.Status)
	}
}

// Edge Cases and Error Scenarios
func (s *NodeManagerTestSuite) TestHandshakeSequenceNumberLogic() {
	// Test initial handshake with new node
	nodeInfo := s.createNodeInfo("new-node")

	// First add some events to the event store to have a non-zero latest sequence
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		err := s.eventStore.StoreEvent(ctx, watcher.StoreEventRequest{
			Operation:  watcher.OperationCreate,
			ObjectType: testutils.TypeString,
			Object:     fmt.Sprintf("test-event-%d", i),
		})
		s.Require().NoError(err)
	}

	// Get the latest sequence number for verification
	latestSeqNum, err := s.eventStore.GetLatestEventNum(ctx)
	s.Require().NoError(err)

	// Perform initial handshake
	resp1, err := s.manager.Handshake(ctx, messages.HandshakeRequest{
		NodeInfo:               nodeInfo,
		LastOrchestratorSeqNum: 100, // Should be ignored for new nodes
	})
	s.Require().NoError(err)
	s.Require().True(resp1.Accepted)

	// Verify the node was assigned the latest sequence number
	state, err := s.manager.Get(ctx, nodeInfo.ID())
	s.Require().NoError(err)
	s.Assert().Equal(latestSeqNum, state.ConnectionState.LastOrchestratorSeqNum,
		"New node should be assigned latest sequence number")
	s.Assert().Equal(latestSeqNum, resp1.StartingOrchestratorSeqNum,
		"New node should receive latest sequence number as starting point")

	// Update sequence numbers via heartbeat
	updatedOrchSeqNum := uint64(200)
	updatedComputeSeqNum := uint64(150)
	_, err = s.manager.Heartbeat(ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest: messages.HeartbeatRequest{
			NodeID:                 nodeInfo.ID(),
			LastOrchestratorSeqNum: updatedOrchSeqNum,
		},
		LastComputeSeqNum: updatedComputeSeqNum,
	})
	s.Require().NoError(err)

	// Simulate disconnect
	s.clock.Add(s.disconnected + time.Second)
	s.Eventually(func() bool {
		state, err := s.manager.Get(ctx, nodeInfo.ID())
		s.Require().NoError(err)
		return state.ConnectionState.Status == models.NodeStates.DISCONNECTED
	}, 500*time.Millisecond, 20*time.Millisecond)

	// Reconnect with different sequence number - should keep existing
	resp2, err := s.manager.Handshake(ctx, messages.HandshakeRequest{
		NodeInfo:               nodeInfo,
		LastOrchestratorSeqNum: 300, // Should be ignored for reconnecting nodes
	})
	s.Require().NoError(err)
	s.Require().True(resp2.Accepted)
	s.Assert().Contains(resp2.Reason, "reconnected")
	s.Assert().Equal(updatedOrchSeqNum, resp2.StartingOrchestratorSeqNum,
		"Reconnecting node should receive its last known sequence number")

	// Verify sequence numbers were preserved from previous state
	state, err = s.manager.Get(ctx, nodeInfo.ID())
	s.Require().NoError(err)
	s.Assert().Equal(updatedOrchSeqNum, state.ConnectionState.LastOrchestratorSeqNum,
		"Reconnected node should preserve previous orchestrator sequence number")
	s.Assert().Equal(updatedComputeSeqNum, state.ConnectionState.LastComputeSeqNum,
		"Reconnected node should preserve previous compute sequence number")
}

func (s *NodeManagerTestSuite) TestHandshakeSequenceNumberEdgeCases() {
	ctx := context.Background()

	// Test zero sequence numbers in event store
	nodeInfo1 := s.createNodeInfo("zero-seq-node")
	resp1, err := s.manager.Handshake(ctx, messages.HandshakeRequest{
		NodeInfo: nodeInfo1,
	})
	s.Require().NoError(err)
	s.Require().True(resp1.Accepted)

	state1, err := s.manager.Get(ctx, nodeInfo1.ID())
	s.Require().NoError(err)
	s.Assert().Equal(uint64(0), state1.ConnectionState.LastOrchestratorSeqNum,
		"New node should get zero sequence when event store is empty")
	s.Assert().Equal(uint64(0), resp1.StartingOrchestratorSeqNum,
		"New node should receive zero as starting sequence when event store is empty")

	// Test concurrent handshakes with sequence numbers
	var wg sync.WaitGroup
	const numConcurrent = 10

	// Add some events first
	for i := 0; i < 5; i++ {
		err = s.eventStore.StoreEvent(ctx, watcher.StoreEventRequest{
			Operation:  watcher.OperationCreate,
			ObjectType: testutils.TypeString,
			Object:     fmt.Sprintf("test-event-%d", i),
		})
		s.Require().NoError(err)
	}

	latestSeqNum, err := s.eventStore.GetLatestEventNum(ctx)
	s.Require().NoError(err)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			nodeInfo := s.createNodeInfo(fmt.Sprintf("concurrent-node-%d", id))
			resp, err := s.manager.Handshake(ctx, messages.HandshakeRequest{
				NodeInfo:               nodeInfo,
				LastOrchestratorSeqNum: 999, // Should be ignored
			})
			s.Require().NoError(err)
			s.Require().True(resp.Accepted)

			// Verify assigned sequence number
			state, err := s.manager.Get(ctx, nodeInfo.ID())
			s.Require().NoError(err)
			s.Assert().Equal(latestSeqNum, state.ConnectionState.LastOrchestratorSeqNum,
				"Concurrent new nodes should all get latest sequence number")
			s.Assert().Equal(latestSeqNum, resp.StartingOrchestratorSeqNum,
				"Concurrent new nodes should receive latest sequence number as starting point")
		}(i)
	}

	wg.Wait()
}

func (s *NodeManagerTestSuite) TestHeartbeatWithoutHandshake() {
	_, err := s.manager.Heartbeat(s.ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest: messages.HeartbeatRequest{
			NodeID: "nonexistent",
		},
	})
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "handshake required")
}

func (s *NodeManagerTestSuite) TestRejectedNodeHandshake() {
	nodeInfo := s.createNodeInfo("rejected-node")

	// First approve then reject the node
	_, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	err = s.manager.RejectNode(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)

	// Try handshake again
	resp, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)
	assert.False(s.T(), resp.Accepted)
	assert.Contains(s.T(), resp.Reason, "rejected")
}

func (s *NodeManagerTestSuite) TestNodeDisconnectionOnMissedHeartbeats() {
	nodeInfo := s.createNodeInfo("node1")
	_, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Need to give some time for the background health check to run
	s.clock.Add(time.Second)          // Add a second to trigger at least one health check
	time.Sleep(50 * time.Millisecond) // Let the health check goroutine run

	// Advance time past disconnect threshold
	s.clock.Add(s.disconnected + 2*time.Second)

	// eventually check if the node is disconnected
	s.Require().Eventually(func() bool {
		state, err := s.manager.Get(s.ctx, nodeInfo.ID())
		s.Require().NoError(err)
		return state.ConnectionState.Status == models.NodeStates.DISCONNECTED
	}, 500*time.Millisecond, 20*time.Millisecond, "expected node to be disconnected")
}

func (s *NodeManagerTestSuite) TestRejectConnectedNode() {
	nodeInfo := s.createNodeInfo("node1")

	// Connect node
	_, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Verify initial connection
	state, err := s.manager.Get(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)
	assert.Equal(s.T(), models.NodeStates.CONNECTED, state.ConnectionState.Status)

	// Track connection state changes
	var events []nodes.NodeConnectionEvent
	eventsMu := sync.Mutex{}
	s.manager.OnConnectionStateChange(func(event nodes.NodeConnectionEvent) {
		eventsMu.Lock()
		events = append(events, event)
		eventsMu.Unlock()
	})

	// Reject node
	err = s.manager.RejectNode(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)

	// Verify node is disconnected and rejected
	state, err = s.manager.Get(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)
	assert.Equal(s.T(), models.NodeStates.DISCONNECTED, state.ConnectionState.Status)
	assert.Equal(s.T(), models.NodeMembership.REJECTED, state.Membership)
	assert.Equal(s.T(), "node rejected", state.ConnectionState.LastError)

	// Verify connection state change event was emitted
	eventsMu.Lock()
	assert.Len(s.T(), events, 1)
	assert.Equal(s.T(), models.NodeStates.CONNECTED, events[0].Previous)
	assert.Equal(s.T(), models.NodeStates.DISCONNECTED, events[0].Current)
	eventsMu.Unlock()

	// Verify heartbeats are rejected
	_, err = s.manager.Heartbeat(s.ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest: messages.HeartbeatRequest{
			NodeID: nodeInfo.ID(),
		},
	})
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "handshake required")
}

func (s *NodeManagerTestSuite) TestDeleteConnectedNode() {
	nodeInfo := s.createNodeInfo("node1")

	// Connect node
	_, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Track connection state changes
	var events []nodes.NodeConnectionEvent
	eventsMu := sync.Mutex{}
	s.manager.OnConnectionStateChange(func(event nodes.NodeConnectionEvent) {
		eventsMu.Lock()
		events = append(events, event)
		eventsMu.Unlock()
	})

	// Delete node
	err = s.manager.DeleteNode(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)

	// Verify connection state change event was emitted
	eventsMu.Lock()
	assert.Len(s.T(), events, 1)
	assert.Equal(s.T(), models.NodeStates.CONNECTED, events[0].Previous)
	assert.Equal(s.T(), models.NodeStates.DISCONNECTED, events[0].Current)
	eventsMu.Unlock()

	// Verify heartbeats are rejected
	_, err = s.manager.Heartbeat(s.ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest: messages.HeartbeatRequest{
			NodeID: nodeInfo.ID(),
		},
	})
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "handshake required")
}

// Concurrency Tests

func (s *NodeManagerTestSuite) TestConcurrentHandshakes() {
	const numNodes = 100
	var wg sync.WaitGroup
	errors := make(chan error, numNodes)

	for i := 0; i < numNodes; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nodeInfo := s.createNodeInfo(fmt.Sprintf("concurrent-node-%d", id))
			_, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		assert.NoError(s.T(), err)
	}

	// Verify all nodes are registered
	nodes, err := s.manager.List(s.ctx)
	s.Require().NoError(err)
	assert.Len(s.T(), nodes, numNodes)
}

func (s *NodeManagerTestSuite) TestConcurrentHealthCheckAndHeartbeat() {
	nodeInfo := s.createNodeInfo("node1")

	// Connect node
	_, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Set up concurrent heartbeats right at the disconnect threshold
	var wg sync.WaitGroup
	const numConcurrent = 10

	// Advance time close to disconnect threshold
	s.clock.Add(s.disconnected - time.Second)

	// Start concurrent heartbeats
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.manager.Heartbeat(s.ctx, nodes.ExtendedHeartbeatRequest{
				HeartbeatRequest: messages.HeartbeatRequest{
					NodeID: nodeInfo.ID(),
				},
			})
			// Either succeed or fail with concurrent update error
			if err != nil {
				assert.Contains(s.T(), err.Error(), "concurrent update conflict")
			}
		}()
	}

	// Wait for heartbeats to complete
	wg.Wait()

	// Verify node is still connected
	state, err := s.manager.Get(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)
	assert.Equal(s.T(), models.NodeStates.CONNECTED, state.ConnectionState.Status)
}

func (s *NodeManagerTestSuite) TestHeartbeatRetrySuccess() {
	nodeInfo := s.createNodeInfo("node1")

	// Connect node
	_, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Start multiple concurrent heartbeats
	var wg sync.WaitGroup
	responses := make(chan error, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.manager.Heartbeat(s.ctx, nodes.ExtendedHeartbeatRequest{
				HeartbeatRequest: messages.HeartbeatRequest{
					NodeID: nodeInfo.ID(),
				},
			})
			responses <- err
		}()
	}

	wg.Wait()
	close(responses)

	// At least one heartbeat should succeed
	var successCount int
	for err := range responses {
		if err == nil {
			successCount++
		}
	}
	assert.Greater(s.T(), successCount, 0, "At least one heartbeat should succeed")
}

func (s *NodeManagerTestSuite) TestConcurrentOperations() {
	const (
		numNodes             = 50
		numHeartbeatsPerNode = 100
		heartbeatInterval    = time.Millisecond
	)

	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(numNodes)

	manager, err := nodes.NewManager(nodes.ManagerParams{
		Store:                 s.store,
		EventStore:            s.eventStore,
		Clock:                 clock.New(), // Use real clock for this test
		NodeDisconnectedAfter: s.disconnected,
		HealthCheckFrequency:  1 * time.Second,
		ManualApproval:        true,
		PersistInterval:       5 * time.Second,
	})
	s.Require().NoError(err)

	err = manager.Start(s.ctx)
	s.Require().NoError(err)

	for i := 0; i < numNodes; i++ {
		go func(nodeID string) {
			defer wg.Done()

			// Perform handshake
			nodeInfo := s.createNodeInfo(nodeID)
			resp, err := manager.Handshake(ctx, messages.HandshakeRequest{
				NodeInfo: nodeInfo,
			})
			s.Require().NoError(err)
			s.Require().True(resp.Accepted)

			// Send heartbeats with incrementing sequence numbers
			for j := 0; j < numHeartbeatsPerNode; j++ {
				orchSeqNum := uint64(j * 2)      // Orchestrator sequence numbers
				computeSeqNum := uint64(j*2 + 1) // Compute sequence numbers

				heartbeatResp, err := manager.Heartbeat(ctx, nodes.ExtendedHeartbeatRequest{
					HeartbeatRequest: messages.HeartbeatRequest{
						NodeID:                 nodeID,
						LastOrchestratorSeqNum: orchSeqNum,
						AvailableCapacity:      models.Resources{CPU: 4, Memory: 8192},
					},
					LastComputeSeqNum: computeSeqNum,
				})
				s.Require().NoError(err)
				s.Require().Equal(computeSeqNum, heartbeatResp.LastComputeSeqNum)

				time.Sleep(heartbeatInterval)
			}
		}(fmt.Sprintf("node-%d", i))
	}

	wg.Wait()

	// Verify final state of all nodes
	nodes, err := manager.List(ctx)
	s.Require().NoError(err)
	s.Require().Len(nodes, numNodes)

	// Verify each node's state and sequence numbers
	for _, state := range nodes {
		// Verify connection status
		s.Assert().Equal(models.NodeStates.CONNECTED, state.ConnectionState.Status)

		// Verify final sequence numbers
		expectedOrchSeqNum := uint64((numHeartbeatsPerNode - 1) * 2)
		expectedComputeSeqNum := uint64((numHeartbeatsPerNode-1)*2 + 1)

		s.Assert().Equal(expectedOrchSeqNum, state.ConnectionState.LastOrchestratorSeqNum,
			"Node %s should have correct orchestrator sequence number", state.Info.ID())
		s.Assert().Equal(expectedComputeSeqNum, state.ConnectionState.LastComputeSeqNum,
			"Node %s should have correct compute sequence number", state.Info.ID())

		// Verify resources were updated
		s.Assert().Equal(models.Resources{CPU: 4, Memory: 8192}, state.Info.ComputeNodeInfo.AvailableCapacity,
			"Node %s should have correct resources", state.Info.ID())
	}
}

// Resource Management Tests

func (s *NodeManagerTestSuite) TestResourceUpdates() {
	nodeInfo := s.createNodeInfo("resource-node")
	_, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Update resources via heartbeat
	newResources := models.Resources{CPU: 8, Memory: 16384, GPU: 2}
	_, err = s.manager.Heartbeat(s.ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest: messages.HeartbeatRequest{
			NodeID:            nodeInfo.ID(),
			AvailableCapacity: newResources,
		},
	})
	s.Require().NoError(err)

	// Verify resource update
	state, err := s.manager.Get(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)
	assert.Equal(s.T(), newResources, state.Info.ComputeNodeInfo.AvailableCapacity)
}

// State Management Tests

func (s *NodeManagerTestSuite) TestNodeDeletion() {
	nodeInfo := s.createNodeInfo("delete-node")
	_, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Delete node
	err = s.manager.DeleteNode(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)

	// Verify node is deleted
	_, err = s.manager.Get(s.ctx, nodeInfo.ID())
	assert.Error(s.T(), err)
	assert.True(s.T(), bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))
}

func (s *NodeManagerTestSuite) TestConnectionStateChangeEvents() {
	var events []nodes.NodeConnectionEvent
	eventsMu := sync.Mutex{}

	// Register handler before connecting
	s.manager.OnConnectionStateChange(func(event nodes.NodeConnectionEvent) {
		eventsMu.Lock()
		events = append(events, event)
		eventsMu.Unlock()
	})

	nodeInfo := s.createNodeInfo("event-node")

	// Connect
	_, err := s.manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Need to give some time for the background health check to run
	s.clock.Add(time.Second)          // Add a second to trigger at least one health check
	time.Sleep(50 * time.Millisecond) // Let the health check goroutine run

	// Disconnect via timeout
	s.clock.Add(s.disconnected + time.Second)

	// Wait for health check
	s.Require().Eventually(func() bool {
		eventsMu.Lock()
		defer eventsMu.Unlock()
		return len(events) == 2
	}, 500*time.Millisecond, 20*time.Millisecond, "expected 2 events but got %d. %+v", len(events), events)

	eventsMu.Lock()
	defer eventsMu.Unlock()

	require.Len(s.T(), events, 2)
	assert.Equal(s.T(), models.NodeStates.DISCONNECTED, events[0].Previous)
	assert.Equal(s.T(), models.NodeStates.CONNECTED, events[0].Current)
	assert.Equal(s.T(), models.NodeStates.CONNECTED, events[1].Previous)
	assert.Equal(s.T(), models.NodeStates.DISCONNECTED, events[1].Current)
}

// Lifecycle Tests

func (s *NodeManagerTestSuite) TestStartStop() {
	// Create a new manager without starting it
	manager, err := nodes.NewManager(nodes.ManagerParams{
		Store:                 s.store,
		EventStore:            s.eventStore,
		Clock:                 s.clock,
		NodeDisconnectedAfter: s.disconnected,
		HealthCheckFrequency:  1 * time.Second,
		PersistInterval:       100 * time.Millisecond, // Short interval for testing
	})
	s.Require().NoError(err)

	// Start with cancellable context
	err = manager.Start(s.ctx)
	s.Require().NoError(err)
	s.Require().True(manager.Running())

	// Verify it's running by doing a handshake
	nodeInfo := s.createNodeInfo("test-node")
	_, err = manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Stop manager
	err = manager.Stop(s.ctx)
	s.Require().NoError(err)
	s.Require().False(manager.Running())
}
func (s *NodeManagerTestSuite) TestStartAlreadyStarted() {
	// Create and start a manager
	manager, err := nodes.NewManager(nodes.ManagerParams{
		Store:                 s.store,
		EventStore:            s.eventStore,
		Clock:                 s.clock,
		NodeDisconnectedAfter: s.disconnected,
	})
	s.Require().NoError(err)

	// First start should succeed
	err = manager.Start(s.ctx)
	s.Require().NoError(err)
	s.Require().True(manager.Running())

	// Second start should fail
	err = manager.Start(s.ctx)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "already running")
	s.Require().True(manager.Running())

	// Cleanup
	err = manager.Stop(s.ctx)
	s.Require().NoError(err)
	s.Require().False(manager.Running())
}

func (s *NodeManagerTestSuite) TestStartContextCancellation() {
	manager, err := nodes.NewManager(nodes.ManagerParams{
		Store:                 s.store,
		EventStore:            s.eventStore,
		Clock:                 s.clock,
		NodeDisconnectedAfter: s.disconnected,
		HealthCheckFrequency:  1 * time.Second,
	})
	s.Require().NoError(err)

	// Create a context we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Start the manager
	err = manager.Start(ctx)
	s.Require().NoError(err)
	s.Require().True(manager.Running())

	// Cancel the context
	cancel()

	// Verify manager stops gracefully
	s.Eventually(func() bool {
		return !manager.Running()
	}, time.Second, 10*time.Millisecond, "expected manager to stop")
}

func (s *NodeManagerTestSuite) TestStopAlreadyStopped() {
	// Create and start a manager
	manager, err := nodes.NewManager(nodes.ManagerParams{
		Store:                 s.store,
		EventStore:            s.eventStore,
		Clock:                 s.clock,
		NodeDisconnectedAfter: s.disconnected,
	})
	s.Require().NoError(err)

	err = manager.Start(s.ctx)
	s.Require().NoError(err)
	s.Require().True(manager.Running())

	// First stop should succeed
	err = manager.Stop(s.ctx)
	s.Require().NoError(err)
	s.Require().False(manager.Running())

	// Second stop should succeed (idempotent)
	err = manager.Stop(s.ctx)
	assert.NoError(s.T(), err)
	s.Require().False(manager.Running())
}

// Persistence Tests

func (s *NodeManagerTestSuite) TestPeriodicStatePersistence() {
	// Create manager with short persist interval
	persistInterval := 100 * time.Millisecond
	manager, err := nodes.NewManager(nodes.ManagerParams{
		Store:                 s.store,
		EventStore:            s.eventStore,
		Clock:                 s.clock,
		NodeDisconnectedAfter: s.disconnected,
		PersistInterval:       persistInterval,
	})
	s.Require().NoError(err)
	err = manager.Start(s.ctx)
	s.Require().NoError(err)

	s.T().Cleanup(func() {
		s.Require().NoError(manager.Stop(s.ctx))
	})

	// advance clock once to trigger persistence
	s.clock.Add(persistInterval + time.Millisecond)
	time.Sleep(50 * time.Millisecond) // Let the persistence goroutine run

	// Connect a node
	nodeInfo := s.createNodeInfo("persistence-test")
	_, err = manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Update node resources
	newResources := models.Resources{CPU: 8, Memory: 16384}
	lastOrchestratorSeqNum := uint64(123)
	lastComputeSeqNum := uint64(456)
	_, err = manager.Heartbeat(s.ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest: messages.HeartbeatRequest{
			NodeID:                 nodeInfo.ID(),
			AvailableCapacity:      newResources,
			LastOrchestratorSeqNum: lastOrchestratorSeqNum,
		},
		LastComputeSeqNum: lastComputeSeqNum,
	})
	s.Require().NoError(err)

	// Advance clock and wait for persistence
	s.clock.Add(persistInterval + time.Millisecond)

	s.Eventually(func() bool {
		state, err := s.store.Get(s.ctx, nodeInfo.ID())
		if err != nil {
			return false
		}
		return state.ConnectionState.LastComputeSeqNum == lastComputeSeqNum
	}, time.Second, 10*time.Millisecond)

	// Verify state was persisted
	state, err := s.store.Get(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)
	assert.Equal(s.T(), models.NodeStates.CONNECTED, state.ConnectionState.Status)
	assert.Equal(s.T(), newResources, state.Info.ComputeNodeInfo.AvailableCapacity)
	assert.Equal(s.T(), lastOrchestratorSeqNum, state.ConnectionState.LastOrchestratorSeqNum)
	assert.Equal(s.T(), lastComputeSeqNum, state.ConnectionState.LastComputeSeqNum)

	// Cleanup
	err = manager.Stop(s.ctx)
	s.Require().NoError(err)
}

func (s *NodeManagerTestSuite) TestStatePersistenceOnStop() {
	// Create manager
	manager, err := nodes.NewManager(nodes.ManagerParams{
		Store:                 s.store,
		EventStore:            s.eventStore,
		Clock:                 s.clock,
		NodeDisconnectedAfter: s.disconnected,
		PersistInterval:       time.Hour, // Long interval to ensure persistence happens on stop
	})
	s.Require().NoError(err)
	err = manager.Start(s.ctx)
	s.Require().NoError(err)

	// Connect a node
	nodeInfo := s.createNodeInfo("persistence-test")
	_, err = manager.Handshake(s.ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Update node resources
	newResources := models.Resources{CPU: 8, Memory: 16384}
	lastOrchestratorSeqNum := uint64(123)
	lastComputeSeqNum := uint64(456)
	_, err = manager.Heartbeat(s.ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest: messages.HeartbeatRequest{
			NodeID:                 nodeInfo.ID(),
			AvailableCapacity:      newResources,
			LastOrchestratorSeqNum: lastOrchestratorSeqNum,
		},
		LastComputeSeqNum: lastComputeSeqNum,
	})
	s.Require().NoError(err)

	// Stop manager - should trigger persistence
	err = manager.Stop(s.ctx)
	s.Require().NoError(err)

	// Verify state was persisted
	state, err := s.store.Get(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)
	assert.Equal(s.T(), models.NodeStates.CONNECTED, state.ConnectionState.Status)
	assert.Equal(s.T(), newResources, state.Info.ComputeNodeInfo.AvailableCapacity)
	assert.Equal(s.T(), lastOrchestratorSeqNum, state.ConnectionState.LastOrchestratorSeqNum)
	assert.Equal(s.T(), lastComputeSeqNum, state.ConnectionState.LastComputeSeqNum)
}

func (s *NodeManagerTestSuite) TestPersistenceWithContextCancellation() {
	// Create manager with short persist interval
	manager, err := nodes.NewManager(nodes.ManagerParams{
		Store:                 s.store,
		EventStore:            s.eventStore,
		Clock:                 s.clock,
		NodeDisconnectedAfter: s.disconnected,
		PersistInterval:       100 * time.Millisecond,
	})
	s.Require().NoError(err)

	// Create a cancelable context
	ctx, cancel := context.WithCancel(s.ctx)
	err = manager.Start(ctx)
	s.Require().NoError(err)

	// Connect a node
	nodeInfo := s.createNodeInfo("persistence-test")
	_, err = manager.Handshake(ctx, messages.HandshakeRequest{NodeInfo: nodeInfo})
	s.Require().NoError(err)

	// Update node resources
	newResources := models.Resources{CPU: 8, Memory: 16384}
	lastOrchestratorSeqNum := uint64(123)
	lastComputeSeqNum := uint64(456)
	_, err = manager.Heartbeat(s.ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest: messages.HeartbeatRequest{
			NodeID:                 nodeInfo.ID(),
			AvailableCapacity:      newResources,
			LastOrchestratorSeqNum: lastOrchestratorSeqNum,
		},
		LastComputeSeqNum: lastComputeSeqNum,
	})
	s.Require().NoError(err)

	// Cancel context before persistence interval
	cancel()

	s.Eventually(func() bool {
		state, err := s.store.Get(s.ctx, nodeInfo.ID())
		if err != nil {
			return false
		}
		return state.ConnectionState.LastComputeSeqNum == lastComputeSeqNum
	}, time.Second, 10*time.Millisecond)

	// Verify state was persisted
	state, err := s.store.Get(s.ctx, nodeInfo.ID())
	s.Require().NoError(err)
	assert.Equal(s.T(), models.NodeStates.CONNECTED, state.ConnectionState.Status)
	assert.Equal(s.T(), newResources, state.Info.ComputeNodeInfo.AvailableCapacity)
	assert.Equal(s.T(), lastOrchestratorSeqNum, state.ConnectionState.LastOrchestratorSeqNum)
	assert.Equal(s.T(), lastComputeSeqNum, state.ConnectionState.LastComputeSeqNum)
}
