package watchers

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

// BProtocolDispatcher handles forwarding messages based on execution state changes
type BProtocolDispatcher struct {
	callback compute.Callback
}

// NewBProtocolDispatcher creates a new BProtocolDispatcher with the given callback
func NewBProtocolDispatcher(callback compute.Callback) *BProtocolDispatcher {
	return &BProtocolDispatcher{
		callback: callback,
	}
}

// HandleEvent processes a watcher event and publishes appropriate messages
func (d *BProtocolDispatcher) HandleEvent(ctx context.Context, event watcher.Event) error {
	upsert, ok := event.Object.(models.ExecutionUpsert)
	if !ok {
		return bacerrors.New("failed to process event: expected models.ExecutionUpsert, got %T", event.Object).
			WithComponent(bprotocolErrComponent)
	}

	execution := upsert.Current
	if execution.OrchestrationProtocol() != models.ProtocolBProtocolV2 {
		return nil
	}

	// Prepare base response with common fields
	routingMetadata := legacy.RoutingMetadata{
		// the source of this response is the bidders nodeID.
		SourcePeerID: execution.NodeID,
		// the target of this response is the source of the request.
		TargetPeerID: execution.Job.OrchestratorID(),
	}
	executionMetadata := legacy.NewExecutionMetadata(execution)
	updateEvent := models.Event{}
	if len(upsert.Events) > 0 {
		updateEvent = *upsert.Events[0]
	}

	// Create appropriate message based on execution state
	switch execution.ComputeState.StateType {
	case models.ExecutionStateAskForBidAccepted:
		log.Ctx(ctx).Debug().Msgf("Accepting bid for execution %s", execution.ID)
		d.callback.OnBidComplete(ctx, legacy.BidResult{
			RoutingMetadata:   routingMetadata,
			ExecutionMetadata: executionMetadata,
			Accepted:          true,
			Event:             updateEvent,
		})
	case models.ExecutionStateAskForBidRejected:
		log.Ctx(ctx).Debug().Msgf("Rejecting bid for execution %s", execution.ID)
		d.callback.OnBidComplete(ctx, legacy.BidResult{
			RoutingMetadata:   routingMetadata,
			ExecutionMetadata: executionMetadata,
			Accepted:          false,
			Event:             updateEvent,
		})
	case models.ExecutionStateBidAccepted:
		// Handle the case where bid is asked with pre-approval where compute state jumps to BidAccepted directly
		if upsert.Previous.ComputeState.StateType == models.ExecutionStateNew {
			log.Ctx(ctx).Debug().Msgf("Accepting and running execution %s", execution.ID)
			d.callback.OnBidComplete(ctx, legacy.BidResult{
				RoutingMetadata:   routingMetadata,
				ExecutionMetadata: executionMetadata,
				Accepted:          true,
				Event:             updateEvent,
			})
		}
	case models.ExecutionStateCompleted:
		log.Ctx(ctx).Debug().Msgf("Execution %s completed", execution.ID)
		d.callback.OnRunComplete(ctx, legacy.RunResult{
			RoutingMetadata:   routingMetadata,
			ExecutionMetadata: executionMetadata,
			PublishResult:     execution.PublishedResult,
			RunCommandResult:  execution.RunOutput,
		})
	case models.ExecutionStateFailed:
		log.Ctx(ctx).Debug().Msgf("Execution %s failed", execution.ID)
		d.callback.OnComputeFailure(ctx, legacy.ComputeError{
			RoutingMetadata:   routingMetadata,
			ExecutionMetadata: executionMetadata,
			Event:             updateEvent,
		})
	default:
		// No message created for other states
	}

	return nil
}
