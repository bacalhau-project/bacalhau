package transport

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
)

// TransportLayer is the interface for the transport layer.
type TransportLayer interface {
	// ComputeProxy enables orchestrator nodes to send job requests to compute nodes.
	ComputeProxy() compute.Endpoint
	// CallbackProxy enables compute nodes to send results and responses back to orchestrator nodes
	CallbackProxy() compute.Callback
	// NodeInfoPubSub enables compute nodes to publish their info and capabilities
	// to orchestrator nodes for job matching and discovery.
	NodeInfoPubSub() pubsub.PubSub[models.NodeInfo]
	// NodeInfoDecorator enables transport layer to enrich node info with data
	// required for request routing
	NodeInfoDecorator() models.NodeInfoDecorator
	// DebugInfoProviders enables transport layer to provide meaningful debug info to operators
	DebugInfoProviders() []model.DebugInfoProvider
	// RegisterComputeCallback registers a compute callback with the transport layer
	// so that incoming compute responses are forwarded to the handler
	RegisterComputeCallback(callback compute.Callback) error
	// RegisterComputeEndpoint registers a compute endpoint with the transport layer
	// so that incoming orchestrator requests are forwarded to the handler
	RegisterComputeEndpoint(endpoint compute.Endpoint) error
	// Close closes the transport layer.
	Close(ctx context.Context) error
}
