package compute

import (
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type Option func(*Node) error

func WithEngineProvider(p executor.ExecutorProvider) Option {
	return func(n *Node) error {
		n.EngineProvider = p
		return nil
	}
}

func WithPublisherProvider(p publisher.PublisherProvider) Option {
	return func(n *Node) error {
		n.PublisherProvider = p
		return nil
	}
}

func WithStorageProvider(p storage.StorageProvider) Option {
	return func(n *Node) error {
		n.StorageProvider = p
		return nil
	}
}

func WithCapacityProvider(p CapacityProvider) Option {
	return func(n *Node) error {
		n.CapacityProvider = p
		return nil
	}
}

func WithExecutorProvider(p ExecutorProvider) Option {
	return func(n *Node) error {
		n.ExecutorService = p
		return nil
	}
}

func WithEndpointProvider(p EndpointProvider) Option {
	return func(n *Node) error {
		n.EndpointProvider = p
		return nil
	}
}

func WithManagementClient(c *compute.ManagementClient) Option {
	return func(n *Node) error {
		n.ManagementClient = c
		return nil
	}
}

func WithLabelsProvider(p models.LabelsProvider) Option {
	return func(n *Node) error {
		n.LabelsProvider = p
		return nil
	}
}

func WithNodeInfoDecorator(d models.NodeInfoDecorator) Option {
	return func(n *Node) error {
		n.NodeInfoDecorator = d
		return nil
	}
}
