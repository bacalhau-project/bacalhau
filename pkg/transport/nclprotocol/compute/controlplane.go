package compute

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
)

// ControlPlane manages the periodic control operations between a compute node and
// the orchestrator. It is responsible for:
// - Sending periodic heartbeats to indicate node health
// - Updating node information when changes occur
// - Maintaining checkpoints of message processing progress
type ControlPlane struct {
	cfg Config // Global configuration for the control plane

	// Core components
	requester          ncl.Publisher                // Used to send messages to orchestrator
	healthTracker      *HealthTracker               // Tracks node health status
	incomingSeqTracker *nclprotocol.SequenceTracker // Tracks processed message sequences
	checkpointName     string                       // Identifier for checkpoint storage

	// State tracking
	latestNodeInfo models.NodeInfo // Cache of most recent node information
	lastCheckpoint uint64          // Last checkpointed sequence number

	// Lifecycle management
	stopCh  chan struct{}  // Signals background goroutines to stop
	wg      sync.WaitGroup // Tracks active background goroutines
	mu      sync.RWMutex   // Protects state changes
	running bool           // Indicates if control plane is active
}

// ControlPlaneParams encapsulates all dependencies needed to create a new ControlPlane
type ControlPlaneParams struct {
	Config             Config
	Requester          ncl.Publisher                // For sending control messages
	HealthTracker      *HealthTracker               // For health monitoring
	IncomingSeqTracker *nclprotocol.SequenceTracker // For sequence tracking
	CheckpointName     string                       // For checkpoint identification
}

// NewControlPlane creates a new ControlPlane instance with the provided parameters.
// It initializes the control plane but does not start any background operations.
func NewControlPlane(params ControlPlaneParams) (*ControlPlane, error) {
	return &ControlPlane{
		cfg:                params.Config,
		requester:          params.Requester,
		healthTracker:      params.HealthTracker,
		incomingSeqTracker: params.IncomingSeqTracker,
		checkpointName:     params.CheckpointName,
		lastCheckpoint:     params.IncomingSeqTracker.GetLastSeqNum(),
		stopCh:             make(chan struct{}),
	}, nil
}

// Start begins the control plane operations. It launches a background goroutine
// that manages periodic tasks:
// - Heartbeat sending
// - Node info updates
// - Progress checkpointing
func (cp *ControlPlane) Start(ctx context.Context) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.running {
		return fmt.Errorf("control plane already running")
	}

	cp.latestNodeInfo = cp.cfg.NodeInfoProvider.GetNodeInfo(ctx)

	cp.wg.Add(1)
	go cp.run(ctx)

	cp.running = true
	return nil
}

// run is the main control loop that manages periodic operations.
// It uses separate timers for each operation type to ensure consistent intervals.
func (cp *ControlPlane) run(ctx context.Context) {
	defer cp.wg.Done()

	// Initialize timers for periodic operations
	heartbeat := time.NewTicker(cp.cfg.HeartbeatInterval)
	nodeInfo := time.NewTicker(cp.cfg.NodeInfoUpdateInterval)
	checkpoint := time.NewTicker(cp.cfg.CheckpointInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-cp.stopCh:
			return

		case <-heartbeat.C:
			if err := cp.heartbeat(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to send heartbeat")
			}
		case <-nodeInfo.C:
			if err := cp.updateNodeInfo(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to update node info")
			}
		case <-checkpoint.C:
			if err := cp.checkpointProgress(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to checkpoint progress")
			}
		}
	}
}

// heartbeat sends a heartbeat message to the orchestrator to indicate the node is alive
// and healthy. It includes:
// - Current available compute capacity
// - Queue usage information
// - Latest processed message sequence number
// Updates health tracking on successful heartbeat.
func (cp *ControlPlane) heartbeat(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, cp.cfg.RequestTimeout)
	defer cancel()

	// Get latest node info for capacity reporting
	nodeInfo := cp.cfg.NodeInfoProvider.GetNodeInfo(ctx)
	cp.latestNodeInfo = nodeInfo

	msg := envelope.NewMessage(messages.HeartbeatRequest{
		NodeID:                 cp.latestNodeInfo.NodeID,
		AvailableCapacity:      nodeInfo.ComputeNodeInfo.AvailableCapacity,
		QueueUsedCapacity:      nodeInfo.ComputeNodeInfo.QueueUsedCapacity,
		LastOrchestratorSeqNum: cp.incomingSeqTracker.GetLastSeqNum(),
	}).WithMetadataValue(envelope.KeyMessageType, messages.HeartbeatRequestMessageType)

	response, err := cp.requester.Request(ctx, ncl.NewPublishRequest(msg))
	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}

	payload, ok := response.GetPayload(messages.HeartbeatResponse{})
	if !ok {
		return fmt.Errorf("invalid heartbeat response payload. expected messages.HeartbeatResponse, got %T", payload)
	}

	cp.healthTracker.HeartbeatSuccess()
	return nil
}

// updateNodeInfo checks for changes in node information and sends updates to the
// orchestrator when changes are detected. This includes changes to:
// - Node capacity
// - Supported features
// - Configuration
// - Labels
// Updates health tracking on successful updates.
func (cp *ControlPlane) updateNodeInfo(ctx context.Context) error {
	// Only send updates when node info has changed
	prevNodeInfo := cp.latestNodeInfo
	cp.latestNodeInfo = cp.cfg.NodeInfoProvider.GetNodeInfo(ctx)
	if !prevNodeInfo.HasStaticConfigChanged(cp.latestNodeInfo) {
		return nil
	}

	log.Debug().Msg("Node info changed, sending update")

	ctx, cancel := context.WithTimeout(ctx, cp.cfg.RequestTimeout)
	defer cancel()

	msg := envelope.NewMessage(messages.UpdateNodeInfoRequest{
		NodeInfo: cp.latestNodeInfo,
	}).WithMetadataValue(envelope.KeyMessageType, messages.NodeInfoUpdateRequestMessageType)

	_, err := cp.requester.Request(ctx, ncl.NewPublishRequest(msg))
	if err != nil {
		return fmt.Errorf("node info update request failed: %w", err)
	}

	cp.healthTracker.UpdateSuccess()
	return nil
}

// checkpointProgress saves the latest processed message sequence number if it has
// changed since the last checkpoint. This allows for resuming message processing
// from the last known point after node restarts.
func (cp *ControlPlane) checkpointProgress(ctx context.Context) error {
	newCheckpoint := cp.incomingSeqTracker.GetLastSeqNum()
	if newCheckpoint == cp.lastCheckpoint {
		return nil
	}
	if err := cp.cfg.Checkpointer.Checkpoint(ctx, cp.checkpointName, newCheckpoint); err != nil {
		log.Error().Err(err).Msg("failed to checkpoint incoming sequence number")
	} else {
		cp.lastCheckpoint = newCheckpoint
	}
	return nil
}

// Stop gracefully shuts down the control plane and waits for background operations
// to complete or until the context is cancelled.
func (cp *ControlPlane) Stop(ctx context.Context) error {
	cp.mu.Lock()
	if !cp.running {
		cp.mu.Unlock()
		return nil
	}

	cp.running = false
	close(cp.stopCh)
	cp.mu.Unlock()

	// Wait for graceful shutdown
	done := make(chan struct{})
	go func() {
		cp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
