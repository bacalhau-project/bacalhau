package nodefx

import (
	"context"
	"fmt"
	"sync"

	"github.com/imdario/mergo"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/nodefx/auth"
	"github.com/bacalhau-project/bacalhau/pkg/nodefx/compute"
	"github.com/bacalhau-project/bacalhau/pkg/nodefx/requester"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/agent"
	auth_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/auth"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

type BacalhauNode struct {
	fx.In

	Transport        *nats_transport.NATSTransport
	Server           *publicapi.Server
	NodeInfoProvider *routing.NodeInfoProvider
	Compute          *compute.ComputeNode     `optional:"true"`
	Requester        *requester.RequesterNode `optional:"true"`
}

func getNodeID() (types.NodeID, error) {
	nodeName, err := pkgconfig.Get[string](types.NodeName)
	if err != nil {
		return "", err
	}

	if nodeName != "" {
		return types.NodeID(nodeName), nil
	}
	nodeNameProviderType, err := pkgconfig.Get[string](types.NodeNameProvider)
	if err != nil {
		return "", err
	}

	nodeNameProviders := map[string]idgen.NodeNameProvider{
		"hostname": idgen.HostnameProvider{},
		"aws":      idgen.NewAWSNodeNameProvider(),
		"gcp":      idgen.NewGCPNodeNameProvider(),
		"uuid":     idgen.UUIDNodeNameProvider{},
		"puuid":    idgen.PUUIDNodeNameProvider{},
	}
	nodeNameProvider, ok := nodeNameProviders[nodeNameProviderType]
	if !ok {
		return "", fmt.Errorf(
			"unknown node name provider: %s. Supported providers are: %s", nodeNameProviderType, lo.Keys(nodeNameProviders))
	}

	nodeName, err = nodeNameProvider.GenerateNodeName(context.TODO())
	if err != nil {
		return "", err
	}

	// set the new name in the config, so it can be used and persisted later.
	pkgconfig.SetValue(types.NodeName, nodeName)
	return types.NodeID(nodeName), nil
}

func NewNode(ctx context.Context, ndcfg node.NodeConfig, ipfsClient ipfs.Client) (*BacalhauNode, func() error, error) {
	bacalhauNode := new(BacalhauNode)

	err := mergo.Merge(&ndcfg.APIServerConfig, publicapi.DefaultConfig())
	if err != nil {
		return nil, nil, err
	}

	var nodeConfig types.NodeConfig
	if err := pkgconfig.ForKey(types.Node, &nodeConfig); err != nil {
		return nil, nil, err
	}

	log.Ctx(ctx)
	app := fx.New(
		fx.RecoverFromPanics(),

		fx.Supply(log.Ctx(ctx)),
		fx.Supply(nodeConfig.ServerMiddlewareConfig),
		fx.Supply(nodeConfig.Server),
		fx.Provide(getNodeID),
		fx.Module("ipfs",
			fx.Supply(ipfsClient),
		),

		fx.Module("config",
			fx.Supply(ndcfg),
			fx.Supply(nats_transport.NATSTransportConfig{
				NodeID:                   ndcfg.NodeID,
				Port:                     ndcfg.NetworkConfig.Port,
				AdvertisedAddress:        ndcfg.NetworkConfig.AdvertisedAddress,
				AuthSecret:               ndcfg.NetworkConfig.AuthSecret,
				Orchestrators:            ndcfg.NetworkConfig.Orchestrators,
				StoreDir:                 ndcfg.NetworkConfig.StoreDir,
				ClusterName:              ndcfg.NetworkConfig.ClusterName,
				ClusterPort:              ndcfg.NetworkConfig.ClusterPort,
				ClusterPeers:             ndcfg.NetworkConfig.ClusterPeers,
				ClusterAdvertisedAddress: ndcfg.NetworkConfig.ClusterAdvertisedAddress,
				IsRequesterNode:          ndcfg.IsRequesterNode,
			}),
			fx.Provide(
				fx.Annotate(
					func() string {
						return ndcfg.NodeID
					},
					// this is supplied as a string, which is wrong and only needed by InitSharedEndpoint
					fx.ResultTags(`name:"nodeid"`),
				),
			),
		),

		fx.Module("transport",
			fx.Provide(NATSS),
			fx.Provide(NodeInfoProvider),
			fx.Invoke(RegisterNodeInfoProviderDecorators),
		),

		auth.Module,
		publicapi.Module,
		fx.Module("api",
			fx.Invoke(agent.InitAgentEndpoint),
			fx.Invoke(shared.InitSharedEndpoint),
			fx.Invoke(func(router *echo.Echo, provider authn.Provider) {
				auth_endpoint.BindEndpoint(context.TODO(), router, provider)
			}),
		),

		ProvideIf(ndcfg.IsRequesterNode,
			requester.Module,
			fx.Supply(ndcfg.RequesterNodeConfig),
		),

		ProvideIf(ndcfg.IsComputeNode,
			compute.Module,
			compute.SupplyConfig(),
			fx.Supply(ndcfg.ComputeConfig),
		),

		fx.Populate(bacalhauNode),
	)

	// ensure the node was constructed as expected.
	if err := app.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to build bacalhau node: %w", err)
	}

	if bacalhauNode.Requester != nil {
		if err := bacalhauNode.Transport.RegisterComputeCallback(bacalhauNode.Requester.ComputeCallback); err != nil {
			return nil, nil, fmt.Errorf("registering requester node compute callback: %w", err)
		}
	}

	if bacalhauNode.Compute != nil {
		if err := bacalhauNode.Transport.RegisterComputeEndpoint(bacalhauNode.Compute.LocalEndpoint); err != nil {
			return nil, nil, fmt.Errorf("registering compute node endpoint: %w", err)
		}
		bacalhauNode.NodeInfoProvider.RegisterNodeInfoDecorator(bacalhauNode.Compute.NodeInfoDecorator)
	}

	var once sync.Once
	var stopErr error
	shutdown := func() error {
		once.Do(func() {
			stopErr = app.Stop(context.Background())
			if stopErr != nil {
				log.Error().Err(stopErr).Msg("failed to shutdown node")
			}
		})
		return stopErr
	}

	if err := app.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start bacalhau node: %w", err)
	}

	return bacalhauNode, shutdown, nil

}

func ProvideIf(condition bool, provider ...fx.Option) fx.Option {
	if condition {
		return fx.Options(provider...)
	}
	return fx.Options()
}

func NodeInfoProvider(cfg node.NodeConfig) (*routing.NodeInfoProvider, error) {
	// TODO this may miss any labels provided by the compute node if they are created dynamically
	labelsProvider := models.MergeLabelsInOrder(
		&node.ConfigLabelsProvider{StaticLabels: cfg.Labels},
		&node.RuntimeLabelsProvider{},
	)
	nodeInfoProvider := routing.NewNodeInfoProvider(routing.NodeInfoProviderParams{
		NodeID:              cfg.NodeID,
		LabelsProvider:      labelsProvider,
		BacalhauVersion:     *version.Get(),
		DefaultNodeApproval: models.NodeApprovals.APPROVED,
	})
	return nodeInfoProvider, nil
}

func RegisterNodeInfoProviderDecorators(transport *nats_transport.NATSTransport, provider *routing.NodeInfoProvider) {
	provider.RegisterNodeInfoDecorator(transport.NodeInfoDecorator())
}
