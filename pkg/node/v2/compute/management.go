package compute

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

const (
	HeartbeatTopic = "heartbeat"
)

func SetupNetworkClient(
	name string,
	labels map[string]string,
	cfg v2.Heartbeat,
	transport *nats_transport.NATSTransport,
	engines executor.ExecutorProvider,
	storages storage.StorageProvider,
	publishers publisher.PublisherProvider,
	capacityProvider CapacityProvider,
	executor ExecutorProvider,
) (*compute.ManagementClient, error) {
	nodeInfoDecorator := compute.NewNodeInfoDecorator(compute.NodeInfoDecoratorParams{
		Executors:              engines,
		Publisher:              publishers,
		Storages:               storages,
		MaxJobRequirements:     capacityProvider.Capacity(),
		RunningCapacityTracker: capacityProvider.RunningTracker(),
		QueueCapacityTracker:   capacityProvider.QueuedTracker(),
		ExecutorBuffer:         executor.Executor(),
	})

	// TODO the compute store path needs to be a child of the repo path
	computeStorePath := filepath.Join("TODO", pkgconfig.ComputeStorePath)
	if err := os.MkdirAll(computeStorePath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating compute store directory: %s", err)
	}

	// TODO: Make the registration lock folder a config option so that we have it
	// available and don't have to depend on getting the repo folder.
	regFilename := fmt.Sprintf("%s.registration.lock", name)
	regFilename = filepath.Join(computeStorePath, regFilename)

	heartbeatClient, err := heartbeat.NewClient(transport.Client(), name, HeartbeatTopic)
	if err != nil {
		return nil, fmt.Errorf("creating heartbeat client: %w", err)
	}

	// Set up the management client which will attempt to register this node
	// with the requester node, and then if successful will send regular node
	// info updates.
	return compute.NewManagementClient(&compute.ManagementClientParams{
		NodeID: name,
		LabelsProvider: models.MergeLabelsInOrder(
			&node.ConfigLabelsProvider{StaticLabels: labels},
			&node.RuntimeLabelsProvider{},
			capacity.NewGPULabelsProvider(capacityProvider.Capacity()),
		),
		ManagementProxy:          transport.ManagementProxy(),
		NodeInfoDecorator:        nodeInfoDecorator,
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
