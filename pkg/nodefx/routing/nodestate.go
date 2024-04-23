package routing

import (
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

var Module = fx.Module("routing",
	fx.Provide(NodeInfoProvider),
	fx.Invoke(RegisterNodeInfoProviderDecorators),
)

// TODO(forrest) [refactor]: the ideal case here would be to provide a labelsProvider and have a separate method
// to provide the NodeStateProvider that accepts the labels provider as input. The labelsProvider could be provided
// outside of this file, possibly in nodefx/node.go

func NodeInfoProvider(nodeID types.NodeID, c *config.Config) (*routing.NodeStateProvider, error) {
	var labels map[string]string
	if err := c.ForKey(types.NodeLabels, &labels); err != nil {
		return nil, err
	}
	// TODO(forest) [correctness]: this may miss any labels provided by the compute node if they are created dynamically
	labelsProvider := models.MergeLabelsInOrder(
		&node.ConfigLabelsProvider{StaticLabels: labels},
		&node.RuntimeLabelsProvider{},
	)
	nodeInfoProvider := routing.NewNodeStateProvider(routing.NodeStateProviderParams{
		NodeID:         string(nodeID),
		LabelsProvider: labelsProvider,
		// TODO(forrest) [refactor]: we can provide this
		BacalhauVersion:     *version.Get(),
		DefaultNodeApproval: models.NodeMembership.APPROVED,
	})
	return nodeInfoProvider, nil
}

func RegisterNodeInfoProviderDecorators(transport *nats_transport.NATSTransport, provider *routing.NodeStateProvider) {
	// TODO(forrest) [refactor]: this this PR needs to support libp2p then the transport layer will need to be an interface.
	provider.RegisterNodeInfoDecorator(transport.NodeInfoDecorator())
}
