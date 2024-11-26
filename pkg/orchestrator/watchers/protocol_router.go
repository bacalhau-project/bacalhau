package watchers

import (
	"context"
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

// ProtocolRouter routes commands from orchestrator to compute nodes using their
// supported protocols. It handles command dispatch for job execution, bidding,
// and cancellation.
type ProtocolRouter struct {
	nodeStore          routing.NodeInfoStore
	supportedProtocols map[models.Protocol]bool
}

// ProtocolRouterParams configures a new ProtocolRouter
type ProtocolRouterParams struct {
	NodeStore          routing.NodeInfoStore
	SupportedProtocols []models.Protocol
}

// NewProtocolRouter creates a new dispatcher that routes compute events
// to nodes using their preferred protocol
func NewProtocolRouter(params ProtocolRouterParams) (*ProtocolRouter, error) {
	err := errors.Join(
		validate.NotNil(params.NodeStore, "nodeStore cannot be nil"),
		validate.IsNotEmpty(params.SupportedProtocols, "at least one protocol handler is required"),
	)
	if err != nil {
		return nil, bacerrors.Wrap(err, "failed to create ProtocolRouter").WithComponent(protocolRouterErrComponent)
	}

	supportedProtocols := make(map[models.Protocol]bool)
	for _, protocol := range params.SupportedProtocols {
		supportedProtocols[protocol] = true
	}

	return &ProtocolRouter{
		nodeStore:          params.NodeStore,
		supportedProtocols: supportedProtocols,
	}, nil
}

// PreferredProtocol returns the protocol to use when dispatching an execution
func (d *ProtocolRouter) PreferredProtocol(ctx context.Context, execution *models.Execution) (models.Protocol, error) {
	nodeState, err := d.nodeStore.Get(ctx, execution.NodeID)
	if err != nil {
		return "", bacerrors.Wrap(err,
			"failed to get node info for node %s to determine routing protocol for execution %s",
			execution.NodeID, execution.ID).
			WithComponent(protocolRouterErrComponent)
	}

	// Filter protocols to only those that we have implementations for
	var matchingProtocols []models.Protocol
	for _, protocol := range nodeState.Info.SupportedProtocols {
		if _, ok := d.supportedProtocols[protocol]; ok {
			matchingProtocols = append(matchingProtocols, protocol)
		}
	}

	// Select the preferred protocol, falling back to legacy if necessary
	var preferredProtocol models.Protocol
	if len(matchingProtocols) == 0 {
		// Fall back to default protocol if no supported protocols are found
		// TODO: No longer assume default protocol when v1.5 is no longer supported
		preferredProtocol = models.ProtocolBProtocolV2
	} else {
		// Use protocol selection strategy to choose the best protocol
		preferredProtocol = models.GetPreferredProtocol(matchingProtocols)
	}

	// Validate protocol selection
	if preferredProtocol == "" {
		return "", bacerrors.New(
			"no supported protocol found for node %s (execution %s) - available: %s, supported: %s",
			execution.NodeID,
			execution.ID,
			nodeState.Info.SupportedProtocols,
			matchingProtocols).
			WithComponent(protocolRouterErrComponent)
	}

	return preferredProtocol, nil
}
