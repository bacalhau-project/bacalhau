package watchers

import (
	"context"
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

// Dispatcher routes commands from orchestrator to compute nodes using their
// supported protocols. It handles command dispatch for job execution, bidding,
// and cancellation.
type Dispatcher struct {
	nodeStore   routing.NodeInfoStore
	dispatchers map[models.Protocol]watcher.EventHandler
}

// DispatcherParams configures a new Dispatcher
type DispatcherParams struct {
	NodeStore   routing.NodeInfoStore
	Dispatchers map[models.Protocol]watcher.EventHandler
}

// NewDispatcher creates a new dispatcher that routes compute events
// to nodes using their preferred protocol
func NewDispatcher(params DispatcherParams) (*Dispatcher, error) {
	err := errors.Join(
		validate.NotNil(params.NodeStore, "nodeStore cannot be nil"),
		validate.IsGreaterThanZero(len(params.Dispatchers), "at least one protocol handler is required"),
	)
	for protocol, handler := range params.Dispatchers {
		err = errors.Join(err, validate.NotNil(handler, "dispatcher for protocol %s cannot be nil", protocol))
	}
	if err != nil {
		return nil, bacerrors.Wrap(err, "failed to create Dispatcher").WithComponent(dispatcherErrComponent)
	}

	return &Dispatcher{
		nodeStore:   params.NodeStore,
		dispatchers: params.Dispatchers,
	}, nil
}

// HandleEvent implements watcher.EventHandler
func (d *Dispatcher) HandleEvent(ctx context.Context, event watcher.Event) error {
	// Extract execution information from event
	upsert, ok := event.Object.(models.ExecutionUpsert)
	if !ok {
		return bacerrors.New("failed to process event: expected models.ExecutionUpsert, got %T", event.Object).
			WithComponent(dispatcherErrComponent)
	}

	execution := upsert.Current

	// Get node information to determine supported protocols
	nodeState, err := d.nodeStore.Get(ctx, execution.NodeID)
	if err != nil {
		return bacerrors.Wrap(err,
			"failed to get node info for node %s to determine routing protocol for execution %s",
			execution.NodeID, execution.ID).
			WithComponent(dispatcherErrComponent)
	}

	// Filter protocols to only those that we have implementations for
	var supportedProtocols []models.Protocol
	for _, protocol := range nodeState.Info.SupportedProtocols {
		if _, ok = d.dispatchers[protocol]; ok {
			supportedProtocols = append(supportedProtocols, protocol)
		}
	}

	// Select the preferred protocol, falling back to legacy if necessary
	var preferredProtocol models.Protocol
	if len(supportedProtocols) == 0 {
		// Fall back to default protocol if no supported protocols are found
		// TODO: No longer assume default protocol when v1.5 is no longer supported
		preferredProtocol = models.ProtocolBProtocolV2
	} else {
		// Use protocol selection strategy to choose the best protocol
		preferredProtocol = models.GetPreferredProtocol(supportedProtocols)
	}

	// Validate protocol selection
	if preferredProtocol == "" {
		return bacerrors.New(
			"no supported protocol found for node %s (execution %s) - available: %s, supported: %s",
			execution.NodeID,
			execution.ID,
			nodeState.Info.SupportedProtocols,
			supportedProtocols).
			WithComponent(dispatcherErrComponent)
	}

	// Dispatch the execution update using the selected protocol
	dispatcher := d.dispatchers[preferredProtocol]
	return dispatcher.HandleEvent(ctx, event)
}
