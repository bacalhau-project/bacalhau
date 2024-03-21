package compute

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
)

const (
	infoUpdateFrequencyMinutes     = 5
	resourceUpdateFrequencySeconds = 30
)

type ManagementClientParams struct {
	NodeID               string
	LabelsProvider       models.LabelsProvider
	ManagementProxy      ManagementEndpoint
	NodeInfoDecorator    models.NodeInfoDecorator
	RegistrationFilePath string
	ResourceTracker      capacity.Tracker
}

// ManagementClient is used to call management functions with
// the requester nodes, via the NATS transport. When `Start`ed
// it will periodically send an update to the requester node with
// the latest node info for this node.
type ManagementClient struct {
	closeChannel      chan struct{}
	labelsProvider    models.LabelsProvider
	managementProxy   ManagementEndpoint
	nodeID            string
	nodeInfoDecorator models.NodeInfoDecorator
	registrationFile  *RegistrationFile
	resourceTracker   capacity.Tracker
}

func NewManagementClient(params ManagementClientParams) *ManagementClient {
	return &ManagementClient{
		closeChannel:      make(chan struct{}, 1),
		labelsProvider:    params.LabelsProvider,
		managementProxy:   params.ManagementProxy,
		nodeID:            params.NodeID,
		nodeInfoDecorator: params.NodeInfoDecorator,
		registrationFile:  NewRegistrationFile(params.RegistrationFilePath),
		resourceTracker:   params.ResourceTracker,
	}
}

func (m *ManagementClient) getNodeInfo(ctx context.Context) models.NodeInfo {
	return m.nodeInfoDecorator.DecorateNodeInfo(ctx, models.NodeInfo{
		NodeID:   m.nodeID,
		NodeType: models.NodeTypeCompute,
		Labels:   m.labelsProvider.GetLabels(ctx),
	})
}

// RegisterNode sends a registration request to the requester node. We will do this
// even if we know we have previously registered. This will allow us to not worry
// about split state (some local sentinel object and the remote state) and potential
// issues. The requester node will handle the case of a duplicate registration by
// accepting it but replying with a message to that effect.
func (m *ManagementClient) RegisterNode(ctx context.Context) error {
	nodeInfo := m.getNodeInfo(ctx)
	response, err := m.managementProxy.Register(ctx, requests.RegisterRequest{
		Info: nodeInfo,
	})
	if err != nil {
		return errors.New("failed to register with requester node")
	}

	if response.Accepted {
		log.Ctx(ctx).Debug().Msg("register request accepted")
	} else {
		// Might be an error, or might be rejected because it is in a pending
		// state instead
		log.Ctx(ctx).Error().Msgf("register request rejected: %s", response.Reason)
		return fmt.Errorf("registration rejected: %s", response.Reason)
	}

	return nil
}

func (m *ManagementClient) deliverInfo(ctx context.Context) {
	// We _could_ avoid attempting an update if we are not registered, but
	// by doing so we will get frequent errors that the node is not
	// registered.

	nodeInfo := m.getNodeInfo(ctx)
	response, err := m.managementProxy.UpdateInfo(ctx, requests.UpdateInfoRequest{
		Info: nodeInfo,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to send update info to requester node")
	}

	if response.Accepted {
		log.Ctx(ctx).Debug().Msg("update info accepted")
	} else {
		log.Ctx(ctx).Error().Msgf("update info rejected: %s", response.Reason)
	}
}

func (m *ManagementClient) updateResources(ctx context.Context) {
	log.Ctx(ctx).Debug().Msg("Sending updated resources")

	resources := m.resourceTracker.GetAvailableCapacity(ctx)
	_, err := m.managementProxy.UpdateResources(ctx, requests.UpdateResourcesRequest{
		NodeID:    m.nodeID,
		Resources: resources,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to send resource update to requester node")
	}
}

func (m *ManagementClient) Start(ctx context.Context) {
	infoTicker := time.NewTicker(infoUpdateFrequencyMinutes * time.Minute)
	resourceTicker := time.NewTicker(resourceUpdateFrequencySeconds * time.Second)

	loop := true
	for loop {
		select {
		case <-ctx.Done():
			loop = false
		case <-m.closeChannel:
			loop = false
		case <-infoTicker.C:
			// Send the latest node info to the requester node
			m.deliverInfo(ctx)
		case <-resourceTicker.C:
			// Send the latest resource info
			m.updateResources(ctx)
		}
	}

	resourceTicker.Stop()
	infoTicker.Stop()
}

func (m *ManagementClient) Stop() {
	if m.closeChannel != nil {
		m.closeChannel <- struct{}{}
	}
}
