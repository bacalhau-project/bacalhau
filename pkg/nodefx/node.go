package nodefx

import (
	"context"
	"fmt"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/authn/authnfx"
	"github.com/bacalhau-project/bacalhau/pkg/authz/authzfx"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/nodefx/compute"
	"github.com/bacalhau-project/bacalhau/pkg/nodefx/requester"
	"github.com/bacalhau-project/bacalhau/pkg/nodefx/routing"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/agent"
	auth_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/auth"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	routing2 "github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

type BacalhauNode struct {
	fx.In

	Repo             *repo.FsRepo
	Transport        *nats_transport.NATSTransport
	Server           *publicapi.Server
	NodeInfoProvider *routing2.NodeStateProvider
	Compute          *compute.ComputeNode     `optional:"true"`
	Requester        *requester.RequesterNode `optional:"true"`
}

func New(ctx context.Context, opts ...Option) (*BacalhauNode, func() error, error) {
	settings := &Settings{
		// TODO(forrest) [refactor]: idea here is to use an "in memory repo" as a default, then allow it to be overridden
		// an in memory repo would be useful for testing.
		repo:    nil,
		config:  config.New(),
		options: make(map[interface{}]fx.Option),
	}
	// TODO set default options on the settings, we need a default config for this that we will override.
	// in the below Options statement

	// apply module options
	if err := Options(opts...)(settings); err != nil {
		return nil, nil, fmt.Errorf("applying node options failed: %w", err)
	}

	// now we have a full set of options to build a node, if no opts... param was provided
	// these will all be the default, but if opts... were provided we will have overridden the
	// specified defaults.
	ctors := make([]fx.Option, 0, len(settings.options))
	for _, opt := range settings.options {
		ctors = append(ctors, opt)
	}
	// TODO for now repo is required. But we can provide a "Memrepo" implementation and make it a default
	// if a file system repo (fsrepo) isn't provided.
	if settings.repo == nil {
		panic("repo required")
	}

	bacalhauNode := new(BacalhauNode)
	app := fx.New(
		// ensure this never panics
		fx.RecoverFromPanics(),

		fx.Supply(log.Ctx(ctx)),
		fx.Supply(settings.repo),
		fx.Supply(settings.config),
		fx.Provide(NodeID),
		fx.Provide(NodeKind),

		// TODO(forrest) [refactor]: need an option here to either do NATS or Libp2p then decorate the returned type as TransportLayer
		// or we can simply deprecate libp2p now and avoid all this pain :)
		nats_transport.Module,
		routing.Module,
		authnfx.Module,
		authzfx.Module,
		publicapi.Module,

		// The Set of API endpoints that all nodes (compute and requester) share.
		fx.Module("common_api",
			fx.Invoke(agent.InitAgentEndpoint),
			fx.Invoke(shared.InitSharedEndpoint),
			fx.Invoke(func(router *echo.Echo, provider authn.Provider) {
				auth_endpoint.BindEndpoint(context.TODO(), router, provider)
			}),
		),

		// apply the specified options: compute, requester, and or ipfs.
		fx.Options(ctors...),

		// "build" the bacalhau node instance
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

	if err := bacalhauNode.Repo.WritePersistedConfigs(); err != nil {
		shutdown()
		return nil, nil, fmt.Errorf("error writing persisted config: %w", err)
	}

	return bacalhauNode, shutdown, nil
}

func NodeID(c *config.Config) (types.NodeID, error) {
	var nodeName types.NodeID
	if err := c.ForKey(types.NodeName, &nodeName); err != nil {
		return "", err
	}

	if nodeName != "" {
		return nodeName, nil
	}
	var nameProvider string
	if err := c.ForKey(types.NodeNameProvider, &nameProvider); err != nil {
		return "", err
	}

	nodeNameProviders := map[string]idgen.NodeNameProvider{
		"hostname": idgen.HostnameProvider{},
		"aws":      idgen.NewAWSNodeNameProvider(),
		"gcp":      idgen.NewGCPNodeNameProvider(),
		"uuid":     idgen.UUIDNodeNameProvider{},
		"puuid":    idgen.PUUIDNodeNameProvider{},
	}
	nodeNameProvider, ok := nodeNameProviders[nameProvider]
	if !ok {
		return "", fmt.Errorf(
			"unknown node name provider: %s. Supported providers are: %s", nameProvider, lo.Keys(nodeNameProviders))
	}

	name, err := nodeNameProvider.GenerateNodeName(context.TODO())
	if err != nil {
		return "", err
	}

	// set the new name in the config, so it can be used and persisted later.
	c.Set(types.NodeName, name)
	return types.NodeID(name), nil
}

func NodeKind(c *config.Config) (types.NodeKind, error) {
	var nodeType []string
	err := c.ForKey(types.NodeType, &nodeType)
	if err != nil {
		return types.NodeKind{}, err
	}
	var iscompute bool
	var isrequester bool
	for _, nodeType := range nodeType {
		if nodeType == "compute" {
			iscompute = true
		} else if nodeType == "requester" {
			isrequester = true
		} else {
			err = fmt.Errorf("invalid node type %s. Only compute and requester values are supported", nodeType)
		}
	}
	return types.NodeKind{
		Requester: isrequester,
		Compute:   iscompute,
	}, nil
}
