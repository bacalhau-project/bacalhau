package transport

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
)

// TransportLayer is the interface for the transport layer.
type TransportLayer interface {
	// ComputeProxy returns the compute proxy.
	ComputeProxy() compute.Endpoint
	// CallbackProxy returns the callback proxy.
	CallbackProxy() compute.Callback
	// NodeInfoPubSub returns the node info pubsub.
	NodeInfoPubSub() pubsub.PubSub[models.NodeInfo]
	// NodeInfoDecorator returns the node info decorator.
	NodeInfoDecorator() models.NodeInfoDecorator
	// RegisterComputeCallback registers a compute callback with the transport layer.
	RegisterComputeCallback(callback compute.Callback) error
	// RegisterComputeEndpoint registers a compute endpoint with the transport layer.
	RegisterComputeEndpoint(endpoint compute.Endpoint) error
	// Close closes the transport layer.
	Close(ctx context.Context) error
}
