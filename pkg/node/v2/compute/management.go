package compute

import (
	"fmt"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

const (
	HeartbeatTopic = "heartbeat"
)

func SetupNetworkClient(
	name string,
	r *repo.FsRepo,
	cfg v2.Heartbeat,
	transport *nats_transport.NATSTransport,
	capacityProvider CapacityProvider,
	labelsProvider models.LabelsProvider,
	decorator models.NodeInfoDecorator,
) (*compute.ManagementClient, error) {
	computePath, err := r.ComputePath()
	if err != nil {
		return nil, err
	}

	// TODO: Make the registration lock folder a config option so that we have it
	// available and don't have to depend on getting the repo folder.
	regFilename := fmt.Sprintf("%s.registration.lock", name)
	regFilename = filepath.Join(computePath, regFilename)

	heartbeatClient, err := heartbeat.NewClient(transport.Client(), name, HeartbeatTopic)
	if err != nil {
		return nil, fmt.Errorf("creating heartbeat client: %w", err)
	}

	// Set up the management client which will attempt to register this node
	// with the requester node, and then if successful will send regular node
	// info updates.
	return compute.NewManagementClient(&compute.ManagementClientParams{
		NodeID:                   name,
		LabelsProvider:           labelsProvider,
		ManagementProxy:          transport.ManagementProxy(),
		NodeInfoDecorator:        decorator,
		RegistrationFilePath:     regFilename,
		AvailableCapacityTracker: capacityProvider.RunningTracker(),
		QueueUsageTracker:        capacityProvider.QueuedTracker(),
		HeartbeatClient:          heartbeatClient,
		ControlPlaneSettings: types.ComputeControlPlaneConfig{
			InfoUpdateFrequency:     types.Duration(cfg.InfoInterval),
			ResourceUpdateFrequency: types.Duration(cfg.ResourceInterval),
			HeartbeatFrequency:      types.Duration(cfg.MessageInterval),
			HeartbeatTopic:          HeartbeatTopic,
		},
	}), nil
}
