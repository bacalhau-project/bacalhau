package routing

import (
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/configfx"
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

func NodeInfoProvider(nodeID types.NodeID, c *configfx.Config) (*routing.NodeInfoProvider, error) {
	var labels map[string]string
	if err := c.ForKey(types.NodeLabels, &labels); err != nil {
		return nil, err
	}
	// TODO this may miss any labels provided by the compute node if they are created dynamically
	labelsProvider := models.MergeLabelsInOrder(
		&node.ConfigLabelsProvider{StaticLabels: labels},
		&node.RuntimeLabelsProvider{},
	)
	nodeInfoProvider := routing.NewNodeInfoProvider(routing.NodeInfoProviderParams{
		NodeID:         string(nodeID),
		LabelsProvider: labelsProvider,
		// TODO we can provide this
		BacalhauVersion:     *version.Get(),
		DefaultNodeApproval: models.NodeApprovals.APPROVED,
	})
	return nodeInfoProvider, nil
}

func RegisterNodeInfoProviderDecorators(transport *nats_transport.NATSTransport, provider *routing.NodeInfoProvider) {
	provider.RegisterNodeInfoDecorator(transport.NodeInfoDecorator())
}
