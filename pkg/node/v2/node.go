package v2

import (
	"context"
	"fmt"

	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node/v2/compute"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

type Node struct {
	Transport *nats_transport.NATSTransport
	Server    *publicapi.Server
	Compute   *compute.Node
}

//nolint:funlen,gocyclo // Should be simplified when moving to FX
func New(
	ctx context.Context,
	cfg v2.Bacalhau,
	fsr *repo.FsRepo,
) (*Node, error) {

	server, err := SetupAPIServer("TODO", cfg)
	if err != nil {
		return nil, fmt.Errorf("creating api server: %w", err)
	}
	transport, err := SetupTransport(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating NATS transport: %w", err)
	}
	node := &Node{
		Transport: transport,
		Server:    server,
	}
	if cfg.Compute.Enabled {
		node.Compute, err = compute.SetupNode(ctx, fsr, server, transport, cfg.Name, cfg.Compute)
		if err != nil {
			return nil, fmt.Errorf("creating compute node: %w", err)
		}
	}

	return node, nil
}
