package v2

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	node2 "github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/node/v2/compute"
	"github.com/bacalhau-project/bacalhau/pkg/node/v2/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/agent"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

type Node struct {
	Transport    *nats_transport.NATSTransport
	Server       *publicapi.Server
	Compute      *compute.Node
	Orchestrator *orchestrator.Node
	Config       v2.Bacalhau
}

func (n *Node) Start(ctx context.Context) error {
	if n.Config.Orchestrator.Enabled {
		if err := n.Orchestrator.Start(ctx); err != nil {
			return fmt.Errorf("starting orcherstrator service: %w", err)
		}
	}
	if n.Config.Compute.Enabled {
		if err := n.Compute.Start(ctx); err != nil {
			return fmt.Errorf("starting compute service: %w", err)
		}
	}
	if err := n.Server.ListenAndServe(ctx); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	return nil
}

func (n *Node) Stop(ctx context.Context) error {
	var stopErr error
	if n.Config.Compute.Enabled {
		if err := n.Compute.Stop(ctx); err != nil {
			stopErr = errors.Join(stopErr, fmt.Errorf("stopping compute service: %w", err))
		}
	}
	if n.Config.Orchestrator.Enabled {
		if err := n.Orchestrator.Stop(ctx); err != nil {
			stopErr = errors.Join(stopErr, fmt.Errorf("stopping orchestrator service: %w", err))
		}
	}
	if err := n.Transport.Close(ctx); err != nil {
		stopErr = errors.Join(stopErr, fmt.Errorf("stopping transport service: %w", err))
	}
	if err := n.Server.Shutdown(ctx); err != nil {
		stopErr = errors.Join(stopErr, fmt.Errorf("shutting down server: %w", err))
	}
	return stopErr
}

// Should be simplified when moving to FX
func New(
	ctx context.Context,
	fsr *repo.FsRepo,
	cfg v2.Bacalhau,
) (*Node, error) {

	userKey, err := fsr.LoadUserKey()
	if err != nil {
		return nil, fmt.Errorf("loading user key from repo: %w", err)
	}
	server, err := SetupAPIServer(userKey.PublicKey(), cfg)
	if err != nil {
		return nil, fmt.Errorf("creating api server: %w", err)
	}

	transport, err := SetupTransport(cfg.Name, fsr, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating NATS transport: %w", err)
	}
	node := &Node{
		Transport: transport,
		Server:    server,
		Config:    cfg,
	}

	var debugInfoProviders []models.DebugInfoProvider
	debugInfoProviders = append(debugInfoProviders, transport.DebugInfoProviders()...)
	var labelsProvider models.LabelsProvider

	if cfg.Orchestrator.Enabled {
		authProvider, err := SetupAuthenticators(userKey.PrivateKey(), cfg)
		if err != nil {
			return nil, fmt.Errorf("setting up auth provider")
		}
		node.Orchestrator, err = orchestrator.SetupNode(ctx, cfg.Name, cfg.Orchestrator, fsr, server, transport, authProvider)
		if err != nil {
			return nil, fmt.Errorf("creating orchestrator service: %w", err)
		}
		labelsProvider = models.MergeLabelsInOrder(
			// TODO this seems wrong, but taken from existing code as found.
			&node2.ConfigLabelsProvider{StaticLabels: cfg.Compute.Labels},
			&node2.RuntimeLabelsProvider{},
		)
		debugInfoProviders = append(debugInfoProviders, node.Orchestrator.DebugInfoProvider...)
	}

	if cfg.Compute.Enabled {
		node.Compute, err = compute.SetupNode(ctx, fsr, server, transport, cfg.Name, cfg.Compute)
		if err != nil {
			return nil, fmt.Errorf("creating compute service: %w", err)
		}

		err = transport.RegisterComputeEndpoint(ctx, node.Compute.EndpointProvider.Bidding())
		if err != nil {
			return nil, err
		}

		labelsProvider = node.Compute.LabelsProvider
		debugInfoProviders = append(debugInfoProviders, node.Compute.ExecutorService.DebugProvider())
	}

	// TODO(forrest): understand this code

	// Create a node info provider for NATS, and specify the default node approval state
	// of Approved to avoid confusion as approval state is not used for this transport type.
	nodeInfoProvider := routing.NewNodeStateProvider(routing.NodeStateProviderParams{
		NodeID:              cfg.Name,
		LabelsProvider:      labelsProvider,
		BacalhauVersion:     *version.Get(),
		DefaultNodeApproval: models.NodeMembership.APPROVED,
	})
	nodeInfoProvider.RegisterNodeInfoDecorator(transport.NodeInfoDecorator())
	if cfg.Compute.Enabled {
		nodeInfoProvider.RegisterNodeInfoDecorator(node.Compute.NodeInfoDecorator)
	}

	shared.NewEndpoint(shared.EndpointParams{
		Router:            server.Router,
		NodeID:            cfg.Name,
		NodeStateProvider: nodeInfoProvider,
	})

	agent.NewEndpoint(agent.EndpointParams{
		Router:             server.Router,
		NodeStateProvider:  nodeInfoProvider,
		DebugInfoProviders: debugInfoProviders,
	})

	// We want to register the current requester node to the node store
	// TODO (walid): revisit self node registration of requester node
	if cfg.Orchestrator.Enabled {
		nodeState := nodeInfoProvider.GetNodeState(ctx)
		// TODO what is the liveness here? We are adding ourselves so I assume connected?
		nodeState.Membership = models.NodeMembership.APPROVED
		if err := node.Orchestrator.NodeInfoStore.Add(ctx, nodeState); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to add requester node to the node store")
			return nil, fmt.Errorf("registering node to the node store: %w", err)
		}
	}

	return node, nil
}
