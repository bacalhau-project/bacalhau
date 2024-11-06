package watchers

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

// Deprecated: BProtocolDispatcher implements the legacy protocol dispatcher.
// This implementation is maintained only for backward compatibility with v1.5 nodes.
// New implementations should use NCL protocol instead.
// TODO: Remove when v1.5 is no longer supported.
type BProtocolDispatcher struct {
	id             string
	computeService compute.Endpoint
	jobStore       jobstore.Store
}

type BProtocolDispatcherParams struct {
	ID             string
	ComputeService compute.Endpoint
	JobStore       jobstore.Store
}

func NewBProtocolDispatcher(params BProtocolDispatcherParams) *BProtocolDispatcher {
	return &BProtocolDispatcher{
		id:             params.ID,
		computeService: params.ComputeService,
		jobStore:       params.JobStore,
	}
}

func (d *BProtocolDispatcher) HandleEvent(ctx context.Context, event watcher.Event) error {
	upsert, ok := event.Object.(models.ExecutionUpsert)
	if !ok {
		return bacerrors.New("failed to process event: expected models.ExecutionUpsert, got %T", event.Object).
			WithComponent(bprotocolErrComponent)
	}

	// Skip if there's no state change
	if !upsert.HasStateChange() {
		return nil
	}

	transitions := newExecutionTransitions(upsert)
	execution := upsert.Current

	switch {
	case transitions.shouldAskForPendingBid():
		return d.handleAskForBid(ctx, execution, true)
	case transitions.shouldAskForDirectBid():
		return d.handleAskForBid(ctx, execution, false)
	case transitions.shouldAcceptBid():
		return d.handleBidAccepted(ctx, execution)
	case transitions.shouldRejectBid():
		return d.handleBidRejected(ctx, execution)
	case transitions.shouldCancel():
		return d.handleCancel(ctx, "", execution)
	}

	return nil
}

func (d *BProtocolDispatcher) handleAskForBid(ctx context.Context, execution *models.Execution, waitForApproval bool) error {
	log.Ctx(ctx).Debug().
		Str("nodeID", execution.NodeID).
		Str("executionID", execution.ID).
		Bool("waitForApproval", waitForApproval).
		Msg("Asking for bid")

	request := legacy.AskForBidRequest{
		Execution:       execution,
		WaitForApproval: waitForApproval,
		RoutingMetadata: legacy.RoutingMetadata{
			SourcePeerID: d.id,
			TargetPeerID: execution.NodeID,
		},
	}

	if _, err := d.computeService.AskForBid(ctx, request); err != nil {
		return bacerrors.Wrap(err, "failed to notify node %s to bid for execution %s",
			execution.NodeID, execution.ID).WithComponent(bprotocolErrComponent)
	}

	return nil
}

func (d *BProtocolDispatcher) handleBidAccepted(ctx context.Context, execution *models.Execution) error {
	log.Ctx(ctx).Debug().
		Str("nodeID", execution.NodeID).
		Str("executionID", execution.ID).
		Msg("Accepting bid")

	request := legacy.BidAcceptedRequest{
		ExecutionID: execution.ID,
		RoutingMetadata: legacy.RoutingMetadata{
			SourcePeerID: d.id,
			TargetPeerID: execution.NodeID,
		},
	}

	if _, err := d.computeService.BidAccepted(ctx, request); err != nil {
		return bacerrors.Wrap(err, "failed to notify node %s that bid %s was accepted",
			execution.NodeID, execution.ID).WithComponent(bprotocolErrComponent)
	}

	return nil
}

func (d *BProtocolDispatcher) handleBidRejected(ctx context.Context, execution *models.Execution) error {
	log.Ctx(ctx).Debug().
		Str("nodeID", execution.NodeID).
		Str("executionID", execution.ID).
		Msg("Rejecting bid")

	request := legacy.BidRejectedRequest{
		ExecutionID: execution.ID,
		RoutingMetadata: legacy.RoutingMetadata{
			SourcePeerID: d.id,
			TargetPeerID: execution.NodeID,
		},
	}

	if _, err := d.computeService.BidRejected(ctx, request); err != nil {
		return bacerrors.Wrap(err, "failed to notify node %s that bid %s was rejected",
			execution.NodeID, execution.ID).WithComponent(bprotocolErrComponent)
	}

	return nil
}

func (d *BProtocolDispatcher) handleCancel(ctx context.Context, message string, execution *models.Execution) error {
	log.Ctx(ctx).Debug().
		Str("nodeID", execution.NodeID).
		Str("executionID", execution.ID).
		Msg("Cancelling execution")

	request := legacy.CancelExecutionRequest{
		ExecutionID:   execution.ID,
		Justification: message,
		RoutingMetadata: legacy.RoutingMetadata{
			SourcePeerID: d.id,
			TargetPeerID: execution.NodeID,
		},
	}

	if _, err := d.computeService.CancelExecution(ctx, request); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to notify node %s that execution %s was canceled",
			execution.NodeID, execution.ID)
	}

	// Mark execution as cancelled
	if err := d.jobStore.UpdateExecution(ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: execution.ID,
		NewValues: models.Execution{
			ComputeState: models.State[models.ExecutionStateType]{
				StateType: models.ExecutionStateCancelled,
			},
		},
	}); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to mark execution %s as cancelled", execution.ID)
	}

	return nil
}
