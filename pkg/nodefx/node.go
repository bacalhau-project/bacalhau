package nodefx

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/imdario/mergo"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/authn/ask"
	"github.com/bacalhau-project/bacalhau/pkg/authn/challenge"
	"github.com/bacalhau-project/bacalhau/pkg/authz"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/agent"
	auth_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/auth"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

type BacalhauNode struct {
	fx.In

	Transport        *nats_transport.NATSTransport
	Server           *Server
	NodeInfoProvider *routing.NodeInfoProvider
	Compute          *ComputeNode   `optional:"true"`
	Requester        *RequesterNode `optional:"true"`
}

func NewNode(ctx context.Context, cfg node.NodeConfig, ipfsClient ipfs.Client) (*BacalhauNode, func() error, error) {
	bacalhauNode := new(BacalhauNode)

	err := mergo.Merge(&cfg.APIServerConfig, publicapi.DefaultConfig())
	if err != nil {
		return nil, nil, err
	}

	app := fx.New(
		// TODO: create a client conditionally, and the right way using fx lifecycle
		fx.Supply(ipfsClient),
		// node config nats config, and IPFS config
		fx.Supply(cfg),
		fx.Supply(nats_transport.NATSTransportConfig{
			NodeID:                   cfg.NodeID,
			Port:                     cfg.NetworkConfig.Port,
			AdvertisedAddress:        cfg.NetworkConfig.AdvertisedAddress,
			AuthSecret:               cfg.NetworkConfig.AuthSecret,
			Orchestrators:            cfg.NetworkConfig.Orchestrators,
			StoreDir:                 cfg.NetworkConfig.StoreDir,
			ClusterName:              cfg.NetworkConfig.ClusterName,
			ClusterPort:              cfg.NetworkConfig.ClusterPort,
			ClusterPeers:             cfg.NetworkConfig.ClusterPeers,
			ClusterAdvertisedAddress: cfg.NetworkConfig.ClusterAdvertisedAddress,
			IsRequesterNode:          cfg.IsRequesterNode,
		}),
		fx.Provide(NATSS),

		// this is essentially the API module, needs a few more endpoints
		fx.Provide(Authorizer),
		fx.Provide(NewEchoRouter),
		fx.Provide(NewAPIServer),

		fx.Provide(NodeInfoProvider),

		SupplyIf(cfg.RequesterNodeConfig, cfg.IsRequesterNode),
		ProvideIf(Requester, cfg.IsRequesterNode),

		SupplyIf(cfg.ComputeConfig, cfg.IsComputeNode),
		ProvideIf(Compute, cfg.IsComputeNode),

		fx.Provide(AuthenticatorsProviders),
		fx.Populate(bacalhauNode),

		// TODO this needs the debug providers from the compute node and requester node
		fx.Invoke(agent.InitAgentEndpoint), // requires echo, nodeInfoProvider and DebugInforProviders

		// this is supplied as a string, which is only needed by InitSharedEndpoint
		fx.Provide(
			fx.Annotate(
				func() string {
					return cfg.NodeID
				},
				fx.ResultTags(`name:"nodeid"`),
			),
		),
		fx.Invoke(shared.InitSharedEndpoint), // requires nodeID and nodeInforProvider

		fx.Invoke(RegisterNodeInfoProviderDecorators),
		fx.Invoke(func(router *echo.Echo, provider authn.Provider) {
			auth_endpoint.BindEndpoint(context.TODO(), router, provider)
		}),
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
		bacalhauNode.NodeInfoProvider.RegisterNodeInfoDecorator(bacalhauNode.Compute.nodeInfoDecorator)
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

func Authorizer(cfg node.NodeConfig) (authz.Authorizer, error) {
	authzPolicy, err := policy.FromPathOrDefault(cfg.AuthConfig.AccessPolicyPath, authz.AlwaysAllowPolicy)
	if err != nil {
		return nil, err
	}

	signingKey, err := pkgconfig.GetClientPublicKey()
	if err != nil {
		return nil, err
	}
	return authz.NewPolicyAuthorizer(authzPolicy, signingKey, cfg.NodeID), nil
}

func ProvideIf(constructor func() fx.Option, condition bool) fx.Option {
	if condition {
		return constructor()
	}
	return fx.Options()
}

func SupplyIf(instance interface{}, condition bool) fx.Option {
	if condition {
		return fx.Supply(instance)
	}
	return fx.Options()
}

func PopulateIf[T any](instance *T, condition bool) fx.Option {
	if condition {
		fx.Populate(instance)
	}
	return fx.Options()
}

func NodeInfoProvider(cfg node.NodeConfig) (*routing.NodeInfoProvider, error) {
	// TODO this will miss any labels provided by the compute node I think
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

func AuthenticatorsProviders(cfg node.NodeConfig) (authn.Provider, error) {
	var allErr error
	privKey, allErr := pkgconfig.GetClientPrivateKey()
	if allErr != nil {
		return nil, allErr
	}

	authns := make(map[string]authn.Authenticator, len(cfg.AuthConfig.Methods))
	for name, authnConfig := range cfg.AuthConfig.Methods {
		switch authnConfig.Type {
		case authn.MethodTypeChallenge:
			methodPolicy, err := policy.FromPathOrDefault(authnConfig.PolicyPath, challenge.AnonymousModePolicy)
			if err != nil {
				allErr = errors.Join(allErr, err)
				continue
			}

			authns[name] = challenge.NewAuthenticator(
				methodPolicy,
				challenge.NewStringMarshaller(cfg.NodeID),
				privKey,
				cfg.NodeID,
			)
		case authn.MethodTypeAsk:
			methodPolicy, err := policy.FromPath(authnConfig.PolicyPath)
			if err != nil {
				allErr = errors.Join(allErr, err)
				continue
			}

			authns[name] = ask.NewAuthenticator(
				methodPolicy,
				privKey,
				cfg.NodeID,
			)
		default:
			allErr = errors.Join(allErr, fmt.Errorf("unknown authentication type: %q", authnConfig.Type))
		}
	}

	return provider.NewMappedProvider(authns), allErr
}
