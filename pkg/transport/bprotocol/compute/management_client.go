package compute

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
)

type ManagementClientParams struct {
	NodeInfoProvider models.NodeInfoProvider
	ManagementProxy  bprotocol.ManagementEndpoint
	HeartbeatClient  *HeartbeatClient
	HeartbeatConfig  types.Heartbeat
}

// ManagementClient is used to call management functions with
// the requester nodes, via the NATS transport. When `Start`ed
// it will periodically send an update to the requester node with
// the latest node info for this node.
type ManagementClient struct {
	nodeInfoProvider models.NodeInfoProvider
	done             chan struct{}
	managementProxy  bprotocol.ManagementEndpoint
	heartbeatClient  *HeartbeatClient
	heartbeatConfig  types.Heartbeat
}

func NewManagementClient(params *ManagementClientParams) *ManagementClient {
	return &ManagementClient{
		done:             make(chan struct{}, 1),
		nodeInfoProvider: params.NodeInfoProvider,
		managementProxy:  params.ManagementProxy,
		heartbeatClient:  params.HeartbeatClient,
		heartbeatConfig:  params.HeartbeatConfig,
	}
}

// RegisterNode sends a registration request to the requester node. If we successfully
// register, a sentinel file is created to indicate that we are registered. If present
// the requester node will know it is already registered.  If not present, it will
// attempt to register again, expecting the requester node to gracefully handle any
// previous registrations.
func (m *ManagementClient) RegisterNode(ctx context.Context) error {
	nodeInfo := m.nodeInfoProvider.GetNodeInfo(ctx)
	response, err := m.managementProxy.Register(ctx, legacy.RegisterRequest{
		Info: nodeInfo,
	})
	if err != nil {
		return errors.New("failed to register with requester node")
	}

	if response.Accepted {
		log.Ctx(ctx).Debug().Msg("register request accepted")
	} else {
		return fmt.Errorf("registration rejected: %s", response.Reason)
	}

	return nil
}

func (m *ManagementClient) deliverInfo(ctx context.Context) {
	// We _could_ avoid attempting an update if we are not registered, but
	// by doing so we will get frequent errors that the node is not
	// registered.

	nodeInfo := m.nodeInfoProvider.GetNodeInfo(ctx)
	response, err := m.managementProxy.UpdateInfo(ctx, legacy.UpdateInfoRequest{
		Info: nodeInfo,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to send update info to requester node")
		return
	}

	if response.Accepted {
		log.Ctx(ctx).Debug().Msg("update info accepted")
	} else {
		log.Ctx(ctx).Error().Msgf("update info rejected: %s", response.Reason)
	}
}

func (m *ManagementClient) updateResources(ctx context.Context) {
	nodeInfo := m.nodeInfoProvider.GetNodeInfo(ctx)
	request := legacy.UpdateResourcesRequest{
		NodeID:            nodeInfo.ID(),
		AvailableCapacity: nodeInfo.ComputeNodeInfo.AvailableCapacity,
		QueueUsedCapacity: nodeInfo.ComputeNodeInfo.QueueUsedCapacity,
	}
	log.Ctx(ctx).Trace().Msgf("Sending updated resources: %+v", request)

	_, err := m.managementProxy.UpdateResources(ctx, request)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to send resource update to requester node")
	}
}

func (m *ManagementClient) heartbeat(ctx context.Context, seq uint64) {
	if err := m.heartbeatClient.SendHeartbeat(ctx, seq); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("heartbeat failed sending sequence %d", seq)
	}
}

func (m *ManagementClient) Start(ctx context.Context) {
	infoTicker := time.NewTicker(m.heartbeatConfig.InfoUpdateInterval.AsTimeDuration())
	resourceTicker := time.NewTicker(m.heartbeatConfig.Interval.AsTimeDuration())

	// The heartbeat ticker will be used to send heartbeats to the requester node and
	// should be configured to be about half of the heartbeat frequency of the requester node
	// to ensure that the requester node does not consider this node as dead. If the server
	// heartbeat frequency is 30 seconds, the client heartbeat frequency should be configured to
	// fire more than once in that 30 seconds.
	heartbeatTicker := time.NewTicker(m.heartbeatConfig.Interval.AsTimeDuration())

	defer func() {
		heartbeatTicker.Stop()
		resourceTicker.Stop()
		infoTicker.Stop()

		// Close the heartbeat client and it's resources
		_ = m.heartbeatClient.Close(ctx)
	}()

	// Send an initial heartbeat when we start up
	var heartbeatSequence uint64 = 1
	m.heartbeat(ctx, heartbeatSequence)

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.done:
			return
		case <-infoTicker.C:
			// Send the latest node info to the requester node
			m.deliverInfo(ctx)
		case <-resourceTicker.C:
			// Send the latest resource info
			m.updateResources(ctx)
		case <-heartbeatTicker.C:
			// Send a heartbeat to the requester node
			heartbeatSequence += 1
			m.heartbeat(ctx, heartbeatSequence)
		}
	}
}

func (m *ManagementClient) Stop() {
	if m.done != nil {
		m.done <- struct{}{}
	}
}
