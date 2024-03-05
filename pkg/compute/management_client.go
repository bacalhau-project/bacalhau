package compute

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
	"github.com/rs/zerolog/log"
)

const (
	infoUpdateFrequencyMinutes = 5
)

type ManagementClientParams struct {
	NodeID            string
	LabelsProvider    models.LabelsProvider
	ManagementProxy   ManagementEndpoint
	NodeInfoDecorator models.NodeInfoDecorator
}

// ManagementClient is used to call management functions with
// the requester nodes, via the NATS transport.
type ManagementClient struct {
	closeChannel      chan struct{}
	labelsProvider    models.LabelsProvider
	managementProxy   ManagementEndpoint
	nodeID            string
	nodeInfoDecorator models.NodeInfoDecorator
	registrationFile  *RegistrationFile
}

func NewManagementClient(params ManagementClientParams) *ManagementClient {
	return &ManagementClient{
		closeChannel:      make(chan struct{}),
		labelsProvider:    params.LabelsProvider,
		managementProxy:   params.ManagementProxy,
		nodeID:            params.NodeID,
		nodeInfoDecorator: params.NodeInfoDecorator,
		registrationFile:  NewRegistrationFile(params.NodeID),
	}
}

func (m *ManagementClient) getNodeInfo(ctx context.Context) models.NodeInfo {
	return m.nodeInfoDecorator.DecorateNodeInfo(ctx, models.NodeInfo{
		NodeID:   m.nodeID,
		NodeType: models.NodeTypeCompute,
		Labels:   m.labelsProvider.GetLabels(ctx),
	})
}

func (m *ManagementClient) registerNode(ctx context.Context) {
	// We only want to register this node if we haven't already
	// been registered.
	if m.registrationFile.Exists() {
		log.Ctx(ctx).Debug().Msg("not registering with requester, already registered")
		return
	}

	nodeInfo := m.getNodeInfo(ctx)
	response, err := m.managementProxy.Register(ctx, requests.RegisterRequest{
		Info: nodeInfo,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to register with requester node")
		return
	}

	if response.Accepted {
		log.Ctx(ctx).Debug().Msg("register request accepted")
		if err := m.registrationFile.Set(); err != nil {
			log.Ctx(ctx).Error().Msgf("failed to record registration status")
		}
	} else {
		// Might be an error, or might be rejected because it is in a pending
		// state instead
		log.Ctx(ctx).Error().Msgf("register request rejected: %s", response.Error)
	}
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
		log.Ctx(ctx).Error().Msgf("update info rejected: %s", response.Error)
	}
}

func (m *ManagementClient) Start(ctx context.Context) {
	// Register the node with the requester node although we expect this
	// function to be a noop if we are already registered
	m.registerNode(ctx)

	infoTicker := time.NewTicker(infoUpdateFrequencyMinutes * time.Minute)

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
		}
	}

	infoTicker.Stop()
}

func (m *ManagementClient) Stop() {
	m.closeChannel <- struct{}{}
}
